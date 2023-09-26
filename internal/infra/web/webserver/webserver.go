package webserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type WebServer struct {
	Router        chi.Router
	Handlers      map[string]http.HandlerFunc
	WebServerPort string
}

func NewWebServer(port string) *WebServer {
	return &WebServer{
		WebServerPort: port,
	}
}

func (server *WebServer) AddHandler(path string, handler http.HandlerFunc) {
	server.Handlers[path] = handler
}

func (server *WebServer) Start() error {
	server.Router.Use(middleware.Logger)
	for path,handle := range server.Handlers {
		server.Router.Handle(path,handle)
	}
	
	if err := http.ListenAndServe(server.WebServerPort, server.Router); err != nil {
		panic(err.Error())
	}
	
	return nil
}