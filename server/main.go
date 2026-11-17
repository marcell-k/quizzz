package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	firstPort = 8080
	quizDir   = "quizzes"
	uiDir     = "ui"
)

// mediaHandler serves files by plain file name. Only listed extensions are
// allowed, so .txt question files and anything else never leave the machine.
func mediaHandler(dir string, exts ...string) http.HandlerFunc {
	allowed := make(map[string]bool, len(exts))
	for _, e := range exts {
		allowed[e] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
			http.NotFound(w, r)
			return
		}
		if !allowed[strings.ToLower(filepath.Ext(name))] {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, name))
	}
}

// uiHandler serves the user-editable look: logo images and theme.css.
// A missing theme.css returns an empty stylesheet, so the <link> never 404s.
func uiHandler(dir string) http.HandlerFunc {
	images := mediaHandler(dir, ".jpg", ".jpeg", ".png")
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name != "theme.css" {
			images(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			http.ServeFile(w, r, filepath.Join(dir, name))
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if _, err := io.WriteString(w, "/* ui/theme.css nincs kitöltve. */\n"); err != nil {
			log.Printf("theme: %v", err)
		}
	}
}

// quizzesHandler lists selectable .txt files for the admin picker.
func quizzesHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		names, err := listQuizzes(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(names); err != nil {
			log.Printf("list quizzes: %v", err)
		}
	}
}

// listen tries firstPort..firstPort+19, returns first free.
func listen() (net.Listener, int, error) {
	var lastErr error
	for p := firstPort; p < firstPort+20; p++ {
		ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", p))
		if err == nil {
			return ln, p, nil
		}
		lastErr = err
	}
	return nil, 0, lastErr
}
func logReq(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("req %s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

func main() {
	dist, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	release, err := acquireLock(lockFile)
	if err != nil {
		log.Fatal(err)
	}
	defer release()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		release()
		log.Printf("stopped")
		os.Exit(0)
	}()

	ln, port, err := listen()
	if err != nil {
		log.Fatal(err)
	}

	page := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, dist, name)
		}
	}

	settings, err := LoadSettings(settingsFile)
	if err != nil {
		log.Printf("settings: %v (using defaults)", err)
	}
	game := NewGame(settings, newToken)
	hub := NewHub(game, quizDir, settingsFile, autosaveFile)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", page("player.html"))
	mux.HandleFunc("GET /admin", loopbackOnly(page("admin.html")))
	mux.HandleFunc("GET /display", page("display.html"))
	mux.HandleFunc("GET /help", page("help.html"))
	mux.HandleFunc("GET /ws", hub.playerHandler)
	mux.HandleFunc("GET /ws/admin", loopbackOnly(hub.adminHandler))
	mux.HandleFunc("GET /api/quizzes", loopbackOnly(quizzesHandler(quizDir)))
	mux.HandleFunc("GET /qr.png", loopbackOnly(qrHandler(port)))
	mux.HandleFunc("GET /info", loopbackOnly(infoHandler(port)))
	mux.HandleFunc("GET /media/quizzes/{name}", mediaHandler(quizDir, ".jpg", ".jpeg", ".png"))
	mux.HandleFunc("GET /media/ui/{name}", uiHandler(uiDir))
	mux.Handle("GET /", http.FileServerFS(dist))

	log.Printf("local:  http://localhost:%d/admin", port)
	log.Printf("phones: %s", joinURL(port))
	log.Fatal(http.Serve(ln, logReq(mux)))
}
