package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) original(v Video) string {
	return filepath.Join(a.Config.MediaDir, "originals", v.Original)
}
func admin(w http.ResponseWriter, r *http.Request) bool {
	if !who(r).User.Admin {
		problem(w, 403, "Administrator permission required")
		return false
	}
	return true
}
func (a *App) Admin(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	type user struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Subject  string `json:"subject"`
		Disabled bool   `json:"disabled"`
		Quota    int64  `json:"quota_mb"`
	}
	users := []user{}
	rows, e := a.Store.DB.Query("SELECT id,name,subject,disabled,quota_mb FROM users")
	if e != nil {
		a.failure(w, e)
		return
	}
	for rows.Next() {
		var u user
		if rows.Scan(&u.ID, &u.Name, &u.Subject, &u.Disabled, &u.Quota) == nil {
			users = append(users, u)
		}
	}
	rows.Close()
	jobs := []map[string]any{}
	rows, _ = a.Store.DB.Query("SELECT j.id,j.video_id,j.status,j.attempts,j.error,j.updated FROM jobs j ORDER BY updated DESC LIMIT 100")
	if rows != nil {
		for rows.Next() {
			var id, vid, status, err string
			var attempts int
			var updated int64
			if rows.Scan(&id, &vid, &status, &attempts, &err, &updated) == nil {
				jobs = append(jobs, map[string]any{"id": id, "video_id": vid, "status": status, "attempts": attempts, "error": err, "updated": updated})
			}
		}
		rows.Close()
	}
	reports := []map[string]any{}
	rows, _ = a.Store.DB.Query("SELECT id,video_id,reason,status,created FROM reports ORDER BY created DESC LIMIT 100")
	if rows != nil {
		for rows.Next() {
			var id, vid, reason, status string
			var created int64
			if rows.Scan(&id, &vid, &reason, &status, &created) == nil {
				reports = append(reports, map[string]any{"id": id, "video_id": vid, "reason": reason, "status": status, "created": created})
			}
		}
		rows.Close()
	}
	logs := []map[string]any{}
	rows, _ = a.Store.DB.Query("SELECT at,actor,event,target FROM audit ORDER BY id DESC LIMIT 100")
	if rows != nil {
		for rows.Next() {
			var at int64
			var actor, event, target string
			if rows.Scan(&at, &actor, &event, &target) == nil {
				logs = append(logs, map[string]any{"at": at, "actor": actor, "event": event, "target": target})
			}
		}
		rows.Close()
	}
	used, e := treeSize(a.Config.MediaDir)
	health := "ok"
	if e != nil {
		health = e.Error()
	}
	respond(w, 200, map[string]any{"config": a.Config, "users": users, "jobs": jobs, "reports": reports, "logs": logs, "media_bytes": used, "storage_health": health, "version": Version, "release_url": "https://github.com/oneworkspacestudio/my-tube/releases", "restart_required_for_config": true})
}
func (a *App) AdminConfig(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	c := a.Config
	if !decode(w, r, &c) {
		return
	}
	c.ClientSecret = a.Config.ClientSecret
	c.ConfigPath = a.Config.ConfigPath
	c.Setup = false
	if c.Mode == "managed" && (c.Issuer == "" || c.ClientID == "") {
		problem(w, 400, "Managed mode needs MyAccounts issuer and client ID")
		return
	}
	if e := c.Validate(); e != nil {
		problem(w, 400, e.Error())
		return
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	if c.ConfigPath == "" {
		problem(w, 500, "Config path is not set")
		return
	}
	if e := atomicWrite(c.ConfigPath, b); e != nil {
		problem(w, 500, "Could not save config")
		return
	}
	a.Store.Audit(who(r).User.ID, "config.save", "restart-required")
	respond(w, 200, map[string]any{"ok": true, "restart_required": true})
}
func (a *App) AdminUser(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	var in struct {
		Disabled bool  `json:"disabled"`
		Quota    int64 `json:"quota_mb"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Quota < 1 || in.Quota > a.Config.StorageMB || r.PathValue("id") == who(r).User.ID && in.Disabled {
		problem(w, 400, "Invalid quota or cannot disable own account")
		return
	}
	_, e := a.Store.DB.Exec("UPDATE users SET disabled=?,quota_mb=? WHERE id=?", in.Disabled, in.Quota, r.PathValue("id"))
	if in.Disabled {
		a.Store.DB.Exec("DELETE FROM sessions WHERE user_id=?", r.PathValue("id"))
	}
	a.Store.Audit(who(r).User.ID, "user.update", r.PathValue("id"))
	a.failure(w, e)
}
func (a *App) ResolveReport(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	var in struct {
		Status string `json:"status"`
		Remove bool   `json:"remove"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Status != "resolved" && in.Status != "open" {
		problem(w, 400, "Invalid report status")
		return
	}
	var vid string
	if a.Store.DB.QueryRow("SELECT video_id FROM reports WHERE id=?", r.PathValue("id")).Scan(&vid) != nil {
		problem(w, 404, "Report unavailable")
		return
	}
	if in.Remove {
		a.Store.DB.Exec("UPDATE videos SET deleted=1,visibility='private' WHERE id=?", vid)
	}
	_, e := a.Store.DB.Exec("UPDATE reports SET status=? WHERE id=?", in.Status, r.PathValue("id"))
	a.Store.Audit(who(r).User.ID, "report.moderate", vid)
	a.failure(w, e)
}

// Reconciliation is explicit and never removes live originals due to failed jobs.
func (a *App) Reconcile(w http.ResponseWriter, r *http.Request) {
	if !admin(w, r) {
		return
	}
	var in struct {
		PurgeDeleted bool `json:"purge_deleted"`
	}
	if !decode(w, r, &in) {
		return
	}
	type entry struct {
		id, original string
		deleted      bool
	}
	all := []entry{}
	rows, e := a.Store.DB.Query("SELECT id,original,deleted FROM videos")
	if e != nil {
		a.failure(w, e)
		return
	}
	for rows.Next() {
		var v entry
		rows.Scan(&v.id, &v.original, &v.deleted)
		all = append(all, v)
	}
	rows.Close()
	missing := []string{}
	purged := 0
	valid := map[string]bool{}
	for _, v := range all {
		valid[v.id] = true
		if _, e := os.Stat(filepath.Join(a.Config.MediaDir, "originals", v.original)); os.IsNotExist(e) {
			missing = append(missing, v.id)
			a.Store.DB.Exec("UPDATE videos SET status='missing',error='Original missing; restore from backup' WHERE id=?", v.id)
		}
		if v.deleted && in.PurgeDeleted {
			var running int
			a.Store.DB.QueryRow("SELECT count(*) FROM jobs WHERE video_id=? AND status='running'", v.id).Scan(&running)
			if running > 0 {
				continue
			}
			p := filepath.Join(a.Config.MediaDir, "originals", v.original)
			if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
				problem(w, 500, "Cannot purge original")
				return
			}
			os.RemoveAll(filepath.Join(a.Config.MediaDir, "streaming", v.id))
			os.Remove(filepath.Join(a.Config.MediaDir, "thumbnails", v.id+".jpg"))
			os.Remove(filepath.Join(a.Config.MediaDir, "subtitles", v.id+".vtt"))
			a.Store.DB.Exec("DELETE FROM uploads WHERE video_id=?", v.id)
			if _, e = a.Store.DB.Exec("DELETE FROM videos WHERE id=?", v.id); e != nil {
				a.failure(w, e)
				return
			}
			purged++
			delete(valid, v.id)
		}
	}
	files, _ := os.ReadDir(filepath.Join(a.Config.MediaDir, "thumbnails"))
	for _, f := range files {
		id := strings.TrimSuffix(f.Name(), ".jpg")
		if !valid[id] && !strings.Contains(id, "-") {
			os.Remove(filepath.Join(a.Config.MediaDir, "thumbnails", f.Name()))
		}
	}
	a.Store.Audit(who(r).User.ID, "storage.reconcile", fmt.Sprintf("purged:%d", purged))
	respond(w, 200, map[string]any{"missing": missing, "purged": purged})
}
