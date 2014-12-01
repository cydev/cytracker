package cytracker

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"time"

	"github.com/jackpal/Taipei-Torrent/torrent"
)

type Tracker struct {
	Announce string
	Addr     string
	ID       string
	done     chan struct{}
	m        sync.Mutex // Protects l and t
	l        net.Listener
	torrents trackerTorrents
}

type bmap map[string]interface{}

func randomHexString(n int) string {
	return randomString("0123456789abcdef", n)
}

func randomString(s string, n int) string {
	b := make([]byte, n)
	slen := len(s)
	for i := 0; i < n; i++ {
		b[i] = b[rand.Intn(slen)]
	}
	return string(b)
}

// Start a tracker and run it until interrupted.
func StartTracker(addr string, torrentFiles []string) (err error) {
	quitChan := listenSigInt()
	return startStoppableTracker(addr, torrentFiles, quitChan)
}

func startStoppableTracker(addr string, torrents []string, stop chan os.Signal) (err error) {
	t := NewTracker()
	t.Addr = addr
	for _, torrentFile := range torrents {
		var metaInfo *torrent.MetaInfo
		metaInfo, err = torrent.GetMetaInfo(nil, torrentFile)
		if err != nil {
			return
		}
		name := metaInfo.Info.Name
		if name == "" {
			name = path.Base(torrentFile)
		}
		err = t.Register(metaInfo.InfoHash, name)
		if err != nil {
			return
		}
	}
	go func() {
		select {
		case <-stop:
			log.Printf("got control-C")
			t.Quit()
		}
	}()

	return t.ListenAndServe()
}

func listenSigInt() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	return c
}

func NewTracker() *Tracker {
	return &Tracker{Announce: "/announce", torrents: NewTrackerTorrents()}
}

func (t *Tracker) ListenAndServe() (err error) {
	t.done = make(chan struct{})
	if t.ID == "" {
		t.ID = randomHexString(20)
	}
	addr := t.Addr
	if addr == "" {
		addr = ":80"
	}
	var l net.Listener
	l, err = net.Listen("tcp", addr)
	if err != nil {
		return
	}
	t.m.Lock()
	t.l = l
	t.m.Unlock()
	serveMux := http.NewServeMux()
	announce := t.Announce
	if announce == "" {
		announce = "/"
	}
	serveMux.HandleFunc(announce, t.handleAnnounce)
	scrape := ScrapePattern(announce)
	if scrape != "" {
		serveMux.HandleFunc(scrape, t.handleScrape)
	}
	go t.reaper()
	// This statement will not return until there is an error or the t.l channel is closed
	err = http.Serve(l, serveMux)
	if err != nil {
		select {
		case <-t.done:
			// We're finished. Err is probably a "use of closed network connection" error.
			err = nil
		default:
			// Not finished
		}
	}
	return
}

func (t *Tracker) Quit() (err error) {
	select {
	case <-t.done:
		err = fmt.Errorf("Already done")
		return
	default:
	}
	var l net.Listener
	t.m.Lock()
	l = t.l
	t.m.Unlock()
	l.Close()
	close(t.done)
	return
}

func (t *Tracker) Register(infoHash, name string) (err error) {
	log.Printf("Register(%#v,%#v)", infoHash, name)
	t.m.Lock()
	defer t.m.Unlock()
	err = t.torrents.register(infoHash, name)
	return
}

func (t *Tracker) Unregister(infoHash string) (err error) {
	t.m.Lock()
	defer t.m.Unlock()
	err = t.torrents.unregister(infoHash)
	return
}

func (t *Tracker) reaper() {
	checkDuration := 30 * time.Minute
	reapDuration := 2 * checkDuration
	ticker := time.Tick(checkDuration)
	select {
	case <-t.done:
		return
	case <-ticker:
		t.m.Lock()
		deadline := time.Now().Add(-reapDuration)
		t.torrents.reap(deadline)
		t.m.Unlock()
	}
}

func blank(s string) bool {
	return len(s) == 0
}
