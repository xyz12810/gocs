package dns

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/coversocks/gocs/core"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	//GfwProxy is GFW target for proxy
	GfwProxy = "dns://proxy"
	//GfwLocal is GFW target for local
	GfwLocal = "dns://local"
)

//GFW impl check if domain in gfw list
type GFW struct {
	list map[string]string
	lck  sync.RWMutex
}

//NewGFW will create new GFWList
func NewGFW() (gfw *GFW) {
	gfw = &GFW{
		list: map[string]string{
			"*": GfwLocal,
		},
		lck: sync.RWMutex{},
	}
	return
}

//Set list
func (g *GFW) Set(list, target string) {
	g.lck.Lock()
	defer g.lck.Unlock()
	g.list[list] = target
}

//Get list
func (g *GFW) Get(list string) (target string) {
	g.lck.Lock()
	defer g.lck.Unlock()
	target = g.list[list]
	return
}

//IsProxy return true, if domain target is dns://proxy
func (g *GFW) IsProxy(domain string) bool {
	return g.Find(domain) == GfwProxy
}

//Find domain target
func (g *GFW) Find(domain string) (target string) {
	g.lck.RLock()
	defer g.lck.RUnlock()
	domain = strings.Trim(domain, " \t.")
	if len(domain) < 1 {
		target = g.list["*"]
		return
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		target = g.check(parts...)
	} else {
		n := len(parts) - 1
		for i := 0; i < n; i++ {
			target = g.check(parts[i:]...)
			if len(target) > 0 {
				break
			}
		}
	}
	if len(target) < 1 {
		target = g.list["*"]
	}
	return
}

func (g *GFW) check(parts ...string) (target string) {
	ptxt := fmt.Sprintf("(?m)^[^\\@]*[\\|\\.]*(http://)?(https://)?%v$", strings.Join(parts, "\\."))
	pattern, err := regexp.Compile(ptxt)
	if err == nil {
		for key, val := range g.list {
			if len(pattern.FindString(key)) > 0 {
				target = val
				break
			}
		}
	}
	return
}

func (g *GFW) String() string {
	return "GFW"
}

//Conn impl the  connection for read/write  message
type Conn struct {
	p            *Processor
	key          string
	base         io.ReadWriteCloser
	receiverChan chan []byte
	receiverFunc core.OnReceivedF
	closerFunc   core.OnClosedF
	closed       bool
	reader       io.Reader
	lck          sync.RWMutex
}

//NewConn will create new Conn
func NewConn(p *Processor, key string, base io.ReadWriteCloser, bufferSize int) (conn *Conn) {
	conn = &Conn{
		p:            p,
		key:          key,
		base:         base,
		receiverChan: make(chan []byte, 1024),
		lck:          sync.RWMutex{},
	}
	if bufferSize > 0 {
		conn.reader = bufio.NewReaderSize(core.ReaderF(conn.rawRead), bufferSize)
	} else {
		conn.reader = core.ReaderF(conn.rawRead)
	}
	return
}

//Throughable is for impl core.ThroughReadeCloser
func (c *Conn) Throughable() bool {
	return true
}

//OnReceived is for impl core.ThroughReadeCloser
func (c *Conn) OnReceived(f core.OnReceivedF) (err error) {
	c.receiverFunc = f
	return
}

//OnClosed is core.ThroughReadeCloser impl
func (c *Conn) OnClosed(f core.OnClosedF) (err error) {
	c.closerFunc = f
	return
}

func (c *Conn) receiveData(data []byte) {
	if c.receiverFunc == nil {
		buf := make([]byte, len(data))
		copy(buf, data)
		c.receiverChan <- buf
	} else {
		c.receiverFunc(c, data)
	}
}

func (c *Conn) Read(p []byte) (n int, err error) {
	n, err = c.reader.Read(p)
	return
}

