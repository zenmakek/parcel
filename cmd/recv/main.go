package main

import (
	"fmt"
	"net"
	"os"

	"github.com/zenmakek/parcel/client/transfer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: recv <otp>")
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("[error] could not connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	homeDir, _ := os.UserHomeDir()
	downloadDir := homeDir + "/Downloads"

	if err := transfer.ReceiveFile(conn, os.Args[1], downloadDir); err != nil {
		fmt.Println("[error]", err)
		os.Exit(1)
	}
}
