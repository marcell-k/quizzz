package main

import (
"io/fs"
"log"
"net/http"
)

func main() {
dist, err := fs.Sub(distFS, "web/dist")
if err != nil {
log.Fatal(err)
}

page := func(name string) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
http.ServeFileFS(w, r, dist, name)
}
}

mux := http.NewServeMux()
mux.HandleFunc("GET /{$}", page("player.html"))
mux.HandleFunc("GET /admin", page("admin.html"))
mux.HandleFunc("GET /display", page("display.html"))
mux.HandleFunc("GET /help", page("help.html"))
mux.Handle("GET /", http.FileServerFS(dist))

addr := ":8080"
log.Printf("listening on %s", addr)
log.Fatal(http.ListenAndServe(addr, mux))
}
