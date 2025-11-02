package client

import (
	"net/url"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"

	"Viz/internal/session"
)

func connect(serverURL string, log *zap.Logger) (*websocket.Conn, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		log.Error("Parse server url error: ", zap.Error(err))
		return nil, err
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
		log.Error("Couldn't estabilished conntecion: ", zap.Error(err))
		return nil, err
	}

	log.Info("Connected to server: ", zap.String("url", u.String()))
	return conn, err
}

func StartCall(serverURL string, log *zap.Logger) error {
	conn, err := connect(serverURL, log)
	if err != nil {
		log.Error("Failed connect to server url", zap.Error(err))
		return err
	}

	if err := session.StartSession(conn, log); err != nil {
		log.Error("Error in session", zap.Error(err))
		return err
	}

	return nil
}
