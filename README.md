# Mosaic

A few months ago, my good friend Satish Talim had this great idea to create a series of Go challenges to help Go programmers to up their game. The idea was to create a programming problem every month (or so) that presents a fresh and interesting challenge to the Go community. There would be prizes to win, but more importantly, it was a community effort to help each other to improve ourselves. Satish asked me to write a challenge, and I readily agreed to create [challenge #3](http://golang-challenge.com/go-challenge3/).

Being a web application programmer for most of my career, it was the most natural thing to do to create a challenge that's based on creating a web application. And some time back, [I wrote a mosaic-generating script using Ruby during a hackathon](https://developer.yahoo.com/blogs/ydn/creating-photo-mosaics-yahoo-boss-image-search-7453.html), so I thought to marry both ideas together to create a challenge to create photo mosaic web app.

To be honest, at the time of issuing the challenge I haven't actually written the photo mosaic web app yet. In fact, I only started writing it after the challenge was over. It took me the better part of 2 days to complete the photo mosaic web app. But I wasn't finished yet, and I wanted to go a bit further and use Go's concurrency to improve its performance. What I'm describing below is what I did.

The live site is found at http://mosaic.saush.com. It is deployed using [Docker](https://www.docker.com) to Digital Ocean, through [Tutum](https://www.tutum.co). The performance on the live site is not as good as described here, as it is only running on 1 CPU with 512MB.

## Creating the photo mosaic
A photographic mosaic, or a photo mosaic is a picture (usually a photograph) that has been divided into (usually equal sized) rectangular sections, each of which is replaced with another picture (called a tile picture). If we view it from far away or if you squint at it, then the original picture can be seen. If we look closer though, we will see that the picture is in fact made up of many hundreds or thousands of smaller tile pictures.

The basic idea is simple – the web application allows a user to upload a target picture, which will be used to create a photo mosaic. To make things simple, I will assume that tile pictures are already available and are correctly sized.

Let’s start with the photo mosaic algorithm. The steps are simple and the whole web application can be written without the use of any third-party libraries.

1. Build up a tile database, hash of tile pictures, by scanning a directory of pictures, then using the file name as the key and the average color of the picture as the value. The average color is a 3-tuple calculated from getting the red, green and blue (RGB) of every pixel and adding up all the reds, greens and blues, then divided by the total number of pixels
2. Cut the target picture into smaller pictures of the correct tile size
3. For every tile-sized piece of the target picture, assume the average color to the the color of the top left pixel of that piece
4. Find the corresponding tile in the tile database that is the nearest match to the average color of the piece of the target picture, and place that tile in the corresponding position in the photo mosaic. To find the nearest match, we calculate the Euclidean distance between the two color 3-tuples by converting each color 3-tuple into a point in a 3-dimensional space
5. Remove the tile from the tile database so that each tile in the photo mosaic is unique

I placed all the mosaic creating code in a single source file named mosaic.go. Let’s look at each function in this file. 

```go
// find the average color of the picture
func averageColor(img image.Image) [3]float64 {
	bounds := img.Bounds()
	r, g, b := 0.0, 0.0, 0.0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			r, g, b = r+float64(r1), g+float64(g1), b+float64(b1)
		}
	}
	totalPixels := float64(bounds.Max.X * bounds.Max.Y)
	return [3]float64{r / totalPixels, g / totalPixels, b / totalPixels}
}
```

First is the `averageColor` function, which takes the red, green and blue of each pixel in the image, adds up all the reds, greens and blues and then divides each sum by the total number of pixels in the image. Then we create a 3-tuple (actually a 3 element array) consisting of these numbers. 
Next, we have the `resize` function. The resize function resizes an image to a new width.

```go
// resize an image to its new width
func resize(in image.Image, newWidth int) image.NRGBA {
	bounds := in.Bounds()
	width := bounds.Max.X - bounds.Min.X
	ratio := width / newWidth
	out := image.NewNRGBA(image.Rect(bounds.Min.X/ratio, bounds.Min.X/ratio, bounds.Max.X/ratio, bounds.Max.Y/ratio))
	for y, j := bounds.Min.Y, bounds.Min.Y; y < bounds.Max.Y; y, j = y+ratio, j+1 {
		for x, i := bounds.Min.X, bounds.Min.X; x < bounds.Max.X; x, i = x+ratio, i+1 {
			r, g, b, a := in.At(x, y).RGBA()
			out.SetNRGBA(i, j, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
		}
	}
	return *out
}
```

The `tileDB` function creates a database of the tile picture by scanning the directory where the tile pictures are located.

```go
// populate a tiles database in memory
func tilesDB() map[string][3]float64 {
	fmt.Println("Start populating tiles db ...")
	db := make(map[string][3]float64)
	files, _ := ioutil.ReadDir("tiles")
	for _, f := range files {
		name := "tiles/" + f.Name()
		file, err := os.Open(name)
		if err == nil {
			img, _, err := image.Decode(file)
			if err == nil {
				db[name] = averageColor(img)
			} else {
				fmt.Println(":", err, name)
			}
		} else {
			fmt.Println("cannot open file", name, err)
		}
		file.Close()
	}
	fmt.Println("Finished populating tiles db.")
	return db
}
```

The tile database is a map with a string as the key a 3-tuple (in this case a 3-element array) as the value. I open up each image file in the directory and then get the average color of the image to create an entry in the map. The tile database is used to find the correct tile picture in the tile picture directory. It is passed into the nearest function, along with the target color 3-tuple.

```go
// find the nearest matching image
func nearest(target [3]float64, db *map[string][3]float64) string {
	var filename string
	smallest := 1000000.0
	for k, v := range *db {
		dist := distance(target, v)
		if dist < smallest {
			filename, smallest = k, dist
		}
	}
	delete(*db, filename)
	return filename
}
```

Each entry in the tile database is compared with the target color and the entry with the smallest distance is returned as the nearest tile, and also removed from the tile database. The distance function calculates the Euclidean distance between two 3-tuples.

```go
// find the Eucleadian distance between 2 points
func distance(p1 [3]float64, p2 [3]float64) float64 {
	return math.Sqrt(sq(p2[0]-p1[0]) + sq(p2[1]-p1[1]) + sq(p2[2]-p1[2]))
}

// find the square
func sq(n float64) float64 {
	return n * n
}
```

Finally, scanning and loading the tile database every time a photo mosaic is created can be pretty cumbersome. I want to do that only once, and clone the tile database every time a photo mosaic is created. The source tile database, TILEDB is then created as a global variable and populated on the start of the web application.

```go
var TILESDB map[string][3]float64

// clone the tile database each time we generate the photo mosaic
func cloneTilesDB() map[string][3]float64 {
	db := make(map[string][3]float64)
	for k, v := range TILESDB {
		db[k] = v
	}
	return db
}
```
## The photo mosaic web application

With the mosaic-generating functions in place, I can start writing my web application. The web application is placed in a source code file named main.go.

```go
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"bytes"
	"encoding/base64"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"strconv"
)

func main() {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))
	mux.HandleFunc("/", upload)
	mux.HandleFunc("/mosaic ", mosaic)
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}
// building up the source tile database
	TILESDB = tilesDB()
	fmt.Println("Mosaic server started.")
	server.ListenAndServe()
}

// to display the upload page
func upload(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("upload.html")
	t.Execute(w, nil)
}

// the HandlerFunc that contains all the photo mosaic generating algorithms
func mosaic(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	// get the content from the POSTed form
	r.ParseMultipartForm(10485760) // max body in memory is 10MB
	file, _, _ := r.FormFile("image")
	defer file.Close()
	tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))
	// decode and get original image
	original, _, _ := image.Decode(file)
	bounds := original.Bounds()
	// create a new image for the mosaic
	newimage := image.NewNRGBA(image.Rect(bounds.Min.X, bounds.Min.X, bounds.Max.X, bounds.Max.Y))
	// build up the tiles database
	db := cloneTilesDB()
	// source point for each tile, which starts with 0, 0 of each tile
	sp := image.Point{0, 0}
	for y := bounds.Min.Y; y < bounds.Max.Y; y = y + tileSize {
		for x := bounds.Min.X; x < bounds.Max.X; x = x + tileSize {
			// use the top left most pixel as the average color
			r, g, b, _ := original.At(x, y).RGBA()
			color := [3]float64{float64(r), float64(g), float64(b)}
			// get the closest tile from the tiles DB
			nearest := nearest(color, &db)
			file, err := os.Open(nearest)
			if err == nil {
				img, _, err := image.Decode(file)
				if err == nil {
					// resize the tile to the correct size 
					t := resize(img, tileSize)
					tile := t.SubImage(t.Bounds())
					tileBounds := image.Rect(x, y, x+tileSize, y+tileSize)
					// draw the tile into the mosaic
					draw.Draw(newimage, tileBounds, tile, sp, draw.Src)
				} else {
					fmt.Println("error:", err, nearest)
				}
			} else {
				fmt.Println("error:", nearest)
			}
			file.Close()
		}
	}

	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	buf2 := new(bytes.Buffer)
	jpeg.Encode(buf2, newimage, nil)
	mosaic := base64.StdEncoding.EncodeToString(buf2.Bytes())
	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic":   mosaic,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}
	t, _ := template.ParseFiles("results.html")
	t.Execute(w, images)
}
```

The main logic for creating the photo mosaic is in the mosaic function, which is a handler function. First, I get the uploaded file and also the tile size from the form.

```go
// get the content from the POSTed form
r.ParseMultipartForm(10485760) // max body in memory is 10MB
file, _, _ := r.FormFile("image")
defer file.Close()
tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))
```

Next, I decode the uploaded target image, and also create a new photo mosaic image.

```go
// decode and get original image
original, _, _ := image.Decode(file)
bounds := original.Bounds()
// create a new image for the mosaic
newimage := image.NewNRGBA(image.Rect(bounds.Min.X, bounds.Min.X, bounds.Max.X, bounds.Max.Y))
```

I also clone the source tile database, and set up the source point for each tile (this is needed by the image/draw package later).

```go
// build up the tiles database
db := cloneTilesDB()
// source point for each tile, which starts with 0, 0 of each tile
sp := image.Point{0, 0}
```

I am now ready to iterate through each tile-sized piece of the target image.

```go
for y := bounds.Min.Y; y < bounds.Max.Y; y = y + tileSize {
	for x := bounds.Min.X; x < bounds.Max.X; x = x + tileSize {
		// use the top left most pixel color as the average color
		r, g, b, _ := original.At(x, y).RGBA()
		color := [3]float64{float64(r), float64(g), float64(b)}
		// get the closest tile from the tiles DB
		nearest := nearest(color, &db)
		file, err := os.Open(nearest)
		if err == nil {
			img, _, err := image.Decode(file)
			if err == nil {
				// resize the tile to the correct size 
				t := resize(img, tileSize)
				tile := t.SubImage(t.Bounds())
				tileBounds := image.Rect(x, y, x+tileSize, y+tileSize)
				// draw the tile into the mosaic
				draw.Draw(newimage, tileBounds, tile, sp, draw.Src)
			} else {
				fmt.Println("error:", err, nearest)
			}
		} else {
			fmt.Println("error:", nearest)
		}
		file.Close()
	}
}
```

For every piece, I pick the top left pixel and assume that’s the average color. Then I find the nearest tile in the tile database that matches this color. The tile database gives me a filename, so I open up the tile picture, and resize it to the tile size. The resultant tile is drawn into the photo mosaic I created earlier.

Once the photo mosaic is created, I encode it into JPEG format, then encode it once again into a base64 string. 

```go
	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	buf2 := new(bytes.Buffer)
	jpeg.Encode(buf2, newimage, nil)
	mosaic := base64.StdEncoding.EncodeToString(buf2.Bytes())
	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic":   mosaic,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}
	t, _ := template.ParseFiles("results.html")
	t.Execute(w, images)
```

The original target picture and the photo mosaic are then sent to the results.html template to be displayed on the next page. As you can see, the image is displayed with using a data URL with the base64 content that is embedded in the web page itself.

```html
<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <title>Mosaic</title>
    ...
  </head>   
  <body>
    <div class='container'>
        <div class="col-md-6">
          <img src="data:image/jpg;base64,{{ .original }}" width="100%">
          <div class="lead">Original</div>
        </div>
        <div class="col-md-6">
          <img src="data:image/jpg;base64,{{ .mosaic }}" width="100%">
          <div class="lead">Mosaic – {{ .duration }} </div>
        </div>
        <div class="col-md-12 center">
          <a class="btn btn-lg btn-info" href="/">Go Back</a>
        </div>
    </div>   
    <br>
  </body>
</html>
```

Here’s a screenshot of the mosaic that’s created.



![Figure 1 – Basic photo mosaic web application](https://raw.githubusercontent.com/sausheong/mosaic/master/readme_images/09-01.png)

Now that we have the basic mosaic generating web application, let’s create the concurrent version of it. 

##	Concurrent photo mosaic web application

One of the more frequent use of concurrency is to improve performance. The web application I just showed created a mosaic from a 151KB JPEG image in about 2.25 seconds. The performance is not really fantastic and can definitely be improved using some concurrency. The algorithm I am using in this example to build some concurrency into the photo mosaic web application is quite straightforward. 

1. Split the original image into 4 quarters
2. Process them at the same time
3. Combine the results back into a single mosaic

From a diagrammatic point of view:

![Figure 2 – Concurrency algorithm](https://raw.githubusercontent.com/sausheong/mosaic/master/readme_images/09-05.png)

A word of caution – this is not the only way that performance can be improved or concurrency can be achieved, but only one relatively simple and straightforward way.

The only function that changes in this is the mosaic handler function. In the earlier program, I had a single handler function that created the photo mosaic. In the concurrent version of the photo mosaic web application, I need to break up that function into two separate functions, called cut and combine respectively. Both the cut and the combine functions are called from the mosaic handler function.


```go
func mosaic(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	r.ParseMultipartForm(10485760) // max body in memory is 10MB
	file, _, _ := r.FormFile("image")
	defer file.Close()
	tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))
	original, _, _ := image.Decode(file)
	bounds := original.Bounds()
	db := cloneTilesDB()

	// fan-out
	c1 := cut(original, &db, tileSize, bounds.Min.X, bounds.Min.Y, bounds.Max.X/2, bounds.Max.Y/2)
	c2 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Min.Y, bounds.Max.X, bounds.Max.Y/2)
	c3 := cut(original, &db, tileSize, bounds.Min.X, bounds.Max.Y/2, bounds.Max.X/2, bounds.Max.Y)
	c4 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Max.Y/2, bounds.Max.X, bounds.Max.Y)

	// fan-in
	c := combine(bounds, c1, c2, c3, c4)

	buf1 := new(bytes.Buffer)
	jpeg.Encode(buf1, original, nil)
	originalStr := base64.StdEncoding.EncodeToString(buf1.Bytes())

	t1 := time.Now()
	images := map[string]string{
		"original": originalStr,
		"mosaic":   <-c,
		"duration": fmt.Sprintf("%v ", t1.Sub(t0)),
	}

	t, _ := template.ParseFiles("results.html")
	t.Execute(w, images)
}
```

Cutting up the image is handled by the cut function, in what is known as the fan-out pattern. 

![Figure 3 – Splitting the target picture into 4 quadrants](https://raw.githubusercontent.com/sausheong/mosaic/master/readme_images/09-04.png)

The original image is cut up into 4 quadrants to be processed separately. 

```go
c1 := cut(original, &db, tileSize, bounds.Min.X, bounds.Min.Y, bounds.Max.X/2, bounds.Max.Y/2)
c2 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Min.Y, bounds.Max.X, bounds.Max.Y/2)
c3 := cut(original, &db, tileSize, bounds.Min.X, bounds.Max.Y/2, bounds.Max.X/2, bounds.Max.Y)
c4 := cut(original, &db, tileSize, bounds.Max.X/2, bounds.Max.Y/2, bounds.Max.X, bounds.Max.Y)
```

You might notice that these are regular functions and not goroutines, how can they run concurrently? The answer is because the cut function creates an anonymous goroutine and returns a channel.

```go
func cut(original image.Image, db *map[string][3]float64, tileSize, x1, y1, x2, y2 int) <-chan image.Image {
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
						fmt.Println("error:", err)
					}
				} else {
					fmt.Println("error:", nearest)
				}
				file.Close()
			}
		}
		c <- newimage.SubImage(newimage.Rect)
	}()
	return c
}
```

The logic is exactly the same as in the original photo mosaic web application. I created a channel in the cut function and started up an anonymous goroutine that sends the results to this channel, then return the channel. This way, the channel is immediately returned to the mosaic handler function, and the completed photo mosaic segment is sent to the channel when the processing is done. You might notice that while I’ve created the return channel as a bi-directional channel, I can typecast it to be returned as a receive-only channel. 

I’ve cut the original image into 4 separate pieces and convert each piece into a part of a photo mosaic. It’s time to put them together again, using what is commonly known as the fan-in pattern, in the combine function.

```go
func combine(r image.Rectangle, c1, c2, c3, c4 <-chan image.Image) 
<-chan string {
	c := make(chan string)
	// start a goroutine
	go func() {
		var wg sync.WaitGroup
		img:= image.NewNRGBA(r)
		copy := func(dst draw.Image, r image.Rectangle, 
src image.Image, sp image.Point) {
			draw.Draw(dst, r, src, sp, draw.Src)
			wg.Done()
		}
		wg.Add(4)
		var s1, s2, s3, s4 image.Image
		var ok1, ok2, ok3, ok4 bool
		for  {
			select {
			case s1, ok1 = <-c1:
				go copy(img, s1.Bounds(), s1,
 					image.Point{r.Min.X, r.Min.Y})
			case s2, ok2 = <-c2:
				go copy(img, s2.Bounds(), s2,
image.Point{r.Max.X / 2, r.Min.Y})
			case s3, ok3 = <-c3:
				go copy(img, s3.Bounds(), s3, 
image.Point{r.Min.X, r.Max.Y/2})
			case s4, ok4 = <-c4:
				go copy(img, s4.Bounds(), s4, 
image.Point{r.Max.X / 2, r.Max.Y / 2})
			}
			if (ok1 && ok2 && ok3 && ok4) {
				break
			}
		}
		// wait till all copy goroutines are complete
		wg.Wait()
		buf2 := new(bytes.Buffer)
		jpeg.Encode(buf2, newimage, nil)
		c <- base64.StdEncoding.EncodeToString(buf2.Bytes())
	}()
	return c
}
```

As in the cut function, the main logic in combining the images are in an anonymous goroutine, and I create and return a receive-only channel. As a result, I can encode the original image while combining the 4 photo mosaic segments.

In the anonymous goroutine, I create another anonymous function and assign it to a variable copy. This function copies a photo mosaic segment into the final photo mosaic and will be run as a goroutine later. Because the copy function is called as a goroutine, I will not be able to control when they complete. To synchronize the completion of the copying, I use a WaitGroup. I create a WaitGroup wg, then set the counter to 4 using the Add method. Each time the copy function completes, it will decrement the counter using the Done method. I call the Wait method just before encoding the image to allow all the copy goroutines to complete and I actually have a complete photo mosaic image.

Remember that the input to the combine function includes the 4 channels coming from the cut function containing the photo mosaic segments, and I don’t know when the channels actually have segments. I could try to receive each one of those channels in sequence, but that wouldn’t be very concurrent. What I would like to do is to start processing whichever segment that comes first and the select statement fits the bill nicely.

```go
var s1, s2, s3, s4 image.Image
var ok1, ok2, ok3, ok4 bool
for  {
select {
	case s1, ok1 = <-c1:
		go copy(img, s1.Bounds(), s1, 
image.Point{r.Min.X, r.Min.Y})
	case s2, ok2 = <-c2:
		go copy(img, s2.Bounds(), s2, 
image.Point{r.Max.X / 2, r.Min.Y})
	case s3, ok3 = <-c3:
		go copy(img, s3.Bounds(), s3, 
image.Point{r.Min.X, r.Max.Y / 2})
	case s4, ok4 = <-c4:
		go copy(img, s4.Bounds(), s4, 
image.Point{r.Max.X / 2, r.Max.Y / 2})
	}
	if (ok1 && ok2 && ok3 && ok4) {
		break
	}
}
```

I loop indefinitely and in each iteration, I select the channel that is ready with a value (or if more than one is available, Go randomly assigns me one). I use the image from this channel and start a goroutine with the copy function. Note that I’m using the multi-value format for receiving values from the channel, meaning the second variable (ok1, ok2, ok3 or ok4) tells me if I have successfully received from the channel. The for loop breaks once I have received successfully on all channels.

Moving on, and referring to the WaitGroup wg I used earlier, remember that even though I received all the photo mosaic segments successfully, I have in turn started 4 separate goroutines, which might not have completed at that point in time. The Wait method on the WaitGroup wg blocks the encoding of the assembled photo mosaic until the photo mosaic is completed.

Here’s a screenshot of the results, using the same target picture and tile pictures.


![Figure 4 – Photo mosaic web application with concurrency](https://raw.githubusercontent.com/sausheong/mosaic/master/readme_images/09-02.png)

If you’re sharp-eyed, you might see the slight differences in the photo mosaic that’s generated. The final photo mosaic is assembled from 4 separate pieces and the algorithm doesn’t smoothen out the rough edges. However, you can see the difference in performance – where the basic photo mosaic web application took 2.25 seconds, this one using concurrency only takes almost a quarter of that time, around 646 miliseconds.

For readers who are really observant you might realize that both web applications are actually running on just one CPU! As mentioned by Rob Pike, [concurrency is not parallelism](http://blog.golang.org/concurrency-is-not-parallelism) – what I’ve shown you is how we can take a simple algorithm and break it down into a concurrent one, with no parallelism involved! None of the goroutines are actually running in parallel (since there is only one CPU) even though they are running independently.
Of course it would be cruel not to go the last step and show how it actually runs using multiple CPUs. To do this, I simply need to set the GOMAXPROCS in the runtime to the actual number of CPUs running on my system. The changes are in the main.go file. Remember to import the runtime package before making the change below.

```go
func main() {
	// this prints out the number of CPUs in my system
	fmt.Println("Number of CPUs:", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("Starting mosaic server ...")
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))
	mux.HandleFunc("/", upload)
	mux.HandleFunc("/mosaic", mosaic)
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}
	TILESDB = tilesDB()
	fmt.Println("Mosaic server started.")
	server.ListenAndServe()
}
```

I compile and then upload the same target picture again.


![Figure 5 – Photo mosaic web application with concurrency and 8 CPUs](https://raw.githubusercontent.com/sausheong/mosaic/master/readme_images/09-03.png)

As you can see, the performance has improved 3 times, from 646 milliseconds to 216 milliseconds! And if we compare that with our original photo mosaic web application with 2.25 seconds, that’s a 10 times performance improvement! That is an actual comparison, even though we did not run it with 8 CPUs previously, our original photo mosaic web application is not concurrent and therefore can only use one CPU – giving it 8 CPUs make no difference at all.
What is also interesting to note is that there is no difference between the original and the concurrent web applications, in terms of the photo mosaic algorithm. In fact, between the two applications, the mosaic.go source file was not modified at all. The only difference is concurrency, and that is a testament to how powerful it is.
