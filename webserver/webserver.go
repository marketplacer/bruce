package webserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

// ImageResponse JSON response from uploading a file
type ImageResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// Path used for storing images on local disk
var ImageDir string

func imagePath(imageID string) string {
	if ImageDir == "" {
		log.Fatal("Image dir not set")
	}

	return ImageDir + "/" + imageID
}

// Router returns the entrypoint webserver used from the command line and tests
func Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/bruce/upload", uploadHandler)
	r.HandleFunc("/bruce/image/{imageId}/{filename}", imageHandler)
	r.HandleFunc("/bruce/image/{imageId}/{filename}/{size}", imageHandler)

	return r
}

func imageHandler(r http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	imageData, err := fetchImage(vars["imageId"], vars["size"])
	if os.IsNotExist(err) {
		http.NotFound(r, req)
		return
	} else if err != nil {
		log.Panicln(err)
	}

	imageBytes, err := ioutil.ReadAll(imageData)
	if err != nil {
		log.Panicln(err)
	}
	_, err = r.Write(imageBytes)
	if err != nil {
		log.Panicln(err)
	}

}

func fetchImage(imageID string, size string) (io.Reader, error) {
	imageReader, err := os.Open(imagePath(imageID))

	if err != nil {
		return nil, err
	}

	if size == "" || size == "original" {
		return imageReader, nil
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return nil, errors.New("Invalid size format")
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	originalImage, _, err := image.Decode(imageReader)
	if err != nil {
		return nil, err
	}
	resizedImage := resize.Thumbnail(uint(width), uint(height), originalImage, resize.Lanczos2)

	imageOut := &bytes.Buffer{}
	err = jpeg.Encode(imageOut, resizedImage, nil)
	if err != nil {
		log.Panicln(err)
	}

	return imageOut, nil

}

func uploadHandler(r http.ResponseWriter, req *http.Request) {
	file, header, err := req.FormFile("file")
	filename := filepath.Base(header.Filename)

	if err != nil {
		log.Panicln(err)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Panicln(err)
	}
	sum := ChecksumFile(bytes)

	_, err = os.Stat(imagePath(sum))

	if os.IsNotExist(err) {
		err = ioutil.WriteFile(imagePath(sum), bytes, 0444)
		if err != nil {
			log.Panicln(err)
		}
	} else if err != nil {
		log.Panicln(err)
	}

	imageResponse := ImageResponse{
		URL: "http://" + req.Host + "/bruce/image/" + sum + "/" + filename,
		ID:  sum,
	}

	responseJSON, err := json.Marshal(imageResponse)
	if err != nil {
		log.Panicln(err)
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = r.Write(responseJSON)
	if err != nil {
		log.Panicln(err)
	}
}

// ChecksumFile Return SHA256 for []byte input
func ChecksumFile(file []byte) string {
	hasher := sha256.New()
	_, err := hasher.Write(file)
	if err != nil {
		log.Panicln(err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
