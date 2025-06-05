package server

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	HttpServer *http.Server
}

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		HttpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (srv *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer srv.Shutdown()
	defer wg.Done()

	go func() {
		log.Printf("Server starting on %s\n", srv.HttpServer.Addr)
		if err := srv.HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Closing server due to context cancellation...")
			return
		}
	}
}

func (srv *Server) Shutdown() {
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.HttpServer.Shutdown(shutdownCtx); err != nil {
		log.Println("Server Shutdown:", err)
	}
	log.Println("Server gracefully stopped.")
	<-shutdownCtx.Done()
}
