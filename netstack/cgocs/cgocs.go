package main

import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/coversocks/gocs/core"
	"github.com/coversocks/gocs/netstack"
	"github.com/coversocks/gocs/netstack/dns"
	"github.com/coversocks/gocs/netstack/pcap"
	"github.com/coversocks/gocs/netstack/tcpip"

	_ "net/http/pprof"
)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(newBinderLogger())
	runtime.GOMAXPROCS(1)
	go func() {
		http.HandleFunc("/debug/state/", stateH)
		http.HandleFunc("/debug/test/", testWebH)
		log.Println(http.ListenAndServe(":6060", nil))
	}()
}

func main() {
}

//export cs_hello
func cs_hello() {

}

//stateH is current proxy state
func stateH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := map[string]interface{}{}
	if netProxy == nil {
		res["status"] = "not started"
	} else {
		res["status"] = "ok"
		res["netstack"] = netProxy.State()
		if s, ok := netDialer.(*core.MessageDialer); ok {
			res["message"] = s.State()
		}
	}
	res["cpu"] = runtime.NumCPU()
	res["cgo"] = runtime.NumCgoCall()
	res["go"] = runtime.NumGoroutine()
	data, _ := json.Marshal(res)
	w.Write(data)
}

//testWebH is current proxy state
func testWebH(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	testWeb(url, "", w)
	fmt.Fprintf(w, "all done\n")
}

var netRWC io.ReadWriteCloser
var netProxy *netstack.NetProxy
var netDialer core.RawDialer
var netRunning bool
var netLocker = sync.RWMutex{}

//bootstrap will bootstrap by config file path and mtu and net stream
func bootstrap(conf string, mtu int, dump string, rwc io.ReadWriteCloser, dialer core.RawDialer) (res string) {
	targetRWC := rwc
	if len(dump) > 0 {
		var err error
		targetRWC, err = pcap.NewDumper(rwc, dump)
		if err != nil {
			res = err.Error()
			return
		}
	}
	netDialer = dialer
	netRWC = targetRWC
	netProxy = netstack.NewNetProxy(dialer)
	netProxy.Through = true
	netProxy.MTU, netProxy.Timeout = mtu, 15*time.Second
	err := netProxy.Bootstrap(conf)
	if err != nil {
		res = err.Error()
		if closer, ok := netDialer.(io.Closer); ok {
			closer.Close()
		}
		netDialer = nil
		netRWC.Close()
		return
	}
	if len(dump) < 1 && netProxy.ClientConf.LogLevel > core.LogLevelDebug {
		targetRWC = tcpip.NewPacketPrintRWC(true, "Net", rwc, false)
	}
	netProxy.SetWriter(targetRWC)
	core.InfoLog("NetProxy is bootstrap done")
	return
}

//cs_start the process
//export cs_start
func cs_start() (res string) {
	if netRunning {
		res = "already started"
		return
	}
	netRunning = true
	core.InfoLog("NetProxy is starting")
	// netProxy.StartProcessReader(netRWC)
	netProxy.StartProcessTimeout()
	return
}

//stop the process
//export cs_stop
func cs_stop() (res string) {
	netLocker.Lock()
	defer netLocker.Unlock()
	if !netRunning || netProxy == nil {
		res = "not started"
		return
	}
	if netRWC != nil {
		netRWC.Close()
	}
	if closer, ok := netDialer.(io.Closer); ok {
		closer.Close()
	}
	netProxy.Close()
	netProxy.Wait()
	core.InfoLog("NetProxy is stopped")
	netProxy = nil
	netDialer = nil
	netRWC = nil
	netRunning = false
	return
}

//cs_proxy_set will add proxy setting by key
//export cs_proxy_set
func cs_proxy_set(key string, proxy bool) (res string) {
	netLocker.RLock()
	defer netLocker.RUnlock()
	if netProxy == nil {
		res = "not started"
		return
	}
	if proxy {
		netProxy.NetProcessor.GFW.Set(key, dns.GfwProxy)
	} else {
		netProxy.NetProcessor.GFW.Set(key, dns.GfwLocal)
	}
	return
}

//cs_change_mode will change proxy mode by global/auto
//export cs_change_mode
func cs_change_mode(mode string) (res string) {
	netLocker.RLock()
	defer netLocker.RUnlock()
	if netProxy == nil {
		res = "not started"
		return
	}
	switch mode {
	case "global":
		netProxy.NetProcessor.GFW.Set("*", dns.GfwProxy)
	case "auto":
		netProxy.NetProcessor.GFW.Set("*", dns.GfwLocal)
	}
	return
}

//cs_proxy_mode will return current proxy mode
//export cs_proxy_mode
func cs_proxy_mode() (mode string) {
	netLocker.RLock()
	defer netLocker.RUnlock()
	if netProxy == nil {
		mode = "auto"
	} else if netProxy.NetProcessor.Get("*") == dns.GfwProxy {
		mode = "global"
	} else {
		mode = "auto"
	}
	return
}

//cs_test_web will test web request
//export cs_test_web
func cs_test_web(url, digest string) {
	testWeb(url, digest, os.Stdout)
}

func testWeb(url, digest string, outer io.Writer) {
	fmt.Fprintf(outer, "testWeb start by url:%v,digest:%v\n", url, digest)
	// var raw net.Conn
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (conn net.Conn, err error) {
				conn, err = net.Dial(network, addr)
				return
			},
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(outer, "testWeb new fail with %v\n", err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(outer, "testWeb do fail with %v\n", err)
		return
	}
	defer res.Body.Close()
	buf := bytes.NewBuffer(nil)
	// h := sha1.New()
	_, err = io.Copy(buf, res.Body)
	// if err != nil {
	// 	fmt.Fprintf(outer, "testWeb copy fail with %v\n", err)
	// 	return
	// }
	// v := fmt.Sprintf("%x", h.Sum(nil))
	// if v != digest {
	// 	fmt.Fprintf(outer, "testWeb done fail by digest not match %v,%v\n", v, digest)
	// } else {
	// 	fmt.Fprintf(outer, "testWeb done success by %v\n", v)
	// }
	// if raw != nil {
	// 	raw.Close()
	// }
	fmt.Fprintf(outer, "testWeb response\n%v\n", string(buf.Bytes()))
}
