package cli

import "fmt"

func ShowReceiveMenu() {
	clearScreen()
	fmt.Println()
	fmt.Println("  [ Receive ]")
	fmt.Println()
	fmt.Println("  This feature is coming in Phase 11.")
	fmt.Println("  You will enter a 6-digit OTP to receive files.")
	fmt.Println()
	readInput("  Press Enter to go back...")
}
