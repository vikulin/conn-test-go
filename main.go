package main

import (
	"flag"
	//"time"
	"fmt"
	"net"
	"io"
	"time"
	"math/rand"
	hash "github.com/zeebo/xxh3"

	udt "github.com/vikulin/udt-conn"
)

func main() {
	// utils.SetLogLevel(utils.LogLevelDebug)

	host := flag.String("h", "localhost:48000", "host")
	startServer := flag.Bool("s", false, "server")
	startClient := flag.Bool("c", false, "client")
	bufferLenght := flag.Int("b", 4096, "buffer")
	proto := flag.String("r", "tcp", "proto")
	//totalSize := flag.Int("t", 1073741824, "total")
	totalSize := flag.Int("t", 40960, "total")
	flag.Parse()

	if *startServer {
		// start the server
		go func() {
			hasher := hash.New()
			fmt.Printf("Used hash: %T\n", *hasher)
			var ln net.Listener
			var err error
			switch *proto {
			case "tcp":
				ln, err = net.Listen("tcp", *host)
			case "udt":
				ln, err = udt.Listen("udp", *host)
			case "quic":
				//
			default:
				
			}
			
			if err != nil {
				panic(err)
			}

			fmt.Println("Waiting for incoming connection")
			/**
			Listening for a connection
			**/
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			fmt.Println("Established connection")
			parts := *totalSize / *bufferLenght
			tail := *totalSize - *bufferLenght * parts

			upload(conn, parts, tail, hasher, *bufferLenght)

			conn.Close()
			fmt.Printf("Uploaded data. size: %d, hash: %x\n", *totalSize, hasher.Sum64())
			fmt.Printf("Parts: %d, tail: %x\n", parts, tail)
			//fmt.Printf("RAW data %x\n", data)
		}()
	}

	if *startClient {
		// run the client
		go func() {
			data := make([]byte, *bufferLenght)
			hasher := hash.New()
			fmt.Printf("Used hash: %T\n", *hasher)
			var conn net.Conn
			var err error
			/**
			Started dialing
			**/
			switch *proto {
			case "tcp":
				conn, err = net.Dial("tcp", *host)
			case "udt":
				conn, err = udt.Dial(*host)
			case "quic":
				//
			default:
				
			}
			
			if err != nil {
				panic(err)
			}

			parts := *totalSize / *bufferLenght
			tail := *totalSize - *bufferLenght * parts
		
			download(conn, parts, tail, hasher, data)
			
			conn.Close()
			fmt.Printf("Downloaded data. size: %d, hash: %x\n", *totalSize, hasher.Sum64())
			fmt.Printf("Parts: %d, tail: %x\n", parts, tail)
			//fmt.Printf("RAW data %x\n", data)
		}()
	}

	time.Sleep(time.Hour)
}

func upload(conn net.Conn, parts int, tail int, hasher *hash.Hasher, l int) {

	defer elapsed_time(time.Now(), "upload")
	
	for i := 0; i < parts; i++ {
		data := random_bytes(l)
		hasher.Write(data)
		conn.Write(data)
	}
	if tail > 0 {
		data := random_bytes(tail)
		hasher.Write(data)
		conn.Write(data)
	}
}

func download(conn net.Conn, parts int, tail int, hasher *hash.Hasher, data []byte) {

	defer elapsed_time(time.Now(), "download")

	for i := 0; i < parts; i++ {
		n, err := read_conn(conn, data)
		if err != nil {
			panic(err)
		}
		hasher.Write(data[:n])
	}
	if tail > 0 {
		data := make([]byte, tail)
		n, err := read_conn(conn, data)
		if err != nil {
			panic(err)
		}
		hasher.Write(data[:n])
	}
}

func read_conn(c net.Conn, buffer []byte) (int, error) {
    for {
        n, err := c.Read(buffer)
        if err != nil {
            if err != io.EOF {
               return -1, nil
            } else {
            	return 0, err
            }
        } else {
        	return n, nil
        }
    }
}

func random_bytes(length int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return b
}

func elapsed_time(start time.Time, name string) {
    elapsed := time.Since(start)
    fmt.Printf("%s - %s\n", name, elapsed)
}
