package main

import (
	"sync"
	"context"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/routers"
)

var (
	once sync.Once
	onceErr error
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = ""
	log, _ := cfg.Build()

	ctx := context.Background()
	once.Do(func(){
		if onceErr := portaudio.Initialize(); onceErr != nil {
			log.Fatal("PortAudio init error: ", zap.Error(onceErr))
			return
		}
	})
	defer func() {
		opus.CloseWasmContext(ctx)
		portaudio.Terminate()
	}()

	srv := routers.Setup(log)
//	if err := srv.ListenAndServeTLS("ssl/cert.pem", "ssl/key.pem");err != nil{
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("HTTPS server failed: ", zap.Error(err))
	}
}
