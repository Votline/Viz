package main

import (
	"context"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"

	"Viz/internal/routers"
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = ""
	log, _ := cfg.Build()

	ctx := context.Background()
	defer opus.CloseWasmContext(ctx)

	srv := routers.Setup(log)
	if err := srv.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem");err != nil{
		log.Fatal("HTTPS server failed: ", zap.Error(err))
	}
}
