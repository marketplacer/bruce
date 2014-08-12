package webserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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
	file := "../bruce.jpg"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	fw, err := w.CreateFormFile("image", file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(fw, f); err != nil {
		t.Fatal(err)
	}

	if fw, err = w.CreateFormFile("file", file); err != nil {
		t.Fatal(err)
	}
	if _, err = fw.Write([]byte("KEY")); err != nil {
		t.Fatal(err)
	}

	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	return client.Do(req)
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

func TestFileDownload(t *testing.T) {
	ts := httptest.NewServer(webserver.Router())
	defer ts.Close()
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

	resp, err = http.Get(imageResponse.URL)
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
