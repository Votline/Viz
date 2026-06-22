// Package server run http server annd upgrade connection to websocket
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"Viz/internal/session"
)

func Setup(port string, log *zap.Logger) (*http.Server, error) {
	const op = "server.Run"

	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	mux, err := routing(log, upgrader)
	if err != nil {
		return nil, fmt.Errorf("%s: Error in routing: %w", op, err)
	}

	srv := &http.Server{
		Handler:      mux,
		Addr:         ":" + port,
		ReadTimeout:  28 * time.Second,
		WriteTimeout: 28 * time.Second,
		IdleTimeout:  28 * time.Second,
	}

	return srv, nil
}

func routing(log *zap.Logger, upg *websocket.Upgrader) (*http.ServeMux, error) {
	const op = "server.routing"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("WS handshake starting",
			zap.String("op", op))

		conn, err := upg.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w,
				"Could not upgrade to websocket protocol: "+err.Error(),
				http.StatusInternalServerError)
			log.Error("WS upgrade failed: ",
				zap.String("op", op),
				zap.Error(err))
			return
		}
		defer conn.Close()
		log.Info("WS connection established",
			zap.String("op", op))

		if err := session.StartSession(conn, log); err != nil {
			log.Error("Error in session",
				zap.String("op", op),
				zap.Error(err))
		}

		log.Info("WS connection closed",
			zap.String("op", op))
	})

	return mux, nil
}
