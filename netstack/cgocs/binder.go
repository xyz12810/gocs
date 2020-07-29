package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"unsafe"

	"github.com/coversocks/gocs/core"
)

import "C"

var netBinderRWC *binderRWC
var netBinderDialer *binderDialer
var netBinderLocker = sync.RWMutex{}

//cgo_cs_bootstrap will bootstrap by config file path
//export cs_bootstrap
func cs_bootstrap(conf string, mtu int, retain bool, dump string) (res string) {
	core.InfoLog("Binder bootstrap by conf:%v,mtu:%v,dump:%v", conf, mtu, dump)
	netBinderRWC = newBinderRWC()
	netBinderDialer = newBinderDialer(8 * 1024)
	res = bootstrap(conf, mtu, dump, netBinderRWC, netBinderDialer)
	return
}

//cs_inbound_write write inbound data to the netstack
//export cs_inbound_write
func cs_inbound_write(buffer []byte, offset, length int) (n int) {
	netBinderLocker.RLock()
	binder := netBinderRWC
	if binder == nil {
		netBinderLocker.RUnlock()
		n = -1
		return
	}
	netBinderLocker.RUnlock()
	core.InfoLog("cs_inbound_write write %d bytes\n", length)
	err := netProxy.ProcessBuffer(buffer, offset, length)
	if err != nil {
		n = -1
		core.WarnLog("cs_inbound_write process buffer data fail with %v", err)
		return
	}
	n = length
	return
}

//cs_dial_done write conn data to the netstack
//export cs_dial_done
func cs_dial_done(conn unsafe.Pointer, code int) (n int) {
	netBinderLocker.RLock()
	dialer := netBinderDialer
	if dialer == nil {
		netBinderLocker.RUnlock()
		n = -1
		return
	}
	netBinderLocker.RUnlock()
	core.InfoLog("cs_dial_done wit code %v to %p\n", code, conn)
	err := dialer.DialDone(conn, code)
	if err != nil {
		n = -1
		core.WarnLog("cs_dial_done dial done fail with %v", err)
		return
	}
	return
}

//cs_conn_write write conn data to the netstack
//export cs_conn_write
func cs_conn_write(conn unsafe.Pointer, buffer []byte, offset, length int) (n int) {
	netBinderLocker.RLock()
	dialer := netBinderDialer
	if dialer == nil {
		netBinderLocker.RUnlock()
		n = -1
		return
	}
	netBinderLocker.RUnlock()
	core.InfoLog("cs_conn_write write %d bytes to %p\n", length, conn)
	err := dialer.ReceiveData(conn, buffer[offset:offset+length])
	if err != nil {
		n = -1
		core.WarnLog("cs_conn_write receive data fail with %v", err)
		return
	}
	n = length
	return
}

//cs_conn_close close the connection
//export cs_conn_close
func cs_conn_close(conn unsafe.Pointer) {
	netBinderLocker.RLock()
	dialer := netBinderDialer
	if dialer == nil {
		netBinderLocker.RUnlock()
		return
	}
	netBinderLocker.RUnlock()
	core.InfoLog("cs_conn_close close conn %p\n", conn)
	dialer.CloseConn(conn)
}

//binderRWC is binder net ReadWriteCloser
type binderRWC struct {
	closed bool
	locker sync.RWMutex
}

//newBinderRWC will create new binderRWC
func newBinderRWC() (binder *binderRWC) {
	binder = &binderRWC{
		locker: sync.RWMutex{},
	}
	return
}

func (b *binderRWC) Read(p []byte) (n int, err error) {
	panic("not imppl")
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
	reader       io.Reader
	remote       string
	native       unsafe.Pointer
	receiverChan chan []byte
	receiverFunc core.OnReceivedF
	closerFunc   core.OnClosedF
	err          error
	closeLocker  sync.RWMutex
	dialer       *binderDialer
	connected    bool
	connLocker   sync.RWMutex
}

//newBinderConn will create new binderConn
func newBinderConn(dialer *binderDialer, native unsafe.Pointer, remote string, bufferSize int) (conn *binderConn) {
	conn = &binderConn{
		dialer:       dialer,
		remote:       remote,
		native:       native,
		receiverChan: make(chan []byte, 1024),
		closeLocker:  sync.RWMutex{},
		connLocker:   sync.RWMutex{},
	}
	if bufferSize > 0 {
		conn.reader = bufio.NewReaderSize(core.ReaderF(conn.rawRead), bufferSize)
	} else {
		conn.reader = core.ReaderF(conn.rawRead)
	}
	return
}

