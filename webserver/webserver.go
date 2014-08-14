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

	"launchpad.net/goamz/s3"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

// ImageResponse JSON response from uploading a file
type ImageResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// Path used for storing images on local disk
var (
	ImageDir           string
	S3Connection       *s3.S3
	BucketName         string
	ErrS3NotConfigured = errors.New("S3 Not configured")
	ErrFileNotFound    = errors.New("File not found")
)

func serverError(r http.ResponseWriter, req *http.Request, err error) {
	log.Println("Server error", err)
	http.Error(r, "Something went wrong", 500)
}

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
	if err == ErrFileNotFound {
		http.NotFound(r, req)
		return
	} else if err != nil {
		serverError(r, req, err)
		return
	}

	imageBytes, err := ioutil.ReadAll(imageData)
	if err != nil {
		serverError(r, req, err)
		return
	}
	_, err = r.Write(imageBytes)
	if err != nil {
		serverError(r, req, err)
		return
	}

}

func fetchImage(imageID string, size string) (io.Reader, error) {
	imageReader, err := os.Open(imagePath(imageID))

	if os.IsNotExist(err) {
		if S3Connection == nil {
			return nil, ErrS3NotConfigured
		}

		bucket := S3Connection.Bucket(BucketName)
		downloader, err := bucket.GetReader("/bruce/images/" + imageID)
		if err != nil {
			return nil, err
		}

		outputFile, err := os.Create(imagePath(imageID))
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(outputFile, downloader)
		if err != nil {
			return nil, err
		}

		err = outputFile.Close()
		if err != nil {
			return nil, err
		}
		return fetchImage(imageID, size) // next time this will work for sure
	}

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
		return nil, err
	}

	return imageOut, nil

}

func uploadHandler(r http.ResponseWriter, req *http.Request) {
	file, header, err := req.FormFile("file")
	filename := filepath.Base(header.Filename)

	if err != nil {
		serverError(r, req, err)
		return
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		serverError(r, req, err)
		return
	}
	sum, err := ChecksumFile(bytes)
	if err != nil {
		serverError(r, req, err)
		return
	}

	if S3Connection == nil {
		serverError(r, req, errors.New("No S3 connection"))
		return
	}

	bucket := S3Connection.Bucket(BucketName)

	err = bucket.Put("/bruce/images/"+sum, bytes, "binary-stream", "public-read")
	if err != nil {
		serverError(r, req, err)
		return
	}

	_, err = os.Stat(imagePath(sum))

	if os.IsNotExist(err) {
		err = ioutil.WriteFile(imagePath(sum), bytes, 0444)
		if err != nil {
			serverError(r, req, err)
			return
		}
	} else if err != nil {
		serverError(r, req, err)
		return
	}

	imageResponse := ImageResponse{
		URL: "http://" + req.Host + "/bruce/image/" + sum + "/" + filename,
		ID:  sum,
	}

	responseJSON, err := json.Marshal(imageResponse)
	if err != nil {
		serverError(r, req, err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = r.Write(responseJSON)
	if err != nil {
		serverError(r, req, err)
		return
	}
}

// ChecksumFile Return SHA256 for []byte input
func ChecksumFile(file []byte) (string, error) {
	hasher := sha256.New()
	_, err := hasher.Write(file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
