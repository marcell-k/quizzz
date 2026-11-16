package main

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
)

const firstPort = 8080

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

	ln, port, err := listen()
	if err != nil {
		log.Fatal(err)
	}

	page := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, dist, name)
		}
	}

	game := NewGame([]string{"A csapat", "B csapat", "C csapat", "D csapat", "E csapat"}, newToken)
	hub := NewHub(game)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", page("player.html"))
	mux.HandleFunc("GET /admin", loopbackOnly(page("admin.html")))
	mux.HandleFunc("GET /display", page("display.html"))
	mux.HandleFunc("GET /help", page("help.html"))
	mux.HandleFunc("GET /ws", hub.playerHandler)
	mux.HandleFunc("GET /ws/admin", loopbackOnly(hub.adminHandler))
	mux.HandleFunc("GET /qr.png", loopbackOnly(qrHandler(port)))
	mux.HandleFunc("GET /info", loopbackOnly(infoHandler(port)))
	mux.Handle("GET /", http.FileServerFS(dist))

	log.Printf("local:  http://localhost:%d/admin", port)
	log.Printf("phones: %s", joinURL(port))
	log.Fatal(http.Serve(ln, logReq(mux)))
}
