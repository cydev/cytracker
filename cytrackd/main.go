package main

import (
	"flag"
	"log"

	"github.com/cydev/cytracker"
)

var (
	bindAddr = flag.String("addr", ":8080", "Creates a tracker serving the given torrent file on the given address")
)

func main() {
	log.Println("starting tracker on", *bindAddr)
	if err := cytracker.StartTracker(*bindAddr, flag.Args()); err != nil {
		log.Fatal(err)
	}
}
