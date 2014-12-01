package cytracker

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/jackpal/bencode-go"
)

func ScrapePattern(announcePattern string) string {
	lastSlashIndex := strings.LastIndex(announcePattern, "/")
	if lastSlashIndex >= 0 {
		firstPart := announcePattern[0 : lastSlashIndex+1]
		lastPart := announcePattern[lastSlashIndex+1:]
		announce := "announce"
		if strings.HasPrefix(lastPart, announce) {
			afterAnnounce := lastPart[len(announce):]
			return strings.Join([]string{firstPart, "scrape", afterAnnounce}, "")
		}
	}
	return ""
}

func (t *Tracker) handleScrape(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	infoHashes := r.URL.Query()["info_hash"]
	response := make(bmap)
	response["files"] = t.torrents.scrape(infoHashes)
	var b bytes.Buffer
	err := bencode.Marshal(&b, response)
	if err == nil {
		w.Write(b.Bytes())
	}
}

// scrape returns data about torrent
func (t *trackerTorrent) scrape() (response bmap) {
	response = make(bmap)
	completeCount, incompleteCount := t.countPeers()
	response["complete"] = completeCount
	response["incomplete"] = incompleteCount
	response["downloaded"] = t.downloaded
	if t.name != "" {
		response["name"] = t.name
	}
	return
}
