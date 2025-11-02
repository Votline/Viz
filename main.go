package main

import (
	"fmt"
	"sync"
	"flag"
	"context"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"
	"go.uber.org/zap/zapcore"
	"github.com/gordonklaus/portaudio"

	"Viz/internal/client"
	"Viz/internal/server"
)

var (
	once sync.Once
	onceErr error
)

func setupLog() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var logLevel string
	flag.StringVar(&logLevel, "level", "info", "set log level\ninfo/warn/fatal/100 or ignore")
	flag.Parse()

	switch logLevel {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case "100", "ignore":
		cfg.Level = zap.NewAtomicLevelAt(100)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
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

	if err := srv.ListenAndServe(); err != nil && (choice == "server" || choice == "srv") {
		log.Fatal("HTTPS server failed: ", zap.Error(err))
	}

	if choice == "client" || choice == "clt" {
		var url string
		fmt.Scanln(&url)
		if err := client.StartCall(url, log); err != nil {
			log.Fatal("Error in StartCall client", zap.Error(err))
			return
		}
	}
}
