package client

import (
	"sync"
	"time"
	"net/url"
	"context"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/audio"
	"Viz/internal/encryptor"
)

type Client struct {
	log *zap.Logger
	conn *websocket.Conn
	audioStream *audio.AudioStream
}
func NewClient(log *zap.Logger) (*Client, error) {
	audioStream, err := audio.NewAS(log)
	if err != nil {
		log.Error("Couldn't create audioStream for client: ", zap.Error(err))
		return nil, err
	}
	return &Client{
		log: log,
		audioStream: audioStream,
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
		Host: parsed.Host,
		Path: "/ws",
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

	enc, err := encryptor.Setup(c.log, c.conn)
	if err != nil {
		c.log.Error("Failed to create encryptor: ", zap.Error(err))
	}


	go c.audioStream.RecordStream()
	go c.audioStream.PlayStream()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			select{
			case <-ctx.Done():
				return
			case voiceChunk := <-c.audioStream.VoiceChan:
				encChunk := enc.Encrypt(voiceChunk)
				if err := c.conn.WriteMessage(websocket.BinaryMessage, encChunk); err != nil {
					c.log.Error("WS client write error: ", zap.Error(err))
					cancel()
					return
				}
			default:
				time.Sleep(1*time.Microsecond)
			}
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for {
			select{
			case <-ctx.Done():
				return
			default:
				_, msg, err := c.conn.ReadMessage()
				if err != nil {
					c.log.Error("WS client read error: ", zap.Error(err))
					cancel()
					return
				}
				decMsg, err := enc.Decrypt(msg)
				if err != nil {
					c.log.Error("Decrypt msg error: ", zap.Error(err))
				}
				c.audioStream.AudioQueue.Push(decMsg)
			}
		}
	}()

	wg.Wait()
}
