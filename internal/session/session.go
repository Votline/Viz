package session

import (
	"sync"
	"context"
	
	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
	"Viz/internal/batch"
	"Viz/internal/encryptor"
)

func StartSession(conn *websocket.Conn, log *zap.Logger) error {
	as, err := audio.NewAS(log)
	if err != nil {
		log.Error("Couldn't create audioStream for server: ", zap.Error(err))
		return err
	}

	enc, err := encryptor.Setup(log, conn)
	if err != nil {
		log.Fatal("Failed to create encryptor: ", zap.Error(err))
	}

	go as.RecordStream()
	go as.PlayStream()

	const batchSize = 8
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		batchBuffer := make([][]byte, 0, batchSize)
		for {
			select {
			case voiceChunk := <-as.VoiceChan:
				encChunk := enc.Encrypt(voiceChunk)
				batchBuffer = append(batchBuffer, encChunk)

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
					decFrame, err := enc.Decrypt(frame)
					if err != nil {
						log.Error("Failed to decrypt message", zap.Error(err))
						continue
					}
					as.Queues.Push(decFrame, as.Queues.AQ)
				}
			}
		}
	}()

	wg.Wait()
	return nil
}
