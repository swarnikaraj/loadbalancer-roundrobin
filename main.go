package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

func newSimpleServer(address string) *simpleServer {
	serverURL, err := url.Parse(address)
	if err != nil {
		panic(err)
	}
	return &simpleServer{
		address: address,
		proxy:   httputil.NewSingleHostReverseProxy(serverURL),
	}
}

type LoadBalancer struct {
	servers          []Server
	roundRobinCount  int
	port             string
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		servers:         servers,
		roundRobinCount: 0,
	}
}

func (s *simpleServer) Address() string {
	return s.address
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	numServers := len(lb.servers)
	if numServers == 0 {
		return nil
	}

	for i := 0; i < numServers; i++ {
		index := (lb.roundRobinCount + i) % numServers
		server := lb.servers[index]
		if server.IsAlive() {
			lb.roundRobinCount = (index + 1) % numServers
			return server
		}
	}

	// If no server is available, return nil.
	return nil
}

func (lb *LoadBalancer) ServeProxy(w http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	if targetServer == nil {
		http.Error(w, "No available server", http.StatusServiceUnavailable)
		return
	}

	fmt.Printf("Forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(w, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.google.com/"),
		newSimpleServer("https://www.baidu.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://search.yahoo.com"),
		newSimpleServer("https://duckduckgo.com"),
	}

	lb := NewLoadBalancer("8080", servers)

	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.ServeProxy(w, r)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Printf("Listening on port %s\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
