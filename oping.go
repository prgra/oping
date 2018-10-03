package oping

import (
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Stat one icmp statistic
type Stat struct {
	SendTime time.Time
	RecvTime time.Time
	Recv     bool
	Size     int
}

type ping struct {
	IP  string
	Seq int
}

type pingParam struct {
	ip    string
	cnt   int
	rchan chan []Stat
}

// Pinger main structure
type Pinger struct {
	dbmut    sync.RWMutex
	db       map[ping]chan Stat
	c        *icmp.PacketConn
	workerwg sync.WaitGroup
	ch       chan pingParam
	timeout  time.Duration
	interval time.Duration
}

// Conf config for Pinger
type Conf struct {
	TimeOut  time.Duration
	Interval time.Duration
	Workers  int
}

// New return new pinger
func New(c Conf) (*Pinger, error) {
	if c.Workers == 0 {
		c.Workers = 1000
	}
	if c.TimeOut == 0 {
		c.TimeOut = time.Second * 10
	}
	if c.Interval == 0 {
		c.Interval = time.Millisecond * 1000
	}
	var p Pinger
	p.db = make(map[ping]chan Stat)
	p.ch = make(chan pingParam)
	p.timeout = c.TimeOut
	p.interval = c.Interval
	err := p.listen()
	for x := 0; x < c.Workers; x++ {
		go p.pingWorker()
	}
	return &p, err
}

// worker
func (p *Pinger) pingWorker() error {
	p.workerwg.Add(1)
	for {
		v, ok := <-p.ch
		if !ok {
			break
		}
		st, err := p.rping(v)
		if err != nil {
			return err
		}
		v.rchan <- st
	}
	p.workerwg.Done()
	return nil
}

// Close ping channel and quit all workers
func (p *Pinger) Close() {
	close(p.ch)
	p.workerwg.Wait()
}

// Ping add ping to channel queue
func (p *Pinger) Ping(ip string, cnt int) ([]Stat, error) {
	var pp pingParam
	pp.rchan = make(chan []Stat)
	pp.ip = ip
	pp.cnt = cnt
	p.ch <- pp
	res := <-pp.rchan
	return res, nil
}

// listen start listen icmp proccess in background
func (p *Pinger) listen() (err error) {
	p.c, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}

	rb := make([]byte, 1500)
	go func() {
		for {
			n, peer, err := p.c.ReadFrom(rb)
			if err != nil {
				log.Fatal(err)
			}
			rm, err := icmp.ParseMessage(1, rb[:n])
			if err != nil {
				log.Fatal(err)
			}

			switch body := rm.Body.(type) {
			case *icmp.Echo:

				p.dbmut.Lock()
				ch, ok := p.db[ping{IP: peer.String(), Seq: body.Seq}]
				p.dbmut.Unlock()
				if ok {
					var st Stat
					st.Recv = true
					st.RecvTime = time.Now()
					ch <- st
				}
			default:
				//Go critic, hello
			}
		}
	}()
	return nil
}

// generate data for ping body
func generateData(c int) []byte {
	var date []byte
	for x := 0; x < c; x++ {
		date = append(date, byte(x))
	}
	return date
}

// rping real ping non public
func (p *Pinger) rping(pp pingParam) (ra []Stat, err error) {
	var res Stat
	var wg sync.WaitGroup

	for x := 0; x < pp.cnt; x++ {
		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: os.Getpid() & 0xffff, Seq: x,
				Data: []byte(generateData(128)),
			},
		}

		wb, err := wm.Marshal(nil)
		if err != nil {
			return ra, err
		}

		if _, err := p.c.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(pp.ip)}); err != nil {
			return ra, err
		}
		res.SendTime = time.Now()
		p.dbmut.Lock()
		p.db[ping{IP: pp.ip, Seq: x}] = make(chan Stat)
		p.dbmut.Unlock()

		go func(x int, wg *sync.WaitGroup, res Stat) {
			wg.Add(1)
			p.dbmut.Lock()
			ch := p.db[ping{IP: pp.ip, Seq: x}]
			p.dbmut.Unlock()
			select {
			case st := <-ch:
				p.dbmut.Lock()
				delete(p.db, ping{IP: pp.ip, Seq: x})
				p.dbmut.Unlock()
				st.SendTime = res.SendTime
				st.Recv = true
				st.RecvTime = time.Now()
				ra = append(ra, st)
				wg.Done()
			case <-time.After(p.timeout):
				res.Recv = false
				p.dbmut.Lock()
				delete(p.db, ping{IP: pp.ip, Seq: x})
				p.dbmut.Unlock()
				ra = append(ra, res)
				wg.Done()
			}
		}(x, &wg, res)

		time.Sleep(p.interval)
	}
	wg.Wait()
	return ra, nil
}
