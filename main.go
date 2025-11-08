package main

import (
	"os"
	"fmt"
	"sync"
	"flag"
	"bufio"
	"context"
	"strings"

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
	
	fmt.Printf("Enter mode (server/srv or client/clt): ")
	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Failed to read VIZ mode", zap.Error(err))
	}
	choice = strings.TrimSpace(choice)

	if choice == "server" || choice == "srv" {
		fmt.Print("Enter server port(default 8080): ")
		port, err := reader.ReadString('\n')
		port = strings.TrimSpace(port)
		if err != nil {
			log.Warn("Failed to read server port, default 8080")
			port = "8080"
		}
		srv, err := server.Setup(port, log)
		if err != nil {
			log.Fatal("Couldn't create server: ", zap.Error(err))
		}

		if err := srv.ListenAndServe(); err != nil && (choice == "server" || choice == "srv") {
			log.Fatal("HTTPS server failed: ", zap.Error(err))
		}
	}
	
	if choice == "client" || choice == "clt" {
		fmt.Printf("Enter server URL: ")
		url, err := reader.ReadString('\n')
		url = strings.TrimSpace(url)
		if err != nil {
			log.Fatal("Failed to read server URL", zap.Error(err))
		}
		if err := client.StartCall(url, log); err != nil {
			log.Fatal("Error in StartCall client", zap.Error(err))
			return
		}
	}
}
