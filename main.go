package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

// AddBackend to the server pool
func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

// NextIndex atomically increments the current index and returns it
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, 1) % uint64(len(s.backends)))
}

// GetNextBackend returns the next backend to send a request to
func (s *ServerPool) GetNextBackend() *Backend {
	next := s.NextIndex()
	return s.backends[next]
}

func main() {
	var serverList string
	var port int
	flag.StringVar(&serverList, "backends", "", "Load balanced backends, use commas to separate")
	flag.IntVar(&port, "port", 3000, "Port to serve")
	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("Please provide one or more backends to load balance")
	}

	servers := strings.Split(serverList, ",")

	var serverPool ServerPool
	for _, s := range servers {
		serverURL, err := url.Parse(s)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverURL)
		serverPool.AddBackend(&Backend{
			URL:          serverURL,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Printf("Added backend: %s", serverURL)
	}

	server := http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := serverPool.GetNextBackend()
			if peer != nil {
				log.Printf("Forwarding request to: %s", peer.URL)
				peer.ReverseProxy.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Service not available", http.StatusServiceUnavailable)
		}),
	}

	log.Printf("Load Balancer started at :%d\n", port)
	log.Printf("Load balancing to servers: %v", servers)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
