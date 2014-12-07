package cytracker

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

// key is the client's listen address, in the form IP:port
type trackerPeers map[string]*trackerPeer

type trackerPeer struct {
	listenAddr *net.TCPAddr
	id         string
	lastSeen   time.Time
	uploaded   uint64
	downloaded uint64
	left       uint64
}

func (t trackerPeers) Add(key string, peer *trackerPeer) {
	log.Printf("Peer %s joined", key)
	t[key] = peer
}

func (t trackerPeers) Remove(key string) {
	log.Printf("Peer %s removed", key)
	delete(t, key)
}

func (t trackerPeers) pickRandomPeers(peerKey string, compact bool, count int) (peers []string) {
	// Cheesy approximation to picking randomly from all peers.
	// Depends upon the implementation detail that map iteration is pseudoRandom
	for k, v := range t {
		if k == peerKey {
			continue
		}
		if compact && v.listenAddr.IP.To4() == nil {
			continue
		}
		peers = append(peers, k)
		if len(peers) == count {
			break
		}
	}
	return
}

func (t trackerPeers) writeCompactPeers(b *bytes.Buffer, keys []string) (err error) {
	for _, k := range keys {
		p := t[k]
		la := p.listenAddr
		ip4 := la.IP.To4()
		if ip4 == nil {
			err = fmt.Errorf("Can't write a compact peer for a non-IPv4 peer %v %v", k, p.listenAddr.String())
			return
		}
		_, err = b.Write(ip4)
		if err != nil {
			return
		}
		port := la.Port
		portBytes := []byte{byte(port >> 8), byte(port)}
		_, err = b.Write(portBytes)
		if err != nil {
			return
		}
	}
	return
}

func (t trackerPeers) getPeers(keys []string, noPeerID bool) (peers []bmap, err error) {
	for _, k := range keys {
		p := t[k]
		la := p.listenAddr
		var peer bmap = make(bmap)
		if !noPeerID {
			peer["peer id"] = p.id
		}
		peer["ip"] = la.IP.String()
		peer["port"] = strconv.Itoa(la.Port)
		peers = append(peers, peer)
	}
	return
}

func (t trackerTorrents) reap(deadline time.Time) {
	for _, tt := range t {
		tt.reap(deadline)
	}
}

func (t trackerPeers) reap(deadline time.Time) {
	for address, peer := range t {
		if deadline.After(peer.lastSeen) {
			log.Println("reaping", address)
			delete(t, address)
		}
	}
}

func (t *trackerPeer) isComplete() bool {
	return t.left == 0
}
