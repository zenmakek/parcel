package main

import (
	"fmt"
	"os"

	"github.com/zenmakek/parcel/server/relay"
	"github.com/zenmakek/parcel/server/tracker"
)

func main() {
	fmt.Println("[parcel] starting relay server...")
	fmt.Println("[parcel] starting tracker server...")

	go func() {
		t := tracker.New()
		if err := t.Start(); err != nil {
			fmt.Printf("[tracker] fatal: %v\n", err)
			os.Exit(1)
		}
	}()

	s := relay.New()
	if err := s.Start(); err != nil {
		fmt.Printf("[parcel] fatal: %v\n", err)
		os.Exit(1)
	}
}