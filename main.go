package main

import (
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"io/ioutil"
)

func uploadHandlerFunc(w http.ResponseWriter, r *http.Request) {	
	t, _ := template.ParseFiles("upload.html")
	t.Execute(w, len(TILESDB))
}

func fetchHandlerFunc(w http.ResponseWriter, r *http.Request) {
	files,_ := ioutil.ReadDir("tiles")	
	t, _ := template.ParseFiles("fetch.html")
	t.Execute(w, len(files))
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("Starting mosaic server ...")
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))

	mux.HandleFunc("/", uploadHandlerFunc)
	mux.HandleFunc("/reload", reloadTilesDBHandlerFunc)
	mux.HandleFunc("/fetch", fetchHandlerFunc)
	mux.HandleFunc("/fetch_tiles", fetchTilesHandlerFunc)
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
