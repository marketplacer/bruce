package main

import (
	"fmt"
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

	fmt.Printf("%25s %-s\n", "IMAGE_DIR", webserver.ImageDir)
	fmt.Printf("%25s %-s\n", "BUCKET", webserver.BucketName)
	fmt.Printf("%25s %-s\n", "AWS_ACCESS_KEY_ID", os.Getenv("AWS_ACCESS_KEY_ID"))
	fmt.Printf("%25s %-s\n", "AWS_SECRET_ACCESS_KEY", os.Getenv("AWS_SECRET_ACCESS_KEY"))
	fmt.Printf("%25s %-s\n", "PORT", "8901")
	r := webserver.Router()
	http.Handle("/", r)
	log.Panicln(http.ListenAndServe(":8901", nil))
}
