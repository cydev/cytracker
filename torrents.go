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
	log.Println("registering", infoHash, "as", name)
	if t2, ok := t[infoHash]; ok {
		err = fmt.Errorf("Already have a torrent %#v with infoHash %v", t2.name, infoHash)
		return
	}
	t[infoHash] = &trackerTorrent{name: name, peers: make(trackerPeers)}
	return
}

func (t trackerTorrents) unregister(infoHash string) (err error) {
	log.Println("unregistering", infoHash)
	delete(t, infoHash)
	return
}

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
	peerKey := peerListenAddress.String()
	var peer *trackerPeer
	var ok bool
	if peer, ok = t.peers[peerKey]; ok {
		// Does the new peer match the old peer?
		if peer.id != params.peerID {
			log.Printf("Peer changed ID. %#v != %#v", peer.id, params.peerID)
			delete(t.peers, peerKey)
			peer = nil
		}
	}
	if peer == nil {
		peer = &trackerPeer{
			listenAddr: peerListenAddress,
			id:         params.peerID,
		}
		t.peers[peerKey] = peer
		log.Printf("Peer %s joined", peerKey)
	}
	peer.lastSeen = now
	peer.uploaded = params.uploaded
	peer.downloaded = params.downloaded
	peer.left = params.left
	switch params.event {
	default:
		// TODO(jackpal):maybe report this as a warning
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

	completeCount, incompleteCount := t.countPeers()
	response["complete"] = completeCount
	response["incomplete"] = incompleteCount

	peerCount := len(t.peers)
	numWant := params.numWant
	const DEFAULT_PEER_COUNT = 50
	if numWant <= 0 || numWant > DEFAULT_PEER_COUNT {
		numWant = DEFAULT_PEER_COUNT
	}
	if numWant > peerCount {
		numWant = peerCount
	}

	peerKeys := t.peers.pickRandomPeers(peerKey, params.compact, numWant)
	if params.compact {
		var b bytes.Buffer
		err = t.peers.writeCompactPeers(&b, peerKeys)
		if err != nil {
			return
		}
		response["peers"] = string(b.Bytes())
	} else {
		var peers []bmap
		noPeerID := params.noPeerID
		peers, err = t.peers.getPeers(peerKeys, noPeerID)
		if err != nil {
			return
		}
		response["peers"] = peers
	}
	return
}

func (t *trackerTorrent) reap(deadline time.Time) {
	t.peers.reap(deadline)
}
