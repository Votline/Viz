package main

import (
	"go.uber.org/zap"

	"Viz/internal/routers"
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = ""
	log, _ := cfg.Build()

	srv := routers.Setup(log)
	if err := srv.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem");err != nil{
		log.Fatal("HTTPS server failed: ", zap.Error(err))
	}
}
