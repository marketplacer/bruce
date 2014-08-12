package webserver_test

import (
	"encoding/json"
	"image"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/exchangegroup/bruce/webserver"
)

func setTempImageDir(t *testing.T) {
	imageDir, err := ioutil.TempDir(os.TempDir(), "uploadFile")
	webserver.ImageDir = imageDir

	if err != nil {
		t.Fatal(err)
	}
}

func UploadFile(url string, t *testing.T) (*http.Response, error) {
	setTempImageDir(t)
	path := "../bruce.jpg"
	file, err := os.Open(path)

	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	preader, pwriter := io.Pipe()

	writer := multipart.NewWriter(pwriter)

	go func() {
		part, err := writer.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			t.Fatal(err)
		}
		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}
		err = pwriter.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	resp, err := http.Post(url, writer.FormDataContentType(), preader)
	if err != nil {
		t.Fatal(err)
	}
	return resp, nil

}

func StartServerAndUpload(t *testing.T) webserver.ImageResponse {
	ts := httptest.NewServer(webserver.Router())

	uploadURL := ts.URL + "/bruce/upload"
	resp, err := UploadFile(uploadURL, t)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	var imageResponse webserver.ImageResponse

	jsonDecoder := json.NewDecoder(resp.Body)
	err = jsonDecoder.Decode(&imageResponse)
	if err != nil {
		t.Fatal(err)
	}

	return imageResponse
}

func TestFileUpload(t *testing.T) {
	setTempImageDir(t)
	ts := httptest.NewServer(webserver.Router())
	defer ts.Close()
	url := ts.URL + "/bruce/upload"
	res, err := UploadFile(url, t)

	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
}

func TestFileDownloadFullSize(t *testing.T) {
	imageResponse := StartServerAndUpload(t)

	resp, err := http.Get(imageResponse.URL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Log(string(body))
		t.Fatal("Expected status code 200, got", resp.StatusCode)
	}

	imageContents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	sum := webserver.ChecksumFile(imageContents)

	if imageResponse.ID != sum {
		t.Fatal("Expected imageId", imageResponse.ID, "to equal", sum)
	}

}

func TestDownloadResizedFile(t *testing.T) {
	imageResponse := StartServerAndUpload(t)

	resp, err := http.Get(imageResponse.URL + "/300x300")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Fatal("Expected status 200, got", resp.StatusCode)
	}

	downloadedImage, _, err := image.Decode(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if downloadedImage.Bounds().Max.X != 300 {
		t.Log("Expected image width to be 300 but got ", downloadedImage.Bounds().Max.X)
	}

	if downloadedImage.Bounds().Max.Y != 225 {
		t.Log("Expected image width to be 225 but got ", downloadedImage.Bounds().Max.Y)
	}
}
