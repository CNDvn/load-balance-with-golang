package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type BackEnd struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*BackEnd
	current  int64
}

// Add Backend to the server pool
func (s *ServerPool) AddBackend(backend *BackEnd) {
	s.backends = append(s.backends, backend)
}

func (s *ServerPool) NextIndex() int64 {
	s.current++
	return s.current % int64(len(s.backends))
}

func (s *ServerPool) GetNextBackend() *BackEnd {
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

	// add server to serverPool
	serverPool := ServerPool{current: -1}
	for _, s := range servers {
		serverUrl, err := url.Parse(s)
		if err != nil {
			log.Fatal("err")
		}

		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		serverPool.AddBackend(&BackEnd{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	server := http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := serverPool.GetNextBackend()
			fmt.Println(peer)
			if peer != nil {
				peer.ReverseProxy.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Service not available", http.StatusServiceUnavailable)
		}),
	}

	log.Printf("Load Balancer started at :%d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
