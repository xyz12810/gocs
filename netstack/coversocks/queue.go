package coversocks

import (
	"fmt"

	"github.com/coversocks/gocs/core"
)

var netQueueRWC *core.ChannelRWC

//BootstrapQueue will bootstrap by config file path, n<0 return is fail, n==0 is success
func BootstrapQueue(conf string, mtu int, async, retain bool, dump string) (res string) {
	core.InfoLog("Queue bootstrap by conf:%v,mtu:%v,dump:%v", conf, mtu, dump)
	netQueueRWC = core.NewChannelRWC(async, retain, 1024)
	netMessageDialer = core.NewMessageDialer([]byte("m"), 512*1024)
	res = Bootstrap(conf, mtu, dump, netQueueRWC, netMessageDialer)
	return
}

//QueueOutboundRead read outbound data from the netstack
func QueueOutboundRead(buffer []byte, offset, length int) (n int) {
	b := buffer[offset : offset+length]
	data, err := netQueueRWC.Pull()
	if err != nil {
		n = -1
		core.WarnLog("ReadQuque pull data fail with %v", err)
		return
	}
	if len(data) > length {
		err := fmt.Errorf("ReadQuque buffer is too small expected %v, but %v", len(data), length)
		core.WarnLog("ReadQuque copy data fail with %v", err)
		n = -2
		return
	}
	if len(data) > 0 {
		n = copy(b, data)
	}
	return
}

//QueueInboundWrite write inbound data to the netstack
func QueueInboundWrite(buffer []byte, offset, length int) (n int) {
	err := netQueueRWC.Push(buffer[offset : offset+length])
	if err != nil {
		n = -1
		core.WarnLog("WriteQuque push data fail with %v", err)
		return
	}
	n = length
	return
}
