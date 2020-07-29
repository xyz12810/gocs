package coversocks

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/coversocks/gocs/core"
)

var netBinderRWC *binderRWC
var netBinderDialer *binderDialer
var netBinderLocker = sync.RWMutex{}

//BootstrapBinder will bootstrap by config file path
func BootstrapBinder(conf string, mtu int, retain bool, dump string) (res string) {
	core.InfoLog("Binder bootstrap by conf:%v,mtu:%v,dump:%v", conf, mtu, dump)
	netBinderRWC = newBinderRWC(retain, 8)
	netBinderDialer = newBinderDialer(8 * 1024)
	res = Bootstrap(conf, mtu, dump, netBinderRWC, netBinderDialer)
	return
}

//BinderInboundWrite write inbound data to the netstack
func BinderInboundWrite(buffer []byte, offset, length int) (n int) {
	netBinderLocker.RLock()
	binder := netBinderRWC
	if binder == nil {
		netBinderLocker.RUnlock()
		n = -1
		return
	}
	netBinderLocker.RUnlock()
	err := binder.Push(buffer[offset : offset+length])
	if err != nil {
		n = -1
		core.WarnLog("BinderInboundWrite push data fail with %v", err)
		return
	}
	n = length
	return
}

//BinderConnWrite write inbound data to the netstack
func BinderConnWrite(native int64, buffer []byte, offset, length int) (n int) {
	netBinderLocker.RLock()
	dialer := netBinderDialer
	if dialer == nil {
		netBinderLocker.RUnlock()
		n = -1
		return
	}
	netBinderLocker.RUnlock()
	err := dialer.Push(native, buffer[offset:offset+length])
	if err != nil {
		n = -1
		core.WarnLog("BinderConnWrite push data fail with %v", err)
		return
	}
	n = length
	net.Dial
	return
}

//BinderConnClose write inbound data to the netstack
func BinderConnClose(native int64) {
	netBinderLocker.RLock()
	dialer := netBinderDialer
	if dialer == nil {
		netBinderLocker.RUnlock()
		return
	}
	netBinderLocker.RUnlock()
	dialer.CloseConn(native)
}

//binderRWC is binder net ReadWriteCloser
type binderRWC struct {
	*core.ChannelRWC
	closed bool
	locker sync.RWMutex
}

//newBinderRWC will create new binderRWC
func newBinderRWC(retain bool, bufferSize int) (binder *binderRWC) {
	binder = &binderRWC{
		locker:     sync.RWMutex{},
		ChannelRWC: core.NewChannelRWC(false, retain, bufferSize),
	}
	return
}

func (b *binderRWC) Write(p []byte) (n int, err error) {
	b.locker.RLock()
	defer b.locker.RUnlock()
	if b.closed {
		err = fmt.Errorf("closed")
		return
	}
	cgoPackWrite(p, len(p))
	return
}

//Close will mark rwc is closed
func (b *binderRWC) Close() (err error) {
	b.locker.Lock()
	defer b.locker.Unlock()
	if b.closed {
		err = fmt.Errorf("closed")
		return
	}
	b.closed = true
	return
}

//binderConn is net.Conn impl by binder function
type binderConn struct {
	// *bufio.Reader
	io.Reader
	remote      string
	native      int64
	received    chan []byte
	err         error
	closeLocker sync.RWMutex
	dialer      *binderDialer
}

//newBinderConn will create new binderConn
func newBinderConn(dialer *binderDialer, native int64, remote string, bufferSize int) (conn *binderConn) {
	conn = &binderConn{
		dialer:      dialer,
		remote:      remote,
		native:      native,
		received:    make(chan []byte, 1024),
		closeLocker: sync.RWMutex{},
	}
	// conn.Reader = bufio.NewReaderSize(core.ReaderF(conn.rawRead), bufferSize)
	return
}

