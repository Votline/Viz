package client

import (
	"sync"
	"time"
	"net/url"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"Viz/internal/audio"
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

	var wg sync.WaitGroup
	wg.Add(2)

	go func(){
		defer wg.Done()
		c.audioStream.RecordStream()
	}()
	go func(){
		defer wg.Done()
		c.audioStream.PlayStream()
	}()

	go func(){
		for {
			select{
			case voiceChunk := <-c.audioStream.VoiceChan:
				if err := c.conn.WriteMessage(websocket.BinaryMessage, voiceChunk); err != nil {
					c.log.Error("WS client write error: ", zap.Error(err))
					return
				}
			default:
				time.Sleep(1*time.Microsecond)
			}
		}
	}()
	
	go func(){
		for {
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				c.log.Error("WS client read error: ", zap.Error(err))
				return
			}
			c.audioStream.AudioQueue.Push(msg)
		}
	}()

	wg.Wait()
}