func (c *Conn) rawRead(p []byte) (n int, err error) {
	c.lck.RLock()
	if c.closed {
		c.lck.RUnlock()
		return
	}
	c.lck.RUnlock()
	data := <-c.receiverChan
	if data == nil {
		err = fmt.Errorf("closed")
		return
	}
	if len(data) > len(p) {
		err = fmt.Errorf("Conn.Read buffer is too small for %v, expect %v", len(p), len(data))
		return
	}
	n = copy(p, data)
	// fmt.Printf("Conn(%p).Read---->%v,%v\n", d, n, data)
	return
}

func (c *Conn) Write(p []byte) (n int, err error) {
	c.lck.RLock()
	if c.closed {
		c.lck.RUnlock()
		return
	}
	c.lck.RUnlock()
	n, err = c.base.Write(p)
	return
}

//Close will close the connection
func (c *Conn) Close() (err error) {
	c.lck.Lock()
	if c.closed {
		err = fmt.Errorf("closed")
		c.lck.Unlock()
		return
	}
	c.closed = true
	close(c.receiverChan)
	c.receiverFunc = nil
	c.lck.Unlock()
	if c.closerFunc != nil {
		c.closerFunc(c)
	}
	c.p.close(c)
	return
}

func (c *Conn) String() string {
	return fmt.Sprintf("Conn(%v)", c.base)
}

//Processor impl to core.Processor for process  connection
type Processor struct {
	Async      bool
	Through    bool
	bufferSize int
	conns      map[string]*Conn
	connsLck   sync.RWMutex
	Next       core.Processor
	Target     func(domain string) string
}

//NewProcessor will create new Processor
func NewProcessor(async, through bool, bufferSize int, next core.Processor, target func(domain string) string) (p *Processor) {
	p = &Processor{
		Async:      async,
		Through:    through,
		bufferSize: bufferSize,
		conns:      map[string]*Conn{},
		connsLck:   sync.RWMutex{},
		Next:       next,
		Target:     target,
	}
	return
}

//ProcConn will process connection
func (p *Processor) ProcConn(r io.ReadWriteCloser, target string) (err error) {
	if through, ok := r.(core.ThroughReadeCloser); p.Through && ok && through.Throughable() {
		through.OnReceived(p.procData)
		through.OnClosed(p.procClose)
		return
	}
	if p.Async {
		go p.proc(r, target)
	} else {
		err = p.proc(r, target)
	}
	return
}

func (p *Processor) proc(r io.ReadWriteCloser, target string) (err error) {
	core.DebugLog("Processor dns runner is starting for %v", r)
	var n int
	buf := make([]byte, 32*1024)
	for {
		n, err = r.Read(buf)
		if err != nil {
			core.InfoLog("Processor(DNS) connection %v read fail with %v", r, err)
			break
		}
		err = p.procData(r, buf[0:n])
	}
	p.procClose(r)
	core.DebugLog("Processor dns runner is stopped for %v", r)
	return
}

func (p *Processor) close(c *Conn) {
	p.connsLck.Lock()
	delete(p.conns, c.key)
	p.connsLck.Unlock()
}

func (p *Processor) procData(r io.ReadWriteCloser, data []byte) (err error) {
	msg := new(dnsmessage.Message)
	err = msg.Unpack(data)
	if err != nil {
		core.WarnLog("Processor(DNS) unpack dns package fail with %v by %v", err, data)
		return
	}
	var target = p.Target(msg.Questions[0].Name.GoString())
	var key = fmt.Sprintf("%p-%v", r, target)
	p.connsLck.Lock()
	conn, ok := p.conns[key]
	if !ok {
		conn = NewConn(p, key, r, p.bufferSize)
		p.conns[key] = conn
	}
	p.connsLck.Unlock()
	if !ok {
		err = p.Next.ProcConn(conn, target)
		if err != nil {
			//drop it
			core.WarnLog("Processor dns runner proc %v fail with %v", r, err)
			return
		}
	}
	conn.receiveData(data)
	return
}

func (p *Processor) procClose(r io.ReadWriteCloser) (err error) {
	r.Close()
	prefix := fmt.Sprintf("%p-", r)
	closing := []*Conn{}
	p.connsLck.Lock()
	allc := len(p.conns)
	for key, c := range p.conns {
		if strings.HasPrefix(key, prefix) {
			closing = append(closing, c)
			delete(p.conns, key)
		}
	}
	p.connsLck.Unlock()
	core.DebugLog("Processor dns connection is closed for %v and close %v/%v connection", r, len(closing), allc)
	for _, c := range closing {
		c.Close()
	}
	return
}

