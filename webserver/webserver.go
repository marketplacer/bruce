package webserver

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
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
	// this should be settable, especially for tests
	return ImageDir + "/" + imageID
}

// Router returns the entrypoint webserver used from the command line and tests
func Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/bruce/upload", uploadHandler)
	r.HandleFunc("/bruce/image/{size}/{imageId}/{filename}", imageHandler)

	return r
}

func imageHandler(r http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	contents, err := ioutil.ReadFile(imagePath(vars["imageId"]))
	if os.IsNotExist(err) {
		http.NotFound(r, req)
		return
	} else if err != nil {
		log.Panicln(err)
	}
	r.Write(contents)
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
		URL: "http://" + req.Host + "/bruce/image/original/" + sum + "/" + filename,
		ID:  sum,
	}

	responseJSON, err := json.Marshal(imageResponse)
	if err != nil {
		log.Panicln(err)
	}
	req.Header.Add("Content-Type", "application/json")
	r.Write(responseJSON)
}

// ChecksumFile Return SHA256 for []byte input
func ChecksumFile(file []byte) string {
	hasher := sha256.New()
	hasher.Write(file)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
