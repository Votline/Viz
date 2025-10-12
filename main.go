package main

import (
	"log"
	"Viz/internal/routers"
)

func main() {
	srv := routers.Setup()
	log.Fatal(srv.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem"))
}
