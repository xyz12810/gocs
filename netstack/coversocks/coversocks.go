package coversocks

import (
	"encoding/json"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/coversocks/gocs/netstack/dns"
	"github.com/coversocks/gocs/netstack/pcap"
	"github.com/coversocks/gocs/netstack/tcpip"

	"net/http"
	_ "net/http/pprof"

	"github.com/coversocks/gocs/core"
	"github.com/coversocks/gocs/netstack"
)

var xx = false

func init() {
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(1)
	// go func() {
	// 	http.HandleFunc("/debug/state", stateH)
	// 	log.Println(http.ListenAndServe(":6060", nil))
	// }()
}

//Hello will start inner debug and do nothing
func Hello() {
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

var netRWC io.ReadWriteCloser
var netProxy *netstack.NetProxy
var netDialer core.RawDialer
var netRunning bool
var netLocker = sync.RWMutex{}

//Bootstrap will bootstrap by config file path and mtu and net stream
func Bootstrap(conf string, mtu int, dump string, rwc io.ReadWriteCloser, dialer core.RawDialer) (res string) {
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

//Start the process
func Start() (res string) {
	if netRunning {
		res = "already started"
		return
	}
	netRunning = true
	core.InfoLog("NetProxy is starting")
	netProxy.StartProcessReader(netRWC)
	netProxy.StartProcessTimeout()
	return
}

//Stop the process
func Stop() (res string) {
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

//ProxySet will add proxy setting by key
func ProxySet(key string, proxy bool) (res string) {
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

//ChangeMode will change proxy mode by global/auto
func ChangeMode(mode string) (res string) {
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

//ProxyMode will return current proxy mode
func ProxyMode() (mode string) {
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

//TestWeb will test web request
func TestWeb(url, digest string) {
	// fmt.Printf("testWeb start by url:%v,digest:%v\n", url, digest)
	// var raw net.Conn
	// client := &http.Client{
	// 	Transport: &http.Transport{
	// 		Dial: func(network, addr string) (conn net.Conn, err error) {
	// 			conn, err = net.Dial(network, addr)
	// 			return
	// 		},
	// 	},
	// }
	// req, err := http.NewRequest("GET", url, nil)
	// if err != nil {
	// 	fmt.Printf("testWeb new fail with %v\n", err)
	// 	return
	// }
	// res, err := client.Do(req)
	// if err != nil {
	// 	fmt.Printf("testWeb do fail with %v\n", err)
	// 	return
	// }
	// defer res.Body.Close()
	// h := sha1.New()
	// _, err = io.Copy(h, res.Body)
	// if err != nil {
	// 	fmt.Printf("testWeb copy fail with %v\n", err)
	// 	return
	// }
	// v := fmt.Sprintf("%x", h.Sum(nil))
	// if v != digest {
	// 	fmt.Printf("testWeb done fail by digest not match %v,%v\n", v, digest)
	// } else {
	// 	fmt.Printf("testWeb done success by %v\n", v)
	// }
	// if raw != nil {
	// 	raw.Close()
	// }
}
