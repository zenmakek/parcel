package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

func readInput(prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func Run() {
	for {
		clearScreen()
		fmt.Println()
		fmt.Println("  ██████╗  █████╗ ██████╗  ██████╗███████╗██╗     ")
		fmt.Println("  ██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔════╝██║     ")
		fmt.Println("  ██████╔╝███████║██████╔╝██║     █████╗  ██║     ")
		fmt.Println("  ██╔═══╝ ██╔══██║██╔══██╗██║     ██╔══╝  ██║     ")
		fmt.Println("  ██║     ██║  ██║██║  ██║╚██████╗███████╗███████╗")
		fmt.Println("  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚══════╝╚══════╝")
		fmt.Println()
		fmt.Println("  No accounts. No links. Just a code.")
		fmt.Println()
		fmt.Println("  [1] Send")
		fmt.Println("  [2] Receive")
		fmt.Println("  [3] Settings")
		fmt.Println("  [0] Exit")
		fmt.Println()

		choice := readInput("  > ")

		switch choice {
		case "1":
			ShowSendMenu()
		case "2":
			ShowReceiveMenu()
		case "3":
			ShowSettingsMenu()
		case "0":
			clearScreen()
			fmt.Println()
			fmt.Println("  Goodbye.")
			fmt.Println()
			os.Exit(0)
		default:
			fmt.Println("\n  Invalid option. Press Enter to continue.")
			reader.ReadString('\n')
		}
	}
}
