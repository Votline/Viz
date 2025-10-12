package routers

import (
	"time"
	"net/http"

	"go.uber.org/zap"

	"Viz/internal/audio"
)

func Setup(log *zap.Logger) *http.Server {
	mux := routing(log)

	srv := &http.Server{
		Handler: mux,
		Addr: ":8443",
		ReadTimeout: 28*time.Second,
		WriteTimeout: 28*time.Second,
		IdleTimeout: 28*time.Second,
	}

	return srv
}

func routing(log *zap.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		log.Info("Starting")
		audio.Start(log)
		log.Info("Done")
	})

	return mux
}