//State will return the state
func (p *Processor) State() (state interface{}) {
	conns := map[string]interface{}{}
	p.connsLck.Lock()
	for key, c := range p.conns {
		conns[key] = map[string]interface{}{
			"base": fmt.Sprintf("%v", c.base),
		}
	}
	p.connsLck.Unlock()
	state = conns
	return
}

func (p *Processor) String() string {
	return "Processor(DNS)"
}

//RecordConn is  connection for recording  response
type RecordConn struct {
	io.ReadWriteCloser
	p *RecordProcessor
}

//NewRecordConn will create new RecordConn
func NewRecordConn(p *RecordProcessor, base io.ReadWriteCloser) (conn *RecordConn) {
	conn = &RecordConn{
		p:               p,
		ReadWriteCloser: base,
	}
	return
}

func (r *RecordConn) Read(p []byte) (n int, err error) {
	n, err = r.ReadWriteCloser.Read(p)
	return
}

func (r *RecordConn) Write(p []byte) (n int, err error) {
	msg := new(dnsmessage.Message)
	xerr := msg.Unpack(p)
	if xerr != nil {
		core.WarnLog("Record unpack dns fail with %v by %v", xerr, p)
	}
	if xerr == nil && len(msg.Answers) > 0 {
		for _, a := range msg.Answers {
			if a.Header.Type == dnsmessage.TypeA {
				key := a.Body.(*dnsmessage.AResource).GoString()
				val := msg.Questions[0].Name.GoString()
				core.DebugLog("Record recoding %v->%v", key, val)
				r.p.Record(key, val)
			}
		}
	}
	n, err = r.ReadWriteCloser.Write(p)
	return
}

//Close will close base connection
func (r *RecordConn) Close() (err error) {
	err = r.ReadWriteCloser.Close()
	core.DebugLog("%v is closed", r)
	return
}

func (r *RecordConn) String() string {
	return fmt.Sprintf("RecordConn(%v)", r.ReadWriteCloser)
}

//RecordProcessor to impl processor for record  response
type RecordProcessor struct {
	Next   core.Processor
	allIP  map[string]string
	allLck sync.RWMutex
}

//NewRecordProcessor will create new RecordProcessor
func NewRecordProcessor(next core.Processor) (r *RecordProcessor) {
	r = &RecordProcessor{
		Next:   next,
		allIP:  map[string]string{},
		allLck: sync.RWMutex{},
	}
	return
}

//Record the key and value
func (r *RecordProcessor) Record(key, val string) {
	r.allLck.Lock()
	r.allIP[key] = val
	r.allLck.Unlock()
}

//IsRecorded will check if key is recorded
func (r *RecordProcessor) IsRecorded(key string) (ok bool) {
	r.allLck.RLock()
	_, ok = r.allIP[key]
	r.allLck.RUnlock()
	return
}

//Value will return record value by key
func (r *RecordProcessor) Value(key string) (val string) {
	r.allLck.RLock()
	val, _ = r.allIP[key]
	r.allLck.RUnlock()
	return
}

//Clear all recorded
func (r *RecordProcessor) Clear() {
	r.allLck.Lock()
	r.allIP = map[string]string{}
	r.allLck.Unlock()
}

//ProcConn will process connection
func (r *RecordProcessor) ProcConn(base io.ReadWriteCloser, target string) (err error) {
	core.DebugLog("RecordProcessor proc %v", base)
	err = r.Next.ProcConn(NewRecordConn(r, base), target)
	return
}

//State will return the state
func (r *RecordProcessor) State() (state interface{}) {
	all := map[string]interface{}{}
	r.allLck.RLock()
	for k, v := range r.allIP {
		all[k] = v
	}
	r.allLck.RUnlock()
	state = all
	return
}

func (r *RecordProcessor) String() string {
	return "RecordProcessor"
}
