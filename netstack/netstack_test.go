package netstack

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/coversocks/gocs/core"
	"github.com/coversocks/gocs/netstack/pcap"
	"github.com/coversocks/gocs/netstack/tcpip"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
}
func TestUDP(t *testing.T) {
	rawSender, err := pcap.NewReader("testdata/test_udp.pcap")
	if err != nil {
		t.Error(err)
		return
	}
	stackSender := tcpip.NewWaitReader(rawSender, 100*time.Millisecond)
	l := NewNetProxy(core.RawDialerF(func(network, address string) (conn net.Conn, err error) {
		// echo, err := core.NewEchoConn()
		// if err == nil {
		// 	conn = core.NewConnWrapper(tcpip.NewPrintRWC(true, "Dial", echo))
		// }
		conn = core.NewConnWrapper(core.NewCallThroughRC(ioutil.Discard))
		return
	}))
	l.Through = true
	l.Bootstrap("../default-client.json")
	l.SetWriter(tcpip.NewPacketPrintRWC(true, "Out", tcpip.NewWrapRWC(ioutil.Discard), l.Eth))
	in := tcpip.NewPacketPrintRWC(true, "In", tcpip.NewWrapRWC(stackSender), l.Eth)
	err = l.ProcessReader(in)
	l.Close()
	fmt.Printf("proc stop by %v\n", err)
	l.Wait()
}
