package server

import (
	"sync"
	"time"
	"context"
	"net/http"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
	"Viz/internal/encryptor"
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

		enc, err := encryptor.Setup(log, conn)
		if err != nil {
			log.Fatal("Failed to create encryptor: ", zap.Error(err))
		}

		go audioStream.RecordStream()
		go audioStream.PlayStream()
		
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		wg.Add(1)
		go func(){
			defer wg.Done()
			for {
				select {
				case voiceChunk := <-audioStream.VoiceChan:
					encChunk := enc.Encrypt(voiceChunk)
					if err := conn.WriteMessage(websocket.BinaryMessage, encChunk); err != nil {
						log.Error("WS server write failed: ", zap.Error(err))
						cancel()
						return
					}
				case <-ctx.Done():
					return
				default:
					time.Sleep(1*time.Millisecond)
				}
			}
		}()

		wg.Add(1)
		go func(){
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, msg, err := conn.ReadMessage()
					if err != nil {
						log.Error("WS server read failed: ", zap.Error(err))
						cancel()
						return
					}
					decMsg, err := enc.Decrypt(msg)
					if err != nil {
						log.Error("Decrypt message error: ", zap.Error(err))
						continue
					}
					audioStream.AudioQueue.Push(decMsg)
				}
			}
		}()

		wg.Wait()
		log.Info("WS connection closed")
	})

	return mux, nil
}
