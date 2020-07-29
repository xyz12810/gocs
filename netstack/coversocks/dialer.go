package coversocks

import (
	"io"

	"github.com/coversocks/gocs/core"
)

var netMessageDialer *core.MessageDialer

//ReadMessage queue one message
func ReadMessage(b []byte) (n int) {
	netLocker.RLock()
	if netMessageDialer == nil {
		n = -1
		netLocker.RUnlock()
		return
	}
	msg := netMessageDialer
	netLocker.RUnlock()
	n, err := msg.Read(b)
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
	netLocker.RLock()
	if netMessageDialer == nil {
		n = -1
		netLocker.RUnlock()
		return
	}
	msg := netMessageDialer
	netLocker.RUnlock()
	n, err := msg.Write(b)
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
