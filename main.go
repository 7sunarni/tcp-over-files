package main

import (
	"flag"
	"log"
	"net"
	"os"
)

func main() {
	parseFlag()
	if *side == "server" {
		for {
			Server()
		}
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
		log.Println("type must be server or client")
		os.Exit(1)
	}
	if *input == "" {
		log.Println("missing input file")
		os.Exit(1)
	}

	if *output == "" {
		log.Println("missing output file")
		os.Exit(1)
	}

	if *side == "client" && *clientPort == "" {
		log.Println("missing client port")
		os.Exit(1)
	}

	if *side == "server" && *serverForward == "" {
		log.Println("missing server forward")
		os.Exit(1)
	}
}

func Client() error {
	listener, err := net.Listen("tcp", ":"+*clientPort)
	if err != nil {
		log.Println("listen failed ", err)
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept failed ", err)
			return err
		}
		fileTunnel := NewFileTunnel()
		if fileTunnel == nil {
			log.Println("file tunnel empty")
			return err
		}
		fileTunnel.conn = &WrapConn{
			Real: conn,
		}
		fileTunnel.Tunnel()
	}
}

func Server() error {
	fileTunnel := NewFileTunnel()
	if fileTunnel == nil {
		log.Println("file tunnel empty")
		return nil
	}

	conn, err := net.Dial("tcp", ":"+*serverForward)
	if err != nil {
		log.Printf("listen failed %s", err)
		return err
	}

	fileTunnel.conn = &WrapConn{
		Real: conn,
	}
	fileTunnel.Tunnel()
	return nil
}