func (b *binderConn) Write(p []byte) (n int, err error) {
	b.connLocker.RLock() //only for wait connected
	b.connLocker.RUnlock()
	var res = cgoConnWrite(b.native, p, len(p))
	if res > 0 {
		n = int(res)
	} else {
		err = fmt.Errorf("code:%v", res)
	}
	return
}

func (b *binderConn) Read(p []byte) (n int, err error) {
	n, err = b.reader.Read(p)
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
	data := <-b.receiverChan
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
	b.dialer.removeConn(b.native)
	err = b.closeByError(fmt.Errorf("closed by local"))
	return
}

func (b *binderConn) closeByError(e error) (err error) {
	b.closeLocker.RLock()
	if b.err != nil {
		b.closeLocker.RUnlock()
		err = b.err
		return
	}
	b.err = e
	close(b.receiverChan)
	b.closeLocker.RUnlock()
	cgoConnClose(b.native)
	b.native = nil
	return
}

//Throughable is core.ThroughReadeCloser impl
func (b *binderConn) Throughable() bool {
	return true
}

//OnReceived is core.ThroughReadeCloser impl
func (b *binderConn) OnReceived(f core.OnReceivedF) (err error) {
	b.receiverFunc = f
	return
}

//OnClosed is core.ThroughReadeCloser impl
func (b *binderConn) OnClosed(f core.OnClosedF) (err error) {
	b.closerFunc = f
	return
}

func (b *binderConn) receiveData(data []byte) {
	if b.receiverFunc == nil {
		buf := make([]byte, len(data))
		copy(buf, data)
		b.receiverChan <- buf
	} else {
		b.receiverFunc(b, data)
	}
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
	conns      map[unsafe.Pointer]*binderConn
	locker     sync.RWMutex
	bufferSize int
}

//newBinderDialer will create new binderDialer
func newBinderDialer(bufferSize int) (dialer *binderDialer) {
	dialer = &binderDialer{
		conns:      map[unsafe.Pointer]*binderConn{},
		locker:     sync.RWMutex{},
		bufferSize: bufferSize,
	}
	return
}

//Dial will dial one binder connection
func (b *binderDialer) Dial(network, address string) (conn net.Conn, err error) {
	remote := []byte(fmt.Sprintf("%v://%v", network, address))
	native := cgoConnDial(remote, len(remote))
	if native == nil {
		err = fmt.Errorf("binder dial to %v fail", string(remote))
		return
	}
	bconn := newBinderConn(b, native, string(remote), b.bufferSize)
	b.locker.Lock()
	b.conns[native] = bconn
	b.locker.Unlock()
	bconn.connLocker.Lock() //wait connected
	conn = bconn
	return
}

//ReceiveData will receive data to connection.
func (b *binderDialer) DialDone(native unsafe.Pointer, code int) (err error) {
	b.locker.RLock()
	conn, ok := b.conns[native]
	b.locker.RUnlock()
	if !ok {
		err = fmt.Errorf("connection not exist by %p", native)
		return
	}
	if conn.connected || conn.err != nil {
		return
	}
	if code != 0 {
		b.removeConn(conn.native)
		conn.closeByError(fmt.Errorf("dial fail with code %v", code))
	}
	conn.connLocker.Unlock()
	return
}

//ReceiveData will receive data to connection.
func (b *binderDialer) ReceiveData(native unsafe.Pointer, p []byte) (err error) {
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
	conn.receiveData(p)
	return
}

func (b *binderDialer) CloseConn(native unsafe.Pointer) (err error) {
	b.locker.RLock()
	conn, ok := b.conns[native]
	b.locker.RUnlock()
	if ok {
		b.removeConn(native)
		err = conn.closeByError(fmt.Errorf("close by remote"))
	} else {
		err = fmt.Errorf("connection not exist by %p", native)
	}
	return
}

func (b *binderDialer) removeConn(key unsafe.Pointer) {
	b.locker.Lock()
	delete(b.conns, key)
	b.locker.Unlock()
}

type binderLogger struct {
}

func newBinderLogger() (logger *binderLogger) {
	logger = &binderLogger{}
	return
}

func (b *binderLogger) Write(p []byte) (n int, err error) {
	cgoLogWrite(p, len(p))
	n = len(p)
	return
}
