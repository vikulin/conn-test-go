package main

import (
	"flag"
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

	startServer := flag.Bool("s", false, "server")
	startClient := flag.Bool("c", false, "client")
	bufferLenght := flag.Int("b", 4096, "buffer")
	//totalSize := flag.Int("t", 1073741824, "total")
	totalSize := flag.Int("t", 40960, "total")
	flag.Parse()

	if *startServer {
		// start the server
		go func() {
			hasher := hash.New()
			ln, err := udt.Listen("udp", ":8081")
			if err != nil {
				panic(err)
			}

			fmt.Println("Waiting for incoming connection")
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			fmt.Println("Established connection")
			parts := *totalSize % *bufferLenght
			tail := *totalSize - *bufferLenght * parts
			for i := 0; i < parts; i++ {
				data := random_bytes(*bufferLenght)
				hasher.Write(data)
				conn.Write(data)
			}
			if tail > 0 {
				data := random_bytes(tail)
				hasher.Write(data)
				conn.Write(data)
			}
			fmt.Printf("Uploaded data. size: %d, hash: %x\n", *totalSize, hasher.Sum64())
			//fmt.Printf("RAW data %x\n", data)
		}()
	}

	if *startClient {
		// run the client
		go func() {
			conn, err := udt.Dial("localhost:8081")
			if err != nil {
				panic(err)
			}

			// listen for reply
			data := make([]byte, *bufferLenght)
			
			hasher := hash.New()
			
			parts := *totalSize % *bufferLenght
			tail := *totalSize - *bufferLenght * parts
			for i := 0; i < parts; i++ {
				n, err := read_conn(conn, data)
				if err != nil {
					panic(err)
				}
				hasher.Write(data[:n])
				conn.Write(data[:n])
			}
			if tail > 0 {
				data := make([]byte, tail)
				n, err := read_conn(conn, data)
				if err != nil {
					panic(err)
				}
				hasher.Write(data[:n])
				conn.Write(data[:n])
			}
			fmt.Printf("Uploaded data. size: %d, hash: %x\n", *totalSize, hasher.Sum64())
			//fmt.Printf("RAW data %x\n", data)
		}()
	}

	time.Sleep(time.Hour)
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
