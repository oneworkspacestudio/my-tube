package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type identity struct {
	User User
	CSRF string
	Hash string
}
type contextKey struct{}

func who(r *http.Request) identity { id, _ := r.Context().Value(contextKey{}).(identity); return id }

type authProvider struct {
	mu       sync.Mutex
	provider *oidc.Provider
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func (a *App) provider(ctx context.Context) (*authProvider, error) {
	p := &a.oidc
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.provider != nil {
		return p, nil
	}
	if a.Config.Issuer == "" || a.Config.ClientID == "" {
		return nil, errors.New("MyAccounts is not configured")
	}
	u, e := url.Parse(a.Config.Issuer)
	if e != nil || u.Host == "" {
		return nil, errors.New("invalid issuer")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return nil, errors.New("issuer must use HTTPS")
		}
	}
	provider, e := oidc.NewProvider(ctx, a.Config.Issuer)
	if e != nil {
		return nil, e
	}
	p.config = oauth2.Config{ClientID: a.Config.ClientID, ClientSecret: a.Config.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: a.Config.PublicURL + "/auth/callback", Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}
	p.verifier = provider.Verifier(&oidc.Config{ClientID: a.Config.ClientID})
	p.provider = provider
	return p, nil
}
func (a *App) cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: a.cookiePath(), HttpOnly: true, Secure: strings.HasPrefix(a.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (a *App) cookiePath() string {
	if a.Config.BasePath == "" {
		return "/"
	}
	return a.Config.BasePath + "/"
}
func (a *App) newSession(w http.ResponseWriter, u User) error {
	if _, e := a.Store.DB.Exec("DELETE FROM sessions WHERE expires<?", now()); e != nil {
		return e
	}
	token, csrf := secret(), secret()
	expires := time.Now().Add(time.Duration(a.Config.SessionHours) * time.Hour).UnixMilli()
	_, e := a.Store.DB.Exec("INSERT INTO sessions(hash,user_id,csrf,expires)VALUES(?,?,?,?)", digest(token), u.ID, csrf, expires)
	if e != nil {
		return e
	}
	a.cookie(w, "mytube_session", token, a.Config.SessionHours*3600)
	return nil
}
func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	if a.Config.Mode != "managed" {
		problem(w, 400, "Use the local sign-in link printed by MyTube.exe")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	p, e := a.provider(ctx)
	if e != nil {
		problem(w, 503, "MyAccounts is unavailable or not configured. Existing sessions retain their original expiry.")
		return
	}
	state, nonce, verifier, binding := secret(), secret(), oauth2.GenerateVerifier(), secret()
	_, e = a.Store.DB.Exec("DELETE FROM logins WHERE expires<?", now())
	if e != nil {
		problem(w, 500, "Could not begin login")
		return
	}
	_, e = a.Store.DB.Exec("INSERT INTO logins VALUES(?,?,?,?,?)", digest(state), digest(binding), nonce, verifier, time.Now().Add(10*time.Minute).UnixMilli())
	if e != nil {
		problem(w, 500, "Could not begin login")
		return
	}
	a.cookie(w, "mytube_login", binding, 600)
	http.Redirect(w, r, p.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce)), http.StatusFound)
}
func (a *App) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	cookie, e := r.Cookie("mytube_login")
	if e != nil || state == "" {
		problem(w, 400, "Invalid login state")
		return
	}
	var binding, nonce, verifier string
	var expires int64
	e = a.Store.DB.QueryRow("DELETE FROM logins WHERE state=? RETURNING binding,nonce,verifier,expires", digest(state)).Scan(&binding, &nonce, &verifier, &expires)
	a.cookie(w, "mytube_login", "", -1)
	if e != nil || expires < now() || subtle.ConstantTimeCompare([]byte(binding), []byte(digest(cookie.Value))) != 1 {
		problem(w, 400, "Invalid or expired login state")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	p, e := a.provider(ctx)
	if e != nil {
		problem(w, 503, "MyAccounts is unavailable")
		return
	}
	token, e := p.config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if e != nil {
		problem(w, 401, "Authorization code rejected")
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		problem(w, 401, "Missing identity token")
		return
	}
	idt, e := p.verifier.Verify(ctx, raw)
	if e != nil || idt.Nonce != nonce || idt.Subject == "" {
		problem(w, 401, "Invalid identity token")
		return
	}
	var claims struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if e = idt.Claims(&claims); e != nil {
		problem(w, 401, "Invalid identity claims")
		return
	}
	u, e := a.Store.Provision(idt.Issuer, idt.Subject, claims.Name, claims.Email, claims.Picture, a.Config.QuotaMB)
	if e != nil {
		problem(w, 403, "Account unavailable")
		return
	}
	if e = a.newSession(w, u); e != nil {
		problem(w, 500, "Could not create session")
		return
	}
	a.Store.Audit(u.ID, "sign-in", u.ID)
	http.Redirect(w, r, a.cookiePath(), http.StatusSeeOther)
}
func (a *App) LocalLogin(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	a.localMu.Lock()
	defer a.localMu.Unlock()
	token := r.URL.Query().Get("token")
	if (a.Config.Mode != "local" && !a.Config.Setup) || ip == nil || !ip.IsLoopback() || a.localToken == "" || subtle.ConstantTimeCompare([]byte(digest(token)), []byte(a.localToken)) != 1 {
		problem(w, 401, "Invalid local sign-in link")
		return
	}
	u, e := a.Store.Provision("urn:oneworkspace:mytube:local", "owner", "Personal workspace", "", "", a.Config.QuotaMB)
	if e != nil {
		problem(w, 403, "Local account unavailable")
		return
	}
	if e = a.newSession(w, u); e != nil {
		problem(w, 500, "Could not create session")
		return
	}
	a.localToken = ""
	http.Redirect(w, r, a.cookiePath(), http.StatusSeeOther)
}
func (a *App) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie("mytube_session")
		if e != nil {
			problem(w, 401, "Sign in to continue")
			return
		}
		id := identity{Hash: digest(c.Value)}
		var issuer string
		var disabled bool
		e = a.Store.DB.QueryRow(`SELECT u.id,u.name,u.email,u.avatar,u.subject,u.issuer,u.disabled,s.csrf FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.hash=? AND s.expires>?`, id.Hash, now()).Scan(&id.User.ID, &id.User.Name, &id.User.Email, &id.User.Avatar, &id.User.Subject, &issuer, &disabled, &id.CSRF)
		if e != nil || disabled {
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				a.Logger.Print("session lookup failed")
			}
			problem(w, 401, "Session expired; sign in again")
			return
		}
		id.User.Admin = (a.Config.Mode == "local" || a.Config.Setup) && issuer == "urn:oneworkspace:mytube:local"
		for _, sub := range a.Config.AdminSubjects {
			if sub == id.User.Subject && issuer == a.Config.Issuer {
				id.User.Admin = true
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			u, _ := url.Parse(a.Config.PublicURL)
			origin := u.Scheme + "://" + u.Host
			if r.Header.Get("Origin") != origin || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(id.CSRF)) != 1 {
				problem(w, 403, "Request origin or CSRF token rejected")
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			a.writeMu.Lock()
			defer a.writeMu.Unlock()
		}
		next(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, id)))
	}
}