func (b *binderConn) Write(p []byte) (n int, err error) {
	var res = cgoConnWrite(b.native, p, len(p))
	if res > 0 {
		n = int(res)
	} else {
		err = fmt.Errorf("code:%v", res)
	}
	return
}

func (b *binderConn) rawRead(p []byte) (n int, err error) {
	b.closeLocker.RLock()
	if err != nil {
		b.closeLocker.RUnlock()
		err = b.err
		return
	}
	b.closeLocker.RUnlock()
	data := <-b.received
	if len(data) < 1 {
		err = b.err
		return
	}
	if len(p) < len(data) {
		err = fmt.Errorf("buffer to small expect %v, but %v", len(data), len(p))
		return
	}
	n = copy(p, data)
	return
}

//Close will close the channel
func (b *binderConn) Close() (err error) {
	err = b.closeByError(fmt.Errorf("closed by local"), true)
	b.dialer.removeConn(b.native)
	return
}

func (b *binderConn) closeByError(e error, native bool) (err error) {
	b.closeLocker.RLock()
	if b.err != nil {
		b.closeLocker.RUnlock()
		err = b.err
		return
	}
	b.err = e
	close(b.received)
	b.closeLocker.RUnlock()
	if native {
		cgoConnClose(b.native)
	}
	return
}

//LocalAddr return then local network address
func (b *binderConn) LocalAddr() net.Addr {
	return b
}

// RemoteAddr returns the remote network address.
func (b *binderConn) RemoteAddr() net.Addr {
	return b
}

// SetDeadline impl net.Conn do nothing
func (b *binderConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline impl net.Conn do nothing
func (b *binderConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline impl net.Conn do nothing
func (b *binderConn) SetWriteDeadline(t time.Time) error {
	return nil
}

//Network impl net.Addr
func (b *binderConn) Network() string {
	return "binder"
}

func (b *binderConn) String() string {
	return fmt.Sprintf("binder <-> %v", b.remote)
}

//binderDialer is core.RawDialer impl by binder func
type binderDialer struct {
	conns      map[int64]*binderConn
	locker     sync.RWMutex
	bufferSize int
}

//newBinderDialer will create new binderDialer
func newBinderDialer(bufferSize int) (dialer *binderDialer) {
	dialer = &binderDialer{
		conns:      map[int64]*binderConn{},
		locker:     sync.RWMutex{},
		bufferSize: bufferSize,
	}
	return
}

//Dial will dial one binder connection
func (b *binderDialer) Dial(network, address string) (conn net.Conn, err error) {
	remote := []byte(fmt.Sprintf("%v://%v", network, address))
	native := cgoConnDial(remote, len(remote))
	if native < 1 {
		err = fmt.Errorf("binder dial to %v fail", string(remote))
		return
	}
	bconn := newBinderConn(b, native, string(remote), b.bufferSize)
	b.locker.Lock()
	b.conns[native] = bconn
	b.locker.Unlock()
	conn = bconn
	return
}

//Push will push one data to connection.
func (b *binderDialer) Push(native int64, p []byte) (err error) {
	b.locker.RLock()
	conn, ok := b.conns[native]
	b.locker.RUnlock()
	if !ok {
		err = fmt.Errorf("connection not exist by %p", native)
		return
	}
	buf := make([]byte, len(p))
	copy(buf, p)
	conn.closeLocker.RLock()
	defer conn.closeLocker.RUnlock()
	if conn.err != nil {
		err = conn.err
		return
	}
	conn.received <- buf
	return
}

func (b *binderDialer) CloseConn(native int64) (err error) {
	b.locker.RLock()
	conn, ok := b.conns[native]
	b.locker.RUnlock()
	if ok {
		b.removeConn(native)
		err = conn.closeByError(fmt.Errorf("close by remote"), false)
	} else {
		err = fmt.Errorf("connection not exist by %p", native)
	}
	return
}

func (b *binderDialer) removeConn(key int64) {
	b.locker.Lock()
	delete(b.conns, key)
	b.locker.Unlock()
}
