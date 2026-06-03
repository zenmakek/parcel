package main

import (
	"fmt"
	"os"

	"github.com/zenmakek/parcel/server/relay"
)

func main() {
	fmt.Println("[parcel] starting relay server...")

	s := relay.New()
	if err := s.Start(); err != nil {
		fmt.Printf("[parcel] fatal %v\n", err)
		os.Exit(1)
	}
}
