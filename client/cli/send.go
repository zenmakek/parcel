package cli

import (
	"fmt"
	"net"

	"github.com/zenmakek/parcel/client/archive"
	"github.com/zenmakek/parcel/client/transfer"
)

func ShowSendMenu() {
	clearScreen()
	fmt.Println()
	fmt.Println("  [ Send ]")
	fmt.Println()

	path := readInput("  Enter file or folder path: ")
	if path == "" {
		fmt.Println("\n  No path entered.")
		readInput("  Press Enter to go back...")
		return
	}

	if err := transfer.ValidatePath(path); err != nil {
		fmt.Printf("\n  [error] %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	meta, err := transfer.Inspect(path)
	if err != nil {
		fmt.Printf("\n  [error] %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	fmt.Println()
	fmt.Printf("  Name  : %s\n", meta.Filename)
	fmt.Printf("  Size  : %s\n", meta.HumanSize())
	if meta.IsArchive {
		fmt.Println("  Type  : Folder (will be archived as .tar.gz)")
	} else {
		fmt.Println("  Type  : File")
	}
	fmt.Println()

	confirm := readInput("  Send this? [y/n]: ")
	if confirm != "y" && confirm != "Y" {
		fmt.Println("\n  Cancelled.")
		readInput("  Press Enter to go back...")
		return
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

	if meta.IsArchive {
		fmt.Println("  Archiving folder...")
		archivePath := archive.ArchivePath(meta.OriginalPath)

		if err := archive.CompressFolder(meta.OriginalPath, archivePath); err != nil {
			fmt.Printf("\n  [error] Failed to archive folder: %v\n", err)
			readInput("  Press Enter to go back...")
			return
		}

		defer archive.CleanupArchive(archivePath)

		meta, err = transfer.Inspect(archivePath)
		if err != nil {
			fmt.Printf("\n  [error] Failed to inspect archive: %v\n", err)
			readInput("  Press Enter to go back...")
			return
		}

		meta.IsArchive = true
	}

	fmt.Println()
	fmt.Println("  ┌─────────────────────────────┐")
	fmt.Println("  │                             │")
	fmt.Printf("  │   OTP will appear here...   │\n")
	fmt.Println("  │                             │")
	fmt.Println("  └─────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Waiting for OTP from relay...")

	if err := transfer.SendFile(conn, meta); err != nil {
		fmt.Printf("\n  [error] Transfer failed: %v\n", err)
		readInput("  Press Enter to go back...")
		return
	}

	fmt.Println()
	fmt.Println("  ✓ Transfer complete.")
	fmt.Println()
	readInput("  Press Enter to go back...")
}
