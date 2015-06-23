package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type PhotoSearch struct {
	Photos Photoset `json:"photos"`
	Stat   string   `json:"stat"`
}
type Photoset struct {
	Page    int          `json:"page"`
	Pages   int       `json:"pages"`
	Perpage int          `json:"perpage"`
	Total   string       `json:"total"`
	Photo   []PhotoInfo `json:"photo"`
}

type PhotoInfo struct {
	Id       string `json:"id"`
	Owner    string `json:"owner"`
	Secret   string `json:"secret"`
	Server   string `json:"server"`
	Farm     int    `json:"farm"`
}

func reloadTilesDBHandlerFunc(w http.ResponseWriter, r *http.Request) {
	go func() {
		TILESDB = tilesDB()
	}()	
	http.Redirect(w, r, "/", http.StatusFound)	
}

func fetchTilesHandlerFunc(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("query")
	key := "2fa80add43afd95e9c4d8e8b540669fb"
	format := "https://api.flickr.com/services/rest/?method=flickr.photos.search&api_key=%s&tags=%s&media=photo&format=json&nojsoncallback=1&page=%d&per_page=500"

	get := func(page int) {
		url := fmt.Sprintf(format, key, q, page)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("error:", err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error:", err)
		}

		var search PhotoSearch
		json.Unmarshal(body, &search)

		photos := search.Photos.Photo
		for _, p := range photos {
			f := "https://farm%d.staticflickr.com/%s/%s_%s_q.jpg"			
			picUrl := fmt.Sprintf(f, p.Farm, p.Server, p.Id, p.Secret)
			picResp, err := http.Get(picUrl)
			if err != nil {
				fmt.Println("error while extracting tile:", err, picUrl)
			}
			body, err := ioutil.ReadAll(picResp.Body)
			if err == nil {
				filename := fmt.Sprintf("tiles/%d-%s-%s-%s-%s.jpg", p.Farm, p.Server, p.Id, p.Secret, p.Owner)			
				err = ioutil.WriteFile(filename, body, 0644)
				if err != nil {
					fmt.Println("error writing to file:", err, filename)
				}				
			} else {
				fmt.Println("error reading body:", err, picUrl)
			}
			picResp.Body.Close()
		}
	}
	pages := 40
	for i := 1; i <= pages; i++ {
		go get(i)
	}
	http.Redirect(w, r, "/fetch", http.StatusFound)
	
}


