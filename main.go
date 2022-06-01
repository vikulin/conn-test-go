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
	bufferLenght := flag.Int("b", 8, "buffer")
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
			data := random_bytes(*bufferLenght)
			hasher.Write(data)
			conn.Write(data)
			fmt.Printf("Uploaded data. size: %d, hash: %x\n", len(data), hasher.Sum64())
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
			n, err := read_conn(conn, data)
			if err != nil {
				panic(err)
			}
			hasher := hash.New()
			hasher.Write(data[:n])
			fmt.Printf("Downloaded data. size: %d, hash: %x\n", len(data[:n]), hasher.Sum64())
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
