package main

import (
	"log"
	"net/http"
	"os"

	"github.com/exchangegroup/bruce/webserver"
	_ "github.com/joho/godotenv/autoload"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

func main() {
	webserver.ImageDir = os.Getenv("IMAGE_DIR")

	webserver.BucketName = os.Getenv("BUCKET")

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Panicln(err)
	}

	webserver.S3Connection = s3.New(auth, aws.APSoutheast2)
	r := webserver.Router()
	http.Handle("/", r)
	log.Println("Starting bruce on http://localhost:8901")
	log.Panicln(http.ListenAndServe(":8901", nil))
}
