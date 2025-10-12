package routers

import (
	"time"
	"net/http"
)

func Setup() *http.Server {
	r := http.NewServeMux()

	srv := &http.Server{
		Handler: r,
		Addr: ":8443",
		ReadTimeout: 28*time.Second,
		WriteTimeout: 28*time.Second,
		IdleTimeout: 28*time.Second,
	}

	return srv
}
