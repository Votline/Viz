package server

import (
	"time"
	"net/http"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/session"
)

func Setup(port string, log *zap.Logger) (*http.Server, error) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	mux, err := routing(log, upgrader)
	if err != nil {
		log.Error("Couldn't create handler: ", zap.Error(err))
		return nil, err
	}

	srv := &http.Server{
		Handler:      mux,
		Addr:         ":"+port,
		ReadTimeout:  28 * time.Second,
		WriteTimeout: 28 * time.Second,
		IdleTimeout:  28 * time.Second,
	}

	return srv, nil
}

func routing(log *zap.Logger, upg *websocket.Upgrader) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("use 'url/ws'"))
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Info("WS handshake starting")

		conn, err := upg.Upgrade(w, r, nil)
		if err != nil {
			log.Error("WS upgrade failed: ", zap.Error(err))
			return
		}
		defer conn.Close()
		log.Info("WS connection established")
		
		if err := session.StartSession(conn, log); err != nil {
			log.Error("Error in session", zap.Error(err))
		}

		log.Info("WS connection closed")
	})

	return mux, nil
}
