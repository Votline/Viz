package routers

import (
	"log"
	"time"
	"net/http"

	"Viz/internal/audio"
)

func Setup() *http.Server {
	mux := routing()

	srv := &http.Server{
		Handler: mux,
		Addr: ":8443",
		ReadTimeout: 28*time.Second,
		WriteTimeout: 28*time.Second,
		IdleTimeout: 28*time.Second,
	}

	return srv
}

func routing() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		log.Println("Starting")
		audio.Start()
		log.Println("Done")
	})

	return mux
}
