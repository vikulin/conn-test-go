package main

import (
	"flag"
	"fmt"
	"net"
	"io"
	"log"
	"time"
	"strconv"
	"strings"
	"syscall"

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

	//udt "github.com/vikulin/udt-conn"
	//udt "github.com/jbenet/go-udtwrapper/udt"
	quic "github.com/vikulin/quic-conn"
	//sctp "github.com/ishidawataru/sctp"
	sctp "github.com/vikulin/sctp"
	sctp_ti "github.com/thebagchi/sctp-go"
	sctp_ce "git.cs.nctu.edu.tw/calee/sctp"
	kcp "github.com/xtaci/kcp-go/v5"
)

func main() {

	host := flag.String("h", "127.0.0.1:48000", "host")
	startServer := flag.Bool("s", false, "server")
	startClient := flag.Bool("c", false, "client")
	bufferLenght := flag.Int("b", 4096, "buffer")
	proto := flag.String("r", "tcp", "proto")
	totalSize := flag.Int("t", 40960, "total")
	doUpload := flag.Bool("u", false, "upload")
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
				//ln, err = udt.Listen(*host)
				fmt.Printf("Listening UDT...")
			case "quic":
				//generate tls config
				tlsConf := generateTLSConfig()
				ln, err = quic.Listen("udp", *host, tlsConf)
				fmt.Printf("Listening QUIC...")
			case "sctp":
				addr := getAddr(*host)
				ln, err = sctp.NewSCTPListener(addr, sctp.InitMsg{NumOstreams: 255, MaxInstreams: 255}, sctp.OneToOne, false)
				ln.(*sctp.SCTPListener).SetEvents(sctp.SCTP_EVENT_DATA_IO)
				fmt.Printf("Listening SCTP...")
			case "sctp_ti":
				addr := getAddrTi(*host)
				ln, err = sctp_ti.ListenSCTP(
					"sctp4",
					syscall.SOCK_STREAM,
					addr,
					&sctp_ti.SCTPInitMsg{
						NumOutStreams:  100,
						MaxInStreams:   100,
						MaxAttempts:    0,
						MaxInitTimeout: 0,
					},
				)
				fmt.Printf("Listening SCTP...")
			case "sctp_ce":
				addr := getAddrCe(*host)
				ln, err = sctp_ce.ListenSCTP("sctp", addr)
				fmt.Printf("Listening SCTP...")
			case "kcp":
				ln, err = kcp.Listen(*host)
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

			if(*proto=="sctp_ce"){
				conn = sctp_ce.NewSCTPSndRcvInfoWrappedConn(conn.(*sctp_ce.SCTPConn))
			}
			parts := *totalSize / *bufferLenght
			tail := *totalSize - *bufferLenght * parts
			
			if *doUpload {
				upload(conn, parts, tail, *totalSize, hasher, *bufferLenght)
			} else {
				download(conn, parts, tail, *totalSize, hasher, *bufferLenght)
			}
			
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
			hasher := hash.New()
			fmt.Printf("Used hash: %T\n", *hasher)
			var conn net.Conn
			var err error
			/**
			Started dialing
			**/
			switch *proto {
			case "tcp":
				fmt.Printf("Dialling TCP...")
				conn, err = net.Dial("tcp", *host)
			case "udt":
				//conn, err = udt.Dial("0.0.0.0:0", *host)
			case "quic":
				fmt.Printf("Dialling QUIC...")
				tlsConf := &tls.Config{
					InsecureSkipVerify: true,
					NextProtos:   []string{"quic-conn-test"},
				}
				conn, err = quic.Dial(*host, tlsConf)
			case "sctp":
				fmt.Printf("Dialling SCTP...")
				addr := getAddr(*host)
				
				conn, err = sctp.NewSCTPConnection(addr.AddressFamily, sctp.InitMsg{NumOstreams: 255, MaxInstreams: 255}, sctp.OneToOne, false)
				if err != nil {
					panic("failed to dial: " + err.Error())
				}
				//conn.(*sctp.SCTPConn).SetEvents(sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_ASSOCIATION)
				if err := conn.(*sctp.SCTPConn).Connect(addr); err != nil {
					panic("failed to dial: " + err.Error())
				}
			case "sctp_ti":
				fmt.Printf("Dialling SCTP...")
				laddr := getAddrTi("0.0.0.0:0")
				raddr := getAddrTi(*host)
				conn, err = sctp_ti.DialSCTP(
					"sctp4",
					laddr,
					raddr,
					&sctp_ti.SCTPInitMsg{
						NumOutStreams:  100,
						MaxInStreams:   100,
						MaxAttempts:    0,
						MaxInitTimeout: 0,
					},
				)
			case "sctp_ce":
				laddr := &sctp_ce.SCTPAddr{
					Port: 0,
				}
				addr := getAddrCe(*host)
				conn, err = sctp_ce.DialSCTP("sctp", laddr, addr)
			case "kcp":
				fmt.Printf("Dialling KCP...")
				conn, err = kcp.Dial(*host)
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
		
			if *doUpload {
				upload(conn, parts, tail, *totalSize, hasher, *bufferLenght)
			} else {
				download(conn, parts, tail, *totalSize, hasher, *bufferLenght)
			}
			
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

func download(conn net.Conn, parts int, _ int, total int, hasher *hash.Hasher, l int) {

	defer elapsed_time(time.Now(), total, "download")
	data := make([]byte, l)
	i:=0
	t:=0
	for {
		n, err := read_conn(conn, data)
		if err != nil {
			panic(err)
		}
		if n<0 || t == total {
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
            if err == io.EOF {
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
	elapsed_seconds := elapsed.Seconds()
	totalMb := float64(total)/(1024.0*1024.0)
	speed := totalMb/elapsed_seconds /**MB/sec**/
	fmt.Printf("Total: %f MB\n", totalMb)
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

//SCTP infrastructure
func getAddr(host string) *sctp.SCTPAddr {
	//sctp supports multihoming but current implementation reuires only one path
	ips := []net.IPAddr{}
	ip, port, err := net.SplitHostPort(host)
	if err != nil {
		panic(err)
	}
	for _, i := range strings.Split(ip, ",") {
		if a, err := net.ResolveIPAddr("ip", i); err == nil {
			log.Printf("Resolved address '%s' to %s", i, a)
			ips = append(ips, *a)
		} else {
			log.Printf("Error resolving address '%s': %v", i, err)
		}
	}
	p, _ := strconv.Atoi(port)
	addr := &sctp.SCTPAddr{
		IPAddrs: ips,
		Port:    p,
	}
	return addr
}

func getAddrTi(host string) *sctp_ti.SCTPAddr{
	addr, err := sctp_ti.MakeSCTPAddr("sctp4", host)
	if nil != err {
		panic(err)
	}
	return addr
}

func getAddrCe(host string) *sctp_ce.SCTPAddr{

	//sctp supports multihoming but current implementation reuires only one path
	ips := []net.IPAddr{}
	ip, port, err := net.SplitHostPort(host)
	if err != nil {
		panic(err)
	}
	for _, i := range strings.Split(ip, ",") {
		if a, err := net.ResolveIPAddr("ip", i); err == nil {
			log.Printf("Resolved address '%s' to %s", i, a)
			ips = append(ips, *a)
		} else {
			log.Printf("Error resolving address '%s': %v", i, err)
		}
	}
	p, _ := strconv.Atoi(port)
	addr := &sctp_ce.SCTPAddr{
		IPAddrs: ips,
		Port:    p,
	}
	return addr
}

