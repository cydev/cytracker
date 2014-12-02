package cytracker

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScrapeURL(t *testing.T) {
	Convey("Scrape URLs", t, func() {
		tests := []struct{ announce, scrape string }{
			{"", ""},
			{"foo", ""},
			{"x/announce", "x/scrape"},
			{"x/announce?ad#3", "x/scrape?ad#3"},
			{"announce/x", ""},
		}
		for _, test := range tests {
			scrape := ScrapePattern(test.announce)
			So(scrape, ShouldEqual, test.scrape)
		}
	})
}
