package main

import (
	"fmt"

	"sync"
	"context"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/server"
	"Viz/internal/client"
)

var (
	once sync.Once
	onceErr error
)

func setupLog() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = ""
	log, _ := cfg.Build()
	return log
}


func main() {
	log := setupLog()

	ctx := context.Background()
	once.Do(func(){
		if onceErr = portaudio.Initialize(); onceErr != nil {
			log.Fatal("PortAudio init error: ", zap.Error(onceErr))
			return
		}
	})

	defer func() {
		opus.CloseWasmContext(ctx)
		portaudio.Terminate()
	}()
	
	var choice string
	fmt.Scanln(&choice)

	srv, err := server.Setup(log)
	if err != nil {
		log.Fatal("Couldn't create server: ", zap.Error(err))
	}

	clt, err := client.NewClient(log)
	if err != nil {
		log.Fatal("Couldn't create client: ", zap.Error(err))
	}

	if err := srv.ListenAndServe(); err != nil && (choice == "sevrer" || choice == "srv") {
		log.Fatal("HTTPS server failed: ", zap.Error(err))
	}

	if choice == "client" || choice == "clt" {
		var url string
		fmt.Scanln(&url)
		clt.StartCall(url)
	}
}
