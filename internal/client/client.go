// Package client connect to server and start session
package client

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"Viz/internal/session"
)

func Run(serverURL string, log *zap.Logger) error {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Error("Error in dialing server", zap.Error(err))
		return err
	}
	defer conn.Close()

	if err := session.StartSession(conn, log); err != nil {
		log.Error("Error in session", zap.Error(err))
		return err
	}

	return nil
}
