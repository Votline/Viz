package main

import (
	"context"
	"flag"
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
	"github.com/jj11hh/opus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"Viz/internal/client"
	"Viz/internal/server"
)

var (
	once sync.Once
	onceErr error
)

func setupLog() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	
	var logLevel string
	flag.StringVar(&logLevel, "level", "info", "set log level\ninfo/warn/fatal/100 or ignore")
	flag.Parse()

	switch logLevel {
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "fatal":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "100", "ignore":
		cfg.Level = zap.NewAtomicLevelAt(100)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

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
