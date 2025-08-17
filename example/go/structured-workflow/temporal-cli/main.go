package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start", "worker":
		fmt.Println("Starting Temporal worker...")
		if err := startWorker(); err != nil {
			fmt.Printf("Error starting worker: %v\n", err)
			os.Exit(1)
		}
	case "run", "workflow":
		fmt.Println("Running workflow...")
		if err := runWorkflow(); err != nil {
			fmt.Printf("Error running workflow: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Temporal CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  go run . start    - Start the Temporal worker")
	fmt.Println("  go run . worker   - Start the Temporal worker (alias)")
	fmt.Println("  go run . run      - Run a workflow execution")
	fmt.Println("  go run . workflow - Run a workflow execution (alias)")
}
