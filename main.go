package main

import (
	"fmt"
	"html/template"
	"net/http"
	"runtime"
)


func upload(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("upload.html")
	t.Execute(w, nil)
}

func main() {
	runtime.GOMAXPROCS(4)
	fmt.Println("Starting mosaic server ...")	
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))

	mux.HandleFunc("/", upload)
	mux.HandleFunc("/mosaic_no_concurrency", noConcurrencyHandlerFunc)
	mux.HandleFunc("/mosaic_fanout_channel", fanOutWithChannelHandlerFunc)
	mux.HandleFunc("/mosaic_fanout_fanin", fanOutFanInHandlerFunc)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}
	TILESDB = tilesDB()
	fmt.Println("Mosaic server started.")
	server.ListenAndServe()
	
}
