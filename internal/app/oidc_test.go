package app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A real discovery/token/JWKS exchange with signed ID tokens. Live MyAccounts
// deployment acceptance is separately recorded; this fixture tests the RP boundary.
func TestOIDCPKCESignatureNonceStateAndReplay(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "test-key"))
	if e != nil {
		t.Fatal(e)
	}
	var issuer, nonce, challenge, tokenNonce, audience string
	var badSignature, badIssuer, expired bool
	audience = "mytube-test"
	mux := http.NewServeMux()
	provider := httptest.NewServer(mux)
	defer provider.Close()
	issuer = provider.URL
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "code_challenge_methods_supported": []string{"S256"}})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		if e := r.ParseForm(); e != nil {
			t.Error(e)
		}
		hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(hash[:]) != challenge || r.Form.Get("code") != "good-code" {
			problem(w, 400, "invalid grant")
			return
		}
		n := nonce
		if tokenNonce != "" {
			n = tokenNonce
		}
		claimIssuer := issuer
		expires := time.Now().Add(time.Hour).Unix()
		if badIssuer {
			claimIssuer = "https://invalid.example"
		}
		if expired {
			expires = time.Now().Add(-time.Hour).Unix()
		}
		raw, e := jwt.Signed(signer).Claims(map[string]any{"iss": claimIssuer, "sub": "stable-sub", "aud": audience, "exp": expires, "iat": time.Now().Unix(), "nonce": n, "name": "Test Alice", "email": "alice@example.test"}).Serialize()
		if e != nil {
			t.Error(e)
		}
		if badSignature {
			parts := strings.Split(raw, ".")
			sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
			sig[0] ^= 0xff
			parts[2] = base64.RawURLEncoding.EncodeToString(sig)
			raw = strings.Join(parts, ".")
		}
		respond(w, 200, map[string]any{"access_token": "test-access", "token_type": "Bearer", "expires_in": 3600, "id_token": raw})
	})
	a := testApp(t)
	a.Config.Issuer = issuer
	a.Config.ClientID = "mytube-test"
	a.Config.ClientSecret = "test-secret"
	begin := func() (string, *http.Cookie) {
		r := httptest.NewRequest("GET", a.Config.PublicURL+"/auth/login", nil)
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		mustStatus(t, w, 302)
		u, e := url.Parse(w.Header().Get("Location"))
		if e != nil {
			t.Fatal(e)
		}
		q := u.Query()
		if q.Get("code_challenge_method") != "S256" || q.Get("redirect_uri") != a.Config.PublicURL+"/auth/callback" {
			t.Fatal(q)
		}
		nonce = q.Get("nonce")
		challenge = q.Get("code_challenge")
		return q.Get("state"), w.Result().Cookies()[0]
	}
	callback := func(state string, c *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", a.Config.PublicURL+"/auth/callback?state="+url.QueryEscape(state)+"&code=good-code", nil)
		r.AddCookie(c)
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	state, c := begin()
	mustStatus(t, callback(state+"tamper", c), 400)
	w := callback(state, c)
	mustStatus(t, w, 303)
	mustStatus(t, callback(state, c), 400)
	var session *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "mytube_session" {
			session = cookie
		}
	}
	if session == nil {
		t.Fatal("no session cookie")
	}
	if !session.HttpOnly || session.SameSite != http.SameSiteLaxMode {
		t.Fatal(session)
	}
	r := httptest.NewRequest("GET", a.Config.PublicURL+"/api/me", nil)
	r.AddCookie(session)
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 200)
	var data map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &data)
	firstID := data["user"].(map[string]any)["id"]
	state, c = begin()
	tokenNonce = "wrong-nonce"
	mustStatus(t, callback(state, c), 401)
	tokenNonce = ""
	state, c = begin()
	audience = "wrong-audience"
	mustStatus(t, callback(state, c), 401)
	audience = "mytube-test"
	for _, bad := range []*bool{&badSignature, &badIssuer, &expired} {
		*bad = true
		state, c = begin()
		mustStatus(t, callback(state, c), 401)
		*bad = false
	}
	state, c = begin()
	w = callback(state, c)
	mustStatus(t, w, 303)
	var id string
	if e = a.Store.DB.QueryRow("SELECT id FROM users WHERE issuer=? AND subject=?", issuer, "stable-sub").Scan(&id); e != nil || id != firstID {
		t.Fatal("unstable identity", id, firstID, e)
	}
}
