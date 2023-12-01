package main

import (
	"flag"
	"log"
	"net"
	"os"
)

func main() {
	parseFlag()
	f, err := os.OpenFile(*side+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("init log error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	if *side == "server" {
		Server()
	}
	if *side == "client" {
		Client()
	}
}

var (
	side          *string
	input         *string
	output        *string
	clientPort    *string
	serverForward *string
)

func parseFlag() {
	side = flag.String("type", "", "server or client")
	input = flag.String("input", "", "connection input file")
	output = flag.String("output", "", "connection ouput file")
	clientPort = flag.String("listen", "", "client listening")
	serverForward = flag.String("forward", "", "client listening")
	flag.Parse()

	if *side == "" || *side != "server" && *side != "client" {
		log.Println("Prefilght check failed: type must be server or client")
		os.Exit(1)
	}
	if *input == "" {
		log.Println("Prefilght check failed: missing input file")
		os.Exit(1)
	}

	if *output == "" {
		log.Println("Prefilght check failed: missing output file")
		os.Exit(1)
	}

	if *side == "client" && *clientPort == "" {
		log.Println("Prefilght check failed: missing client port")
		os.Exit(1)
	}

	if *side == "server" && *serverForward == "" {
		log.Println("Prefilght check failed: missing server forward")
		os.Exit(1)
	}
}

func Client() error {
	listener, err := net.Listen("tcp", ":"+*clientPort)
	if err != nil {
		log.Printf("Client try listen %s failed: %s", *clientPort, err)
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Client accept connection failed %s", err)
			return err
		}
		fileTunnel := NewFileTunnel()
		fileTunnel.conn = &WrapConn{
			Real: conn,
		}
		fileTunnel.Tunnel()
	}
}

func Server() error {
	for {
		fileTunnel := NewFileTunnel()
		fileTunnel.waitFileReady()
		conn, err := net.Dial("tcp", ":"+*serverForward)
		if err != nil {
			log.Printf("listen failed %s", err)
			return err
		}

		fileTunnel.conn = &WrapConn{
			Real: conn,
		}
		fileTunnel.Tunnel()
	}
}
