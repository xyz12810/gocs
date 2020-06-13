package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	switch os.Args[1] {
	case "udp":
		echoUDP()
	case "file":
		echoFILE()
	case "data":
		echoHTTP()
	case "proxy":
		runProxy()
	}
}
func runTCP() {

	l, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}
	var received uint64
	var privous uint64
	go func() {
		for {
			time.Sleep(time.Second)
			if privous < 1 {
				privous = received
				continue
			}
			show := fmt.Sprintf("%vB/%v", received-privous, time.Second)
			if (received-privous)/1024/1024 > 0 {
				show = fmt.Sprintf("%vMB/%v", (received-privous)/1024/1024, time.Second)
			} else if (received-privous)/1024 > 0 {
				show = fmt.Sprintf("%vKB/%v", (received-privous)/1024, time.Second)
			}
			fmt.Printf("received:%v,%v\n", received, show)
			privous = received
		}
	}()
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			buf := make([]byte, 1024*1024)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					break
				}
				atomic.AddUint64(&received, uint64(n))
				conn.Write(buf[0:n])
				// fmt.Printf("recv(%v):%v\n", n, string(buf[:n]))
			}
		}()
	}
}

func echoUDP() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 53})
	if err != nil {
		panic(err)
	}
	for {
		buf := make([]byte, 10240)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			break
		}
		fmt.Printf("recv %v:%v\n", n, buf[:n])
	}
}

func echoFILE() {
	f, err := os.Open(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	buf := make([]byte, 1024)
	for {
		n, err := f.Read(buf)
		if err != nil {
			break
		}
		fmt.Printf("%v\n", buf[0:n])
	}
}

func echoDataH(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("echo data from %v\n", r.RemoteAddr)
	fmt.Fprintf(w, "%v", r.URL.Query().Get("data"))
}

func echoHTTP() {
	http.HandleFunc("/test", echoDataH)
	// http.ListenAndServe(":8070", nil)
	fmt.Println(http.ListenAndServeTLS(":8070", "server.crt", "server.key", nil))
}

func runProxy() {
	l, err := net.Listen("tcp", os.Args[2])
	if err != nil {
		panic(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			remote, err := net.Dial("tcp", os.Args[3])
			if err != nil {
				conn.Close()
				return
			}
			go func() {
				io.Copy(conn, remote)
				conn.Close()
			}()
			io.Copy(remote, conn)
			remote.Close()
		}()
	}
}
