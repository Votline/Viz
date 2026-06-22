// Package main parses cli arguments and starts the server or client
package main

import (
	"fmt"
	"os"
	"slices"

	"Viz/internal/client"
	"Viz/internal/server"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const helpMsg = `
Usage: (choose your way):
	1. Run as server: ./viz -s <port> <args>
	2. Run as client: ./viz -c <url> <args>

Arguments:
	'-d' or 'debug' to enable debug mode
	'-h' or 'help' to print this message
`

func main() {
	args := os.Args[1:]
	if len(args) < 2 || args[0] == "-h" || args[0] == "help" {
		fmt.Print(helpMsg)
		return
	}

	dbg := slices.Contains(args, "-d") || slices.Contains(args, "debug")
	log := initLog(dbg)

	switch args[0] {
	case "-s":
		port := args[1]
		srv, err := server.Setup(port, log)
		if err != nil {
			log.Fatal("Setup server error: ", zap.Error(err))
		}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("Run server error: ", zap.Error(err))
		}
	case "-c":
		url := args[1]
		if err := client.Run(url, log); err != nil {
			log.Fatal("Run client error: ", zap.Error(err))
		}
	default:
		fmt.Println("Wrong arguments. Use -h or help to print help message")
	}
}

func initLog(dbg bool) *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = ""
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.ConsoleSeparator = " | "
	cfg.Level.SetLevel(zap.ErrorLevel)

	if dbg {
		cfg.Level.SetLevel(zap.DebugLevel)
	}
	log, _ := cfg.Build()

	return log
}
