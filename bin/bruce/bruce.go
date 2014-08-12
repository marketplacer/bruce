package main

import (
	"log"
	"net/http"

	"github.com/exchangegroup/bruce/webserver"
)

func main() {
	webserver.ImageDir = "/tmp"
	r := webserver.Router()
	http.Handle("/", r)
	log.Panicln(http.ListenAndServe(":8901", nil))
}
