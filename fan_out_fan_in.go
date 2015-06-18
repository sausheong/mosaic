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
	"sync"
	"time"
)

// cut out the image and return individual channels with image.Image
// no encoding of JPEG
func cutWithChannelNoEncoding(original image.Image, db *map[string][3]float64, tileSize, x1, y1, x2, y2 int) <-chan image.Image {
	c := make(chan image.Image)
	sp := image.Point{0, 0}
	go func() {
		newimage := image.NewNRGBA(image.Rect(x1, y1, x2, y2))
		for y := y1; y < y2; y = y + tileSize {
			for x := x1; x < x2; x = x + tileSize {
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

		c <- newimage.SubImage(newimage.Rect)
	}()
	return c
}

// combine the images and return the encoding string
func combineImage(r image.Rectangle, c1, c2, c3, c4 <-chan image.Image) <-chan string {
	c := make(chan string)

	go func() {
		newimage := image.NewNRGBA(r)
		var wg sync.WaitGroup
		copy := func(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point) {
			draw.Draw(dst, r, src, sp, draw.Src)
			wg.Done()
		}
		wg.Add(4)
		for i := 1; i < 5; i++ {
			select {
			case s1 := <-c1:
				go copy(newimage, s1.Bounds(), s1, image.Point{r.Min.X, r.Min.Y})
			case s2 := <-c2:
				go copy(newimage, s2.Bounds(), s2, image.Point{r.Max.X / 2, r.Min.Y})
			case s3 := <-c3:
				go copy(newimage, s3.Bounds(), s3, image.Point{r.Min.X, r.Max.Y / 2})
			case s4 := <-c4:
				go copy(newimage, s4.Bounds(), s4, image.Point{r.Max.X / 2, r.Max.Y / 2})
			default:
				if i > 0 {
					i--
				}
			}

		}
		wg.Wait()
		buf2 := new(bytes.Buffer)
		jpeg.Encode(buf2, newimage, nil)
		c <- base64.StdEncoding.EncodeToString(buf2.Bytes())
		close(c)
	}()
	return c
}

//  Handler function for fan-out and fan-in
func fanOutFanInHandlerFunc(w http.ResponseWriter, r *http.Request) {
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

	// fan-out
	c1 := cutWithChannelNoEncoding(original, &db, tileSize, bounds.Min.X, bounds.Min.Y, bounds.Max.X/2, bounds.Max.Y/2)
	c2 := cutWithChannelNoEncoding(original, &db, tileSize, bounds.Max.X/2, bounds.Min.Y, bounds.Max.X, bounds.Max.Y/2)
	c3 := cutWithChannelNoEncoding(original, &db, tileSize, bounds.Min.X, bounds.Max.Y/2, bounds.Max.X/2, bounds.Max.Y)
	c4 := cutWithChannelNoEncoding(original, &db, tileSize, bounds.Max.X/2, bounds.Max.Y/2, bounds.Max.X, bounds.Max.Y)

	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	// fan-in
	c := combineImage(bounds, c1, c2, c3, c4)
	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic":   <-c,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}

	t, _ := template.ParseFiles("results.html")
	t.Execute(w, images)
}
