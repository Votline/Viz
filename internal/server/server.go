package server

import (
	"sync"
	"time"
	"context"
	"net/http"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
	"Viz/internal/batch"
	//"Viz/internal/encryptor"
)

func Setup(log *zap.Logger) (*http.Server, error) {
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
		Addr:         ":8443",
		ReadTimeout:  28 * time.Second,
		WriteTimeout: 28 * time.Second,
		IdleTimeout:  28 * time.Second,
	}

	return srv, nil
}

func routing(log *zap.Logger, upg *websocket.Upgrader) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	as, err := audio.NewAS(log)
	if err != nil {
		log.Error("Couldn't create audioStream for server: ", zap.Error(err))
		return mux, err
	}

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
		/*
			enc, err := encryptor.Setup(log, conn)
			if err != nil {
				log.Fatal("Failed to create encryptor: ", zap.Error(err))
			}
		*/
		go as.RecordStream()
		go as.PlayStream()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const batchSize = 3
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			batchBuffer := make([][]byte, 0, batchSize)
			for {
				select {
				case voiceChunk := <-as.VoiceChan:
					batchBuffer = append(batchBuffer, voiceChunk)

					if len(batchBuffer) == batchSize {
						packedBatch := batch.PackBatch(batchBuffer)
						if err := conn.WriteMessage(websocket.BinaryMessage, packedBatch); err != nil {
							log.Error("WS server write failed: ", zap.Error(err))
							cancel()
							return
						}
						batchBuffer = batchBuffer[:0]
					}
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()

		wg.Add(1)
		go func() {
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

					frames, err := batch.UnpackBatch(msg)
					if err != nil {
						log.Error("Failed to unpack batch", zap.Error(err))
						continue
					}

					for _, frame := range frames {
						as.Queues.Push(frame, as.Queues.AQ)
					}
				}
			}
		}()

		wg.Wait()
		log.Info("WS connection closed")
	})

	return mux, nil
}
