// Package client connect to server and start session
package client

import (
	"fmt"
	"net"

	"go.uber.org/zap"

	"Viz/internal/session"
)

func Run(serverURL string, log *zap.Logger) error {
	const op = "client.Run"

	rAddr, err := net.ResolveUDPAddr("udp", serverURL)
	if err != nil {
		return fmt.Errorf("%s: resolve udp addr: %w", op, err)
	}
	conn, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		return fmt.Errorf("%s: dial udp: %w", op, err)
	}
	defer conn.Close()

	log.Debug("Dialed UDP connection",
		zap.String("op", op))

	if err := session.StartSession(conn, false, log); err != nil {
		return fmt.Errorf("%s: start session: %w", op, err)
	}

	return nil
}
