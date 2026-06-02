package cli

import "fmt"

func ShowSettingsMenu() {
	for {
		clearScreen()
		fmt.Println()
		fmt.Println("  [ Settings ]")
		fmt.Println()
		fmt.Println("  [1] Download Directory  →  ~/Downloads (default)")
		fmt.Println("  [2] Relay Server        →  localhost:8080 (default)")
		fmt.Println("  [0] Back")
		fmt.Println()

		choice := readInput("  > ")

		switch choice {
		case "1":
			fmt.Println("\n  Custom download directory support coming soon.")
			readInput("  Press Enter to continue...")
		case "2":
			fmt.Println("\n  Custom relay server support coming soon.")
			readInput("  Press Enter to continue...")
		case "0":
			return
		default:
			fmt.Println("\n  Invalid option.")
			readInput("  Press Enter to continue...")
		}
	}
}
