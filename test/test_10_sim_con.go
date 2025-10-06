package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
)

/*
*	Tests the app with 10 simultinous connections
*	with two connectors:
*	C1 ports config: 3400
*	C2 ports config: 3150
 */
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m" // Resets all attributes
)

func main() {
	log.Println("Testing 10 simultanious connections.")

	cmd1 := exec.Command("./connector_bin", "", "")
	cmd2 := exec.Command("./server_bin", "", "")

	runWithPrefix("CONNECTOR", cmd1, ColorBlue)
	runWithPrefix("SERVER", cmd2, ColorYellow)

	select {} // block forever so both can run
}

func runWithPrefix(name string, cmd *exec.Cmd, color string) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start %s: %v", name, err)
	}

	// stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf("%s[%s][OUT] %s%s\n", color, name, scanner.Text(), ColorReset)
		}
	}()

	// stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("%s[%s][ERR] %s%s\n", color, name, scanner.Text(), ColorReset)
		}
	}()

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("[%s] exited with error: %v", name, err)
		}
	}()
}
