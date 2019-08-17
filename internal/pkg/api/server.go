package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/datawire/teleproxy/internal/pkg/dns"
	"github.com/datawire/teleproxy/internal/pkg/interceptor"
	"github.com/datawire/teleproxy/internal/pkg/route"
)

type APIServer struct {
	listener net.Listener
	server   http.Server
}

func NewAPIServer(iceptor *interceptor.Interceptor) (*APIServer, error) {
	handler := http.NewServeMux()
	tables := "/api/tables/"
	handler.HandleFunc(tables, func(w http.ResponseWriter, r *http.Request) {
		table := r.URL.Path[len(tables):]

		switch r.Method {
		case http.MethodGet:
			result := iceptor.Render(table)
			if result == "" {
				http.NotFound(w, r)
				return
			}
			w.Write(append([]byte(result), '\n'))
		case http.MethodPost:
			d := json.NewDecoder(r.Body)
			var table []route.Table
			err := d.Decode(&table)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, t := range table {
				iceptor.Update(t)
			}
			dns.Flush()
		case http.MethodDelete:
			iceptor.Delete(table)
		}
	})
	handler.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		var paths []string
		switch r.Method {
		case http.MethodGet:
			paths = iceptor.GetSearchPath()
			result, err := json.Marshal(paths)
			if err != nil {
				// The only way that `json.Marshal` should ever error is with
				// unsupported types, or types with a custom `.MarshalJSON()` that
				// validates the data first.  Because we call it on a `[]string`, it
				// should never error here.
				panic(err)
			}
			w.Write(result)
		case http.MethodPost:
			d := json.NewDecoder(r.Body)
			err := d.Decode(&paths)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			iceptor.SetSearchPath(paths)
		}
	})
	handler.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Goodbye!\n"))
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			// os.FindProcess never fails on Unix systems.
			// I guess we might be in for some trouble if
			// we ever port to Windows?
			panic(err)
		}
		p.Signal(os.Interrupt)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	return &APIServer{
		listener: ln,
		server: http.Server{
			Handler: handler,
		},
	}, nil
}

func (a *APIServer) Port() string {
	return strconv.Itoa(a.listener.Addr().(*net.TCPAddr).Port)
}

func (a *APIServer) Start() {
	go func() {
		if err := a.server.Serve(a.listener); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Printf("API Server: %v", err)
		}
	}()
}

func (a *APIServer) Stop() {
	if err := a.server.Shutdown(context.Background()); err != nil {
		// Error from closing listeners, or context timeout:
		log.Printf("API Server Shutdown: %v", err)
	}
}
