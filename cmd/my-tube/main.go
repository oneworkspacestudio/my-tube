package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/kardianos/service"
	"github.com/oneworkspacestudio/my-tube/internal/app"
	"github.com/oneworkspacestudio/my-tube/web"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type program struct {
	config      app.Config
	server      *http.Server
	application *app.App
	lock        *flock.Flock
	done        chan struct{}
	noBrowser   bool
}

func (p *program) Start(s service.Service) error {
	if e := app.CheckDataDirectory(p.config.DataDir); e != nil {
		return e
	}
	if e := app.ProtectData(p.config.DataDir, p.config.Mode == "managed"); e != nil {
		return e
	}
	if e := os.MkdirAll(p.config.DataDir, 0700); e != nil {
		return e
	}
	if e := app.CheckMediaDirectory(p.config.MediaDir); e != nil {
		return e
	}
	if e := app.ProtectData(p.config.MediaDir, p.config.Mode == "managed"); e != nil {
		return e
	}
	p.lock = flock.New(filepath.Join(p.config.DataDir, "mytube.lock"))
	ok, e := p.lock.TryLock()
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("mytube is already running or a backup/restore is active")
	}
	assets, e := fs.Sub(web.Assets, "dist")
	if e != nil {
		p.lock.Unlock()
		return e
	}
	p.application, e = app.New(p.config, assets, nil)
	if e != nil {
		p.lock.Unlock()
		return e
	}
	listener, e := net.Listen("tcp", p.config.Listen)
	if e != nil {
		p.application.Close()
		p.lock.Unlock()
		return e
	}
	p.application.StartWorkers()
	p.server = &http.Server{Handler: p.application.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 0, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		if e := p.server.Serve(listener); e != nil && e != http.ErrServerClosed {
			log.Printf("HTTP server stopped: %v", e)
		}
	}()
	log.Printf("mytube %s · %s", app.Version, p.config.PublicURL)
	openURL := p.config.PublicURL
	if p.config.Mode == "local" || p.config.Setup {
		openURL = p.application.LocalURL()
		fmt.Println("One-time local sign-in: " + openURL)
	}
	if !p.noBrowser && service.Interactive() && runtime.GOOS == "windows" {
		cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", openURL)
		if cmd.Start() == nil {
			go cmd.Wait()
		}
	}
	return nil
}
func (p *program) Stop(s service.Service) error {
	if p.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	e := p.server.Shutdown(ctx)
	if e != nil {
		_ = p.server.Close()
	}
	<-p.done
	ce := p.application.Close()
	_ = p.lock.Unlock()
	if e != nil {
		return e
	}
	return ce
}
func main() {
	exe, e := os.Executable()
	if e != nil {
		log.Fatal(e)
	}
	defaultConfig := filepath.Join(filepath.Dir(exe), "config.json")
	config := flag.String("config", defaultConfig, "configuration JSON path")
	local := flag.Bool("local", false, "explicit loopback-only single-user mode with one-time sign-in link")
	data := flag.String("data", "", "portable data directory")
	media := flag.String("media", "", "dedicated media directory")
	noBrowser := flag.Bool("no-browser", false, "do not open a browser at interactive startup")
	port := flag.String("port", "", "override loopback HTTP port")
	public := flag.String("public-url", "", "browser-facing URL, including any /mytube prefix")
	file := flag.String("file", "", "backup input/output ZIP path")
	flag.Parse()
	command := "run"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}
	if command == "version" {
		fmt.Println(app.Version)
		return
	}
	c, e := app.LoadConfig(*config)
	if e != nil {
		log.Fatal(e)
	}
	if *local {
		c.Mode = "local"
		c.Setup = false
	}
	if *data != "" {
		if filepath.Clean(c.MediaDir) == filepath.Join(filepath.Clean(c.DataDir), "media") {
			c.MediaDir = filepath.Join(*data, "media")
		}
		c.DataDir = *data
	}
	if *media != "" {
		c.MediaDir = *media
	}
	if *port != "" {
		c.Listen = net.JoinHostPort("127.0.0.1", *port)
		c.PublicURL = "http://" + c.Listen
	}
	if *public != "" {
		c.PublicURL = *public
	} else if c.RedirectURI != "" {
		c.PublicURL = strings.TrimSuffix(c.RedirectURI, "/auth/callback")
	}
	if e = c.Validate(); e != nil {
		log.Fatal(e)
	}
	switch command {
	case "config-check":
		fmt.Println("Configuration valid")
		return
	case "health":
		client := http.Client{Timeout: 5 * time.Second}
		response, err := client.Get("http://" + c.Listen + "/healthz")
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			log.Fatal("Unhealthy")
		}
		fmt.Println("healthy")
		return
	case "migrate":
		if e = app.CheckDataDirectory(c.DataDir); e != nil {
			log.Fatal(e)
		}
		if e = app.ProtectData(c.DataDir, c.Mode == "managed"); e != nil {
			log.Fatal(e)
		}
		lock := flock.New(filepath.Join(c.DataDir, "mytube.lock"))
		ok, err := lock.TryLock()
		if err != nil || !ok {
			log.Fatal("Stop My Tube before migrations")
		}
		s, err := app.OpenStore(c.DataDir)
		if err == nil {
			err = s.Close()
		}
		lock.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Migrations complete")
		return
	}
	if command == "backup" || command == "restore" {
		if e := app.CheckDataDirectory(c.DataDir); e != nil {
			log.Fatal(e)
		}
		if e := app.ProtectData(c.DataDir, c.Mode == "managed"); e != nil {
			log.Fatal(e)
		}
		if *file == "" {
			log.Fatal("--file is required")
		}
		if e = os.MkdirAll(c.DataDir, 0700); e != nil {
			log.Fatal(e)
		}
		lock := flock.New(filepath.Join(c.DataDir, "mytube.lock"))
		ok, err := lock.TryLock()
		if err != nil || !ok {
			log.Fatal("Stop mytube before backup or restore")
		}
		if command == "backup" {
			e = app.BackupAll(c, *file)
		} else {
			e = app.RestoreAll(c, *file)
		}
		_ = lock.Unlock()
		if e != nil {
			log.Fatal(e)
		}
		fmt.Println(command + " completed")
		return
	}
	p := &program{config: c, noBrowser: *noBrowser}
	cfg := &service.Config{Name: "OneWorkspaceMyTube", DisplayName: "One Workspace My Tube", Description: "Portable video library", Arguments: []string{"--config", *config, "--data", c.DataDir, "--media", c.MediaDir, "--no-browser", "run"}, WorkingDirectory: filepath.Dir(exe)}
	svc, e := service.New(p, cfg)
	if e != nil {
		log.Fatal(e)
	}
	switch command {
	case "install", "uninstall", "start", "stop", "restart":
		if command == "install" && c.Mode == "local" {
			log.Fatal("Windows service requires managed OIDC mode in config.json")
		}
		if e = service.Control(svc, command); e != nil {
			log.Fatal(e)
		}
		return
	case "run":
	default:
		log.Fatal("unknown command")
	}
	if !service.Interactive() {
		if e = svc.Run(); e != nil {
			log.Fatal(e)
		}
		return
	}
	if e = p.Start(svc); e != nil {
		log.Fatal(e)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case <-signals:
	case <-p.done:
	}
	if e = p.Stop(svc); e != nil {
		log.Fatal(e)
	}
}
