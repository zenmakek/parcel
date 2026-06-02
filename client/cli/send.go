package cli

import "fmt"

func ShowSendMenu() {
	clearScreen()
	fmt.Println()
	fmt.Println("  [ Send ]")
	fmt.Println()
	fmt.Println("  This feature is coming in Phase 10.")
	fmt.Println("  You will be able to select files and folders to send.")
	fmt.Println()
	readInput("  Press Enter to go back...")
}
