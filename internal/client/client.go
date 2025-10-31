package client

import (
	"sync"
	"time"
	"context"
	"net/url"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case voiceChunk := <-c.as.VoiceChan:
				byteData := make([]byte, len(voiceChunk)*2)
				for i, sample := range voiceChunk {
					byteData[i*2] = byte(sample & 0xFF)
					byteData[i*2+1] = byte((sample >> 8) & 0xFF)
				}
				if err := c.conn.WriteMessage(websocket.BinaryMessage, byteData); err != nil {
					c.log.Error("WS client write error: ", zap.Error(err))
					cancel()
					return
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
				pcmData := make([]int16, len(msg)/2)
				for i := 0; i < len(pcmData); i++ {
					pcmData[i] = int16(msg[i*2]) | (int16(msg[i*2+1]) << 8)
				}
				c.as.Queues.Push(pcmData, c.as.Queues.AQ)
			}
		}
	}()

	wg.Wait()
}
