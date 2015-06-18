package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"
	"time"
)

// fan-out with individual channels //////////////////////////////////////////

func cutWithChannel(original image.Image, db *map[string][3]float64, tileSize, x1, y1, x2, y2 int) <-chan string {
	c := make(chan string)
	go func() {
		newimage := image.NewNRGBA(image.Rect(x1, y1, x2, y2))
		sp := image.Point{0, 0}

		for y := y1; y < y2; y = y + tileSize {
			for x := x1; x < x2; x = x + tileSize {
				// get the RGBA value
				r, g, b, _ := original.At(x, y).RGBA()
				color := [3]float64{float64(r), float64(g), float64(b)}
				nearest := nearest(color, db)
				file, err := os.Open(nearest)
				if err == nil {
					img, _, err := image.Decode(file)
					if err == nil {
						t := resize(img, tileSize)
						tile := t.SubImage(t.Bounds())
						tileBounds := image.Rect(x, y, x+tileSize, y+tileSize)
						draw.Draw(newimage, tileBounds, tile, sp, draw.Src)
					} else {
						fmt.Println("error in decoding nearest", err, nearest)
					}
				} else {
					fmt.Println("error opening file when creating mosaic:", nearest)
				}
				file.Close()
			}
		}
		buf2 := new(bytes.Buffer)
		jpeg.Encode(buf2, newimage, nil)
		c <- base64.StdEncoding.EncodeToString(buf2.Bytes())

	}()
	return c
}

// Handler function for fan-out with individual channels
func fanOutWithChannelHandlerFunc(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	// get the content from the POSTed form
	r.ParseMultipartForm(10485760) // max body in memory is 10MB
	file, _, _ := r.FormFile("image")
	defer file.Close()
	tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))
	//
	//   // decode and get original image
	original, _, _ := image.Decode(file)
	bounds := original.Bounds()
	db := cloneTilesDB()

	c1 := cutWithChannel(original, &db, tileSize, bounds.Min.X, bounds.Min.Y, bounds.Max.X/2, bounds.Max.Y/2)
	c2 := cutWithChannel(original, &db, tileSize, bounds.Max.X/2, bounds.Min.Y, bounds.Max.X, bounds.Max.Y/2)
	c3 := cutWithChannel(original, &db, tileSize, bounds.Min.X, bounds.Max.Y/2, bounds.Max.X/2, bounds.Max.Y)
	c4 := cutWithChannel(original, &db, tileSize, bounds.Max.X/2, bounds.Max.Y/2, bounds.Max.X, bounds.Max.Y)

	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())
	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"part1":    <-c1,
		"part2":    <-c2,
		"part3":    <-c3,
		"part4":    <-c4,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}

	t, _ := template.ParseFiles("results_parts.html")
	t.Execute(w, images)
}
