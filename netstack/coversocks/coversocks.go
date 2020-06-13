package coversocks

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"

	"net/http"
	_ "net/http/pprof"

	"github.com/coversocks/gocs/core"
	"github.com/coversocks/gocs/netstack"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
}

var netQueue chan []byte
var netProxy *netstack.NetProxy
var netMessage *core.MessageDialer

type netQueueWriter struct {
}

func (n *netQueueWriter) Write(p []byte) (l int, err error) {
	l = len(p)
	netQueue <- p
	return
}

func (n *netQueueWriter) Close() (err error) {
	return
}

//Bootstrap will bootstrap by config file path, n<0 return is fail, n==0 is success
func Bootstrap(conf string, mtu int) (n int) {
	var err error
	// netReader, netWriter, err = os.Pipe()
	// if err != nil {
	// 	n = -10
	// 	core.ErrorLog("mobile create pipe fail with %v", err)
	// 	return
	// }
	netQueue = make(chan []byte, 10240)
	netMessage = core.NewMessageDialer([]byte("m"), 3*mtu)
	netProxy = netstack.NewNetProxy(conf, uint32(mtu), &netQueueWriter{}, netMessage)
	err = netProxy.Bootstrap()
	if err != nil {
		n = -10
		core.ErrorLog("mobile bootstrap fail with %v", err)
	}
	return
}

//Start the process
func Start() {
	go netProxy.Proc()
}

//Stop the process
func Stop() {

}

//ReadNet read outbound data from the netstack
func ReadNet(b []byte) (n int) {
	// n, err := netReader.Read(b)
	// if err == nil {
	// 	return
	// }
	// core.WarnLog("NetRead read fail with %v", err)
	// if err == io.EOF {
	// 	n = -1
	// } else {
	// 	n = -99
	// }
	select {
	case data := <-netQueue:
		if len(data) > len(b) {
			panic("too large")
		}
		n = copy(b, data)
	default:
		n = 0
	}
	return
}

//WriteNet write inbound data to the netstack
func WriteNet(b []byte) (n int) {
	res := netProxy.DeliverNetworkPacket(b)
	if res {
		n = len(b)
	}
	fmt.Printf("WriteNet %v by % 02x\n", n, b)
	return
}

//ReadMessage queue one message
func ReadMessage(b []byte) (n int) {
	n, err := netMessage.Read(b)
	if err == nil {
		return
	}
	core.WarnLog("ReadMessage read fail with %v", err)
	if err == io.EOF {
		n = -1
	} else {
		n = -99
	}
	return
}

//WriteMessage done messsage
func WriteMessage(b []byte) (n int) {
	n, err := netMessage.Write(b)
	if err == nil {
		return
	}
	core.WarnLog("ReadMessage write fail with %v", err)
	if err == io.EOF {
		n = -1
	} else {
		n = -99
	}
	return
}

func SetTestAddr(addr string) {
	core.ShowAddress = addr
	netstack.ShowAddress = addr
}

//TestWeb will test web request
func TestWeb(url, digest string) {
	var raw net.Conn
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (conn net.Conn, err error) {
				raw, err = net.Dial(network, addr)
				if err == nil {
					conn = core.NewHashConn(raw, true, "web")
				}
				return
			},
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("testWeb new fail with %v\n", err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("testWeb do fail with %v\n", err)
		return
	}
	defer res.Body.Close()
	h := sha1.New()
	_, err = io.Copy(h, res.Body)
	if err != nil {
		fmt.Printf("testWeb copy fail with %v\n", err)
		return
	}
	v := fmt.Sprintf("%x", h.Sum(nil))
	if v != digest {
		fmt.Printf("testWeb done fail by digest not match %v,%v\n", v, digest)
	} else {
		fmt.Printf("testWeb done success by %x\n", v)
	}
	fmt.Printf("testWeb done by %x\n", h.Sum(nil))
	if raw != nil {
		raw.Close()
	}
}
