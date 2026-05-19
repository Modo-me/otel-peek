package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"otelpeek/receiver"
)

var (
	port = 0
)

func main() {
	flag.IntVar(&port, "p", 0, "Port to listen on")
	flag.Parse()
	initHandler()
}

func initHandler() {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	http.HandleFunc("/v1/traces", receiver.TraceHandler)
	http.HandleFunc("/v1/logs", receiver.LogHandler)
	http.HandleFunc("/v1/metrics", receiver.MetricHandler)
	fmt.Printf("Listening on http://127.0.0.1:%d\n", port)
	log.Fatal(http.Serve(ln, nil))
}
