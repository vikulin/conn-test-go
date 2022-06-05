package main

import (
	"flag"
	"fmt"
	"net"
	"io"
	"time"
	mrand "math/rand"

	/**QUIC related imports**/
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	/**QUIC related imports**/

	hash "github.com/zeebo/xxh3"

	udt "github.com/vikulin/udt-conn"
	quic "github.com/vikulin/quic-conn"
	kcpconn "github.com/xtaci/kcp-go/v5"
)

func main() {

	host := flag.String("h", "127.0.0.1:48000", "host")
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
				fmt.Printf("Listening TCP...")
			case "udt":
				ln, err = udt.Listen("udp", *host)
				fmt.Printf("Listening UDT...")
			case "quic":
				//generate tls config
				tlsConf := generateTLSConfig()
				ln, err = quic.Listen("udp", *host, tlsConf)
				fmt.Printf("Listening QUIC...")
			case "kcp":
				ln, err = kcpconn.Listen(*host)
				fmt.Printf("Listening KCP...")
			default:
				
			}
			
			if err != nil {
				panic(err)
			}

			/**
			Listening for a connection
			**/
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("OK\n")
			}
			parts := *totalSize / *bufferLenght
			tail := *totalSize - *bufferLenght * parts

			upload(conn, parts, tail, *totalSize, hasher, *bufferLenght)
			
			fmt.Printf("Done\n")
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
				//conn, err = udt.Dial(*host)
			case "quic":
				tlsConf := &tls.Config{
					InsecureSkipVerify: true,
					NextProtos:   []string{"quic-conn-test"},
				}
				conn, err = quic.Dial(*host, tlsConf)
			case "kcp":
				fmt.Printf("Dialling...")
				conn, err = kcpconn.Dial(*host)
				//workaround for https://github.com/xtaci/kcp-go/issues/225
				conn.Write([]byte("."))
			default:
				
			}
			
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("OK\n")
			}

			parts := *totalSize / *bufferLenght
			tail := *totalSize - *bufferLenght * parts
		
			download(conn, parts, tail, *totalSize, hasher, data)
			
			fmt.Printf("Done\n")
			conn.Close()
			fmt.Printf("Downloaded data. size: %d, hash: %x\n", *totalSize, hasher.Sum64())
			fmt.Printf("Parts: %d, tail: %x\n", parts, tail)
			//fmt.Printf("RAW data %x\n", data)
		}()
	}

	time.Sleep(time.Hour)
}

func upload(conn net.Conn, parts int, tail int, total int, hasher *hash.Hasher, l int) {

	defer elapsed_time(time.Now(), total, "upload")
	
	for i := 0; i < parts; i++ {
		data := random_bytes(l)
		hasher.Write(data)
		conn.Write(data)
		//fmt.Println("part: %d, size: %d", i, len(data))
	}
	if tail > 0 {
		data := random_bytes(tail)
		hasher.Write(data)
		conn.Write(data)
	}
}

func download(conn net.Conn, parts int, _ int, total int, hasher *hash.Hasher, data []byte) {

	defer elapsed_time(time.Now(), total, "download")
	i:=0
	t:=0
	for {
		n, err := read_conn(conn, data)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if n<0 {
			break
		} else {
			t = t + n
			i++
		}
		//fmt.Println("part: %d, size: %d", i, t)
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
	mrand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	mrand.Read(b)
	return b
}

func elapsed_time(start time.Time, total int, name string) {
	elapsed := time.Since(start)
	elapsed_seconds:=float64(elapsed/time.Second)
	speed := float64(total)/(float64(1024*1024)*elapsed_seconds) /**MB/sec**/
	fmt.Printf("%f MB/sec\n", speed)
	fmt.Printf("%s - %s time\n", elapsed, name)
}

//QUIC infrastructure
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(crand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(crand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-conn-test"},
	}
}

