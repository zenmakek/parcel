package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenmakek/parcel/client/archive"
	"github.com/zenmakek/parcel/client/transfer"
)

func ShowReceiveMenu() {
	clearScreen()
	fmt.Println()
	fmt.Println("  [ Receive ]")
	fmt.Println()

	otp := readInput("  Enter 6-digit OTP: ")
	otp = strings.TrimSpace(otp)

	if len(otp) != 6 {
		fmt.Println("\n  [error] OTP must be exactly 6 digits.")
		readInput("  Press Enter to go back...")
		return
	}

	for _, c := range otp {
		if c < '0' || c > '9' {
			fmt.Println("\n  [error] OTP must be numeric.")
			readInput("  Press Enter to go back...")
			return
		}
	}

	fmt.Println()
	fmt.Println("  Connecting to relay server...")

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("\n  [error] Could not connect to relay: %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}
	defer conn.Close()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("\n  [error] Could not determine home directory: %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	downloadDir := filepath.Join(homeDir, "Downloads")

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("\n  [error] Could not create download directory: %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	fmt.Println("  Waiting for transfer metadata...")
	fmt.Println()

	receivedPath, isArchive, err := transfer.ReceiveFile(conn, otp, downloadDir)
	if err != nil {
		fmt.Printf("\n  [error] %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	if isArchive {
		fmt.Println("  Extracting archive...")

		if err := archive.ExtractArchive(receivedPath, downloadDir); err != nil {
			fmt.Printf("\n  [error] Extraction failed: %v\n", err)
			readInput("  Press Enter to go back...")
			return
		}

		if err := archive.CleanupArchive(receivedPath); err != nil {
			fmt.Printf("\n  [warn] Could not clean up archive: %v\n", err)
		}

		folderName := strings.TrimSuffix(filepath.Base(receivedPath), ".tar.gz")
		fmt.Println()
		fmt.Println("  ✓ Transfer complete.")
		fmt.Printf("  Saved to: %s\n", filepath.Join(downloadDir, folderName))
	} else {
		fmt.Println()
		fmt.Println("  ✓ Transfer complete.")
		fmt.Printf("  Saved to: %s\n", receivedPath)
	}

	fmt.Println()
	readInput("  Press Enter to go back...")
}