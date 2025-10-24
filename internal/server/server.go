package server

import (
	"time"
	"net/http"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
)

func Setup(log *zap.Logger) (*http.Server, error) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool{
			return true
		},
	}
	mux, err := routing(log, upgrader)
	if err != nil {
		log.Error("Couldn't create handler: ", zap.Error(err))
		return nil, err
	}

	srv := &http.Server{
		Handler: mux,
		Addr: ":8443",
		ReadTimeout: 28*time.Second,
		WriteTimeout: 28*time.Second,
		IdleTimeout: 28*time.Second,
	}

	return srv, nil
}

func routing(log *zap.Logger, upg *websocket.Upgrader) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	audioStream, err := audio.NewAS(log)
	if err != nil {
		log.Error("Couldn't create audioStream for server: ", zap.Error(err))
		return mux, err
	}

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

		go audioStream.RecordStream()
		go audioStream.PlayStream()
		breakChan := make(chan bool, 2)

		go func(breakChan chan bool){
			for {
				select {
				case voiceChunk := <-audioStream.VoiceChan:
					if err := conn.WriteMessage(websocket.BinaryMessage, voiceChunk); err != nil {
						log.Error("WS server write failed: ", zap.Error(err))
						breakChan <-true
						return
					}
				case <- breakChan:
					return
				default:
					time.Sleep(1*time.Millisecond)
				}
			}
		}(breakChan)
	
		go func(breakChan chan bool){
			for {
				if <-breakChan {
					return
				}

				_, msg, err := conn.ReadMessage()
				if err != nil {
					log.Error("WS server read failed: ", zap.Error(err))
					breakChan <- true
					return
				}
				audioStream.AudioQueue.Push(msg)
			}
		}(breakChan)

		for {
			select {
			case <-breakChan:
				break
			default:
				time.Sleep(1*time.Millisecond)
			}
		}
	})

	return mux, nil
}
