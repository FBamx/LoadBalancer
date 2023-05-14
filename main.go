package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port           string
	rounRobinCount int
	servers        []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {

	return &LoadBalancer{
		port:           port,
		rounRobinCount: 0,
		servers:        servers,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.rounRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.rounRobinCount++
		server = lb.servers[lb.rounRobinCount%len(lb.servers)]
	}
	lb.rounRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targeServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targeServer.Address())
	targeServer.Serve(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.google.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.baidu.com"),
	}

	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)

}
