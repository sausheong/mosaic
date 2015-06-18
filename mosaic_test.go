package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResize(t *testing.T) {
	t.Skip()
	file, _ := os.Open("cat_m.jpg")
	img, _, _ := image.Decode(file)
	nrbga := resize(img, 10)

	outfile, _ := os.Create("test/test_output.jpg")
	sm := nrbga.SubImage(nrbga.Bounds())
	jpeg.Encode(outfile, sm, nil)
}

func TestTileDB(t *testing.T) {
	t.Skip()
	db := tilesDB()
	if len(db) != 1000 {
		t.Error("There is more than or less than 1000 images in the database")
	}
}

func TestAverageColor(t *testing.T) {
	t.Skip()
	file, _ := os.Open("cat/7405d2cee6c483d4e177ecc50858ec0f342865ba.jpg")
	img, _, _ := image.Decode(file)
	ave := averageColor(img)
	t.Log("average color:", ave)
}

func TestDistance(t *testing.T) {
	t.Skip()
	p1, p2 := [3]float64{1.0, 2.0, 3.0}, [3]float64{4.0, 5.0, 6.0}
	dist := distance(p1, p2)
	if dist != 5.196152422706632 {
		t.Error("Can't find distance")
	}
	p3 := [3]float64{1.0, 2.0, 3.0}
	dist = distance(p1, p3)
	if dist != 0.0 {
		t.Error("Can't find distance")
	}

}

func TestMosaic(t *testing.T) {
	t.Skip()
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_no_concurrency", noConcurrencyHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_no_concurrency", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}
}

func TestFanOutFanIn(t *testing.T) {
	t.Skip()
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_fanin", fanOutFanInHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_fanin", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}
}

// Benchmarks

// Small

func BenchmarkNoConcurrencySmall(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_no_concurrency", noConcurrencyHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_no_concurrency", params, "image", "cat_s.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutSmall(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout", fanOutHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout", params, "image", "cat_s.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutWithChannelSmall(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_channel", fanOutWithChannelHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_channel", params, "image", "cat_s.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutFanInSmall(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_fanin", fanOutFanInHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_fanin", params, "image", "cat_s.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

/// Medium

func BenchmarkNoConcurrencyMedium(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_no_concurrency", noConcurrencyHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_no_concurrency", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutMedium(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout", fanOutHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutWithChannelMedium(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_channel", fanOutWithChannelHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_channel", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutFanInMedium(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_fanin", fanOutFanInHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_fanin", params, "image", "cat_m.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

/// Large

func BenchmarkNoConcurrencyLarge(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_no_concurrency", noConcurrencyHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_no_concurrency", params, "image", "cat_l.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutLarge(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout", fanOutHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout", params, "image", "cat_l.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutWithChannelLarge(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_channel", fanOutWithChannelHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_channel", params, "image", "cat_l.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

func BenchmarkFanOutFanInLarge(b *testing.B) {
	runtime.GOMAXPROCS(4)
	mux := http.NewServeMux()
	mux.HandleFunc("/mosaic_fanout_fanin", fanOutFanInHandlerFunc)

	writer := httptest.NewRecorder()
	params := map[string]string{
		"tile_size": "15",
	}
	request, _ := fileRequest("/mosaic_fanout_fanin", params, "image", "cat_l.jpg")
	mux.ServeHTTP(writer, request)

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(writer, request)
	}
}

// create a file upload
func fileRequest(uri string, params map[string]string, paramName, path string) (req *http.Request, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest("POST", uri, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return
}
