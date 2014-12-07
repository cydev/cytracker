package cytracker

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"
)

type trackerTorrents map[string]*trackerTorrent

// Single-threaded imp
type trackerTorrent struct {
	name       string
	downloaded uint64
	peers      trackerPeers
}

const (
	paramComplete    = "complete"
	paramIncomplete  = "incomplete"
	paramPeers       = "peers"
	defaultPeerCount = 50
)

func NewTrackerTorrents() trackerTorrents {
	return make(trackerTorrents)
}

func (t trackerTorrents) handleAnnounce(now time.Time, peerListenAddress *net.TCPAddr, params *announceParams, response bmap) (err error) {
	log.Println("announce", params)
	torrent := t[params.infoHash]
	if torrent == nil {
		if err = t.register(params.infoHash, params.infoHash); err != nil {
			return err
		}
		torrent = t[params.infoHash]
	}
	return torrent.handleAnnounce(now, peerListenAddress, params, response)
}

func (t trackerTorrents) scrape(infoHashes []string) (files bmap) {
	files = make(bmap)
	if len(infoHashes) > 0 {
		for _, infoHash := range infoHashes {
			if torrent, ok := t[infoHash]; ok {
				files[infoHash] = torrent.scrape()
			}
		}
	} else {
		for infoHash, torrent := range t {
			files[infoHash] = torrent.scrape()
		}
	}
	return
}

func (t trackerTorrents) register(infoHash, name string) (err error) {
	log.Printf("registering %#v with infoHash %v", name, infoHash)
	if t2, ok := t[infoHash]; ok {
		return fmt.Errorf("Already have a torrent %#v with infoHash %v", t2.name, infoHash)
	}
	// MEMORY_ALLOCATION
	// TODO: use torrent pool
	t[infoHash] = &trackerTorrent{name: name, peers: make(trackerPeers)}
	return nil
}

func (t trackerTorrents) unregister(infoHash string) (err error) {
	log.Println("unregistering", infoHash)
	delete(t, infoHash)
	return
}

// TODO: move to structure fields and refactor
func (t *trackerTorrent) countPeers() (complete, incomplete int) {
	for _, p := range t.peers {
		if p.isComplete() {
			complete++
		} else {
			incomplete++
		}
	}
	return
}

func (t *trackerTorrent) handleAnnounce(now time.Time, peerListenAddress *net.TCPAddr, params *announceParams, response bmap) (err error) {
	var (
		// current peer
		peer       *trackerPeer
		peerExists bool
		peerKey    = peerListenAddress.String()
	)

	// checking peer existance
	if peer, peerExists = t.peers[peerKey]; peerExists {
		// checking peer ID persistance
		if peer.id != params.peerID {
			log.Printf("Peer changed ID. %#v != %#v", peer.id, params.peerID)
			// MEMORY_FREE
			t.peers.Remove(peerKey)
			peer = nil
		}
	}

	if peer == nil {
		// peer does not exist
		// creating peer
		// MEMORY_ALLOCATION
		// TODO: use peers buffer
		peer = &trackerPeer{
			listenAddr: peerListenAddress,
			id:         params.peerID,
		}
		t.peers.Add(peerKey, peer)
	}

	// updating params
	// TODO: refactor into function
	peer.lastSeen = now
	peer.uploaded = params.uploaded
	peer.downloaded = params.downloaded
	peer.left = params.left

	log.Printf("Peer %s Event %s", peerKey, params.event)
	// processing event
	switch params.event {
	default:
		// TODO: report this as a warning
		log.Printf("Peer %s Unknown event %s", peerKey, params.event)
	case "":
	case "started":
		// do nothing
	case "completed":
		t.downloaded++
		log.Printf("Peer %s completed. Total completions %d", peerKey, t.downloaded)
	case "stopped":
		// This client is reporting that they have stopped. Drop them from the peer table.
		// And don't send any peers, since they won't need them.
		log.Printf("Peer %s stopped", peerKey)
		delete(t.peers, peerKey)
		params.numWant = 0
	}

	// generating response
	response[paramComplete], response[paramIncomplete] = t.countPeers()

	// calculating peer count for response
	// TODO: extract to function
	peerCount := len(t.peers)
	numWant := params.numWant

	if numWant <= 0 || numWant > defaultPeerCount {
		// numWant incorrect
		numWant = defaultPeerCount
	}
	if numWant > peerCount {
		numWant = peerCount
	}

	// picking random peers from peerlist for current peer
	peerKeys := t.peers.pickRandomPeers(peerKey, params.compact, numWant)
	if params.compact {
		var b bytes.Buffer
		// MEMORY_ALLOCATION
		// TODO: use bytes buffer pool
		err = t.peers.writeCompactPeers(&b, peerKeys)
		if err != nil {
			return
		}
		response[paramPeers] = string(b.Bytes())
	} else {
		var peers []bmap
		// MEMORY_ALLOCATION
		// TODO: use bmap pool
		noPeerID := params.noPeerID
		peers, err = t.peers.getPeers(peerKeys, noPeerID)
		if err != nil {
			return
		}
		response[paramPeers] = peers
	}
	return
}

func (t *trackerTorrent) reap(deadline time.Time) {
	t.peers.reap(deadline)
}
