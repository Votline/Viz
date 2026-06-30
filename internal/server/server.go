// Package server run http server annd upgrade connection to websocket
package server

import (
	"fmt"
	"net"
	"strconv"

	"go.uber.org/zap"

	"Viz/internal/session"
)

func Run(port string, log *zap.Logger) error {
	const op = "server.Setup"

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s: parse port: %w", op, err)
	}

	addr := &net.UDPAddr{
		Port: portInt,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("%s: listen udp: %w", op, err)
	}

	if err := session.StartSession(conn, true, log); err != nil {
		return fmt.Errorf("%s: start session: %w", op, err)
	}

	return nil
}
