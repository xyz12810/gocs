package coversocks

import (
	"io"
	"os"

	"github.com/coversocks/gocs/core"
)

var netFD int
var netFDRWC io.ReadWriteCloser

//BootstrapFD will bootstrap by config file path, n<0 return is fail, n==0 is success
func BootstrapFD(conf string, mtu, fd int, dump string) (res string) {
	core.InfoLog("FD bootstrap by conf:%v,mtu:%v,fd:%v,dump:%v", conf, mtu, fd, dump)
	netFD = fd
	netFDRWC = os.NewFile(uintptr(fd), "FD")
	netMessageDialer = core.NewMessageDialer([]byte("m"), 512*1024)
	res = Bootstrap(conf, mtu, dump, netFDRWC, netMessageDialer)
	return
}
