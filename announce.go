package cytracker

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

type announceParams struct {
	infoHash   string
	peerID     string
	ip         string // optional
	port       int
	uploaded   uint64
	downloaded uint64
	left       uint64
	compact    bool
	noPeerID   bool
	event      string
	numWant    int
	trackerID  string
}

type Values struct {
	url.Values
}

func (v Values) GetBool(key string) (b bool, err error) {
	val := v.Get(key)
	if val == "" {
		err = fmt.Errorf("Missing query parameter: %v", key)
		return
	}
	return strconv.ParseBool(val)
}

func getBool(v url.Values, key string) (b bool, err error) {
	val := v.Get(key)
	if val == "" {
		err = fmt.Errorf("Missing query parameter: %v", key)
		return
	}
	return strconv.ParseBool(val)
}

func (v Values) GetUint64(key string) (i uint64, err error) {
	return getUint64(v.Values, key)
}

func (v Values) GetInt(key string) (i int, err error) {
	val := v.Get(key)
	if val == "" {
		err = fmt.Errorf("Missing query parameter: %v", key)
		return
	}
	return strconv.Atoi(val)
}

func getUint64(v url.Values, key string) (i uint64, err error) {
	val := v.Get(key)
	if val == "" {
		err = fmt.Errorf("Missing query parameter: %v", key)
		return
	}
	return strconv.ParseUint(val, 10, 64)
}

func getUint(v url.Values, key string) (i int, err error) {
	var i64 uint64
	i64, err = getUint64(v, key)
	if err != nil {
		return
	}
	i = int(i64)
	return
}

const (
	paramInfoHash   = "info_hash"
	paramIP         = "ip"
	paramPeerID     = "peer_id"
	paramPort       = "port"
	paramUploaded   = "uploaded"
	paramDownloaded = "downloaded"
	paramLeft       = "left"
	paramCompact    = "compact"
	paramNoPeerID   = "no_peer_id"
	paramEvent      = "event"
	paramNumberWant = "numwant"
	paramTrackerID  = "trackerid"
)

func (a *announceParams) parse(u *url.URL) (err error) {
	q := Values{u.Query()}
	a.infoHash = q.Get(paramInfoHash)
	if blank(a.infoHash) {
		return fmt.Errorf("Missing info_hash")
	}
	a.ip = q.Get(paramIP)
	a.peerID = q.Get(paramPeerID)
	a.port, err = q.GetInt(paramPort)
	if err != nil {
		return
	}
	a.uploaded, err = q.GetUint64(paramUploaded)
	if err != nil {
		return
	}
	a.downloaded, err = q.GetUint64(paramDownloaded)
	if err != nil {
		return
	}
	a.left, err = q.GetUint64(paramLeft)
	if err != nil {
		return
	}
	if !blank(q.Get(paramCompact)) {
		a.compact, err = q.GetBool(paramCompact)
		if err != nil {
			return
		}
	}
	if !blank(q.Get(paramNoPeerID)) {
		a.noPeerID, err = q.GetBool(paramNoPeerID)
		if err != nil {
			return
		}
	}
	if !blank(q.Get(paramNumberWant)) {
		a.numWant, err = q.GetInt(paramNumberWant)
		if err != nil {
			return
		}
	}
	a.event = q.Get(paramEvent)
	a.trackerID = q.Get(paramTrackerID)
	return
}

func newTrackerPeerListenAddress(requestRemoteAddr string, params *announceParams) (addr *net.TCPAddr, err error) {
	var host string
	if !blank(params.ip) {
		host = params.ip
	} else {
		host, _, err = net.SplitHostPort(requestRemoteAddr)
		if err != nil {
			return
		}
	}
	return net.ResolveTCPAddr("tcp", net.JoinHostPort(host, strconv.Itoa(params.port)))
}

func (t *Tracker) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling announce")
	w.Header().Set("Content-Type", "text/plain")
	var (
		params            announceParams
		peerListenAddress *net.TCPAddr
		err               error
		b                 bytes.Buffer
		response          = make(bmap)
	)
	err = params.parse(r.URL)
	if err == nil {
		if params.trackerID != "" && params.trackerID != t.ID {
			err = fmt.Errorf("Incorrect tracker ID: %#v", params.trackerID)
		}
	}
	if err == nil {
		peerListenAddress, err = newTrackerPeerListenAddress(r.RemoteAddr, &params)
	}
	if err == nil {
		now := time.Now()
		t.m.Lock()
		err = t.torrents.handleAnnounce(now, peerListenAddress, &params, response)
		t.m.Unlock()
		if err == nil {
			response["interval"] = int64(30 * 60)
			response["tracker id"] = t.ID
		}
	}
	if err != nil {
		log.Printf("announce from %v failed: %#v", r.RemoteAddr, err.Error())
		errorResponse := make(bmap)
		errorResponse["failure reason"] = err.Error()
		err = bencode.Marshal(&b, errorResponse)
	} else {
		err = bencode.Marshal(&b, response)
	}
	if err == nil {
		w.Write(b.Bytes())
	}
}
