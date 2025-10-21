package routers

import (
	"time"
	"net/http"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
)

func Setup(log *zap.Logger) *http.Server {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool{
			return true
		},
	}
	mux := routing(log, upgrader)

	srv := &http.Server{
		Handler: mux,
		Addr: ":8443",
		ReadTimeout: 28*time.Second,
		WriteTimeout: 28*time.Second,
		IdleTimeout: 28*time.Second,
	}

	return srv
}

func routing(log *zap.Logger, upg *websocket.Upgrader) *http.ServeMux {
	mux := http.NewServeMux()

	audioStream := audio.NewAS(log)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		w.Write([]byte("use 'url/ws'"))
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request){
		log.Info("WS handshake starting")
		
		conn, err := upg.Upgrade(w, r, nil)
		if err != nil {
			log.Error("WS upgrade failed: ", zap.Error(err))
			return
		}
		defer conn.Close()
		log.Info("WS connection established")

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Error("WS read failed: ", zap.Error(err))
				break
			}
			go audioStream.Start()
		}
	})

	return mux
}
