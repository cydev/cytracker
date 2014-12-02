package cytracker

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	trackerStopTimeOut   = time.Second * 1
	trackerStartDuration = time.Millisecond * 5
	trackerAddr          = ":0"
)

var (
	timedOutError = errors.New("Timed out")
)

func TestTrackerStart(t *testing.T) {
	Convey("Tracker stop on channel close", t, func() {
		So(trackerStartDuration, ShouldBeLessThan, trackerStopTimeOut)
		stop := make(chan os.Signal)
		errors := make(chan error)
		go func() {
			log.Println("starting tracker")
			errors <- startStoppableTracker(trackerAddr, nil, stop)
			log.Println("tracker returned")
		}()
		go func() {
			time.Sleep(trackerStartDuration)
			log.Println("closing channel")
			close(stop)
		}()
		timer := time.NewTimer(trackerStopTimeOut)
		select {
		case <-timer.C:
			So(timedOutError, ShouldBeNil)
		case err := <-errors:
			So(err, ShouldBeNil)
		}
	})
}
