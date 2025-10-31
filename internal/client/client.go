package client

import (
	"sync"
	"time"
	"context"
	"net/url"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
	"Viz/internal/batch"
	// "Viz/internal/encryptor"
)

type Client struct {
	log  *zap.Logger
	conn *websocket.Conn
	as   *audio.AudioStream
}

func NewClient(log *zap.Logger) (*Client, error) {
	as, err := audio.NewAS(log)
	if err != nil {
		log.Error("Couldn't create audioStream for client: ", zap.Error(err))
		return nil, err
	}
	return &Client{
		log: log,
		as:  as,
	}, nil
}

func (c *Client) connect(serverURL string) error {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		c.log.Error("Parse server url error: ", zap.Error(err))
		return err
	}
	scheme := "ws"
	if parsed.Scheme == "https" {
		scheme = "wss"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   parsed.Host,
		Path:   "/ws",
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		c.log.Error("Couldn't estabilished conntecion: ", zap.Error(err))
		return err
	}

	c.conn = conn
	c.log.Info("Connected to server: ", zap.String("url", u.String()))
	return nil
}

func (c *Client) StartCall(serverURL string) {
	c.connect(serverURL)
	/*
		enc, err := encryptor.Setup(c.log, c.conn)
		if err != nil {
			c.log.Error("Failed to create encryptor: ", zap.Error(err))
		}
	*/

	go c.as.RecordStream()
	go c.as.PlayStream()

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
			case <-ctx.Done():
				return
			case voiceChunk := <-c.as.VoiceChan:
				batchBuffer = append(batchBuffer, voiceChunk)
				
				if len(batchBuffer) == batchSize {
					packedBatch := batch.PackBatch(batchBuffer)
					if err := c.conn.WriteMessage(websocket.BinaryMessage, packedBatch); err != nil {
						c.log.Error("WS client write error: ", zap.Error(err))
						cancel()
						return
					}
					batchBuffer = batchBuffer[:0]
				}
			default:
				time.Sleep(1 * time.Microsecond)
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
				_, msg, err := c.conn.ReadMessage()
				if err != nil {
					c.log.Error("WS client read error: ", zap.Error(err))
					cancel()
					return
				}

				frames, err := batch.UnpackBatch(msg)
				if err != nil {
					c.log.Error("Failed to unpack batch", zap.Error(err))
					continue
				}

				for _, frame := range frames {
					c.as.Queues.Push(frame, c.as.Queues.AQ)
				}
			}
		}
	}()

	wg.Wait()
}
