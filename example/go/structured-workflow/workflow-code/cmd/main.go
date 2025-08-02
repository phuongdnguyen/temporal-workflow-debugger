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
	case "run", "worker":
		fmt.Println("Starting Temporal worker...")
		if err := runWorker(); err != nil {
			fmt.Printf("Error running worker: %v\n", err)
			os.Exit(1)
		}
	case "start", "workflow":
		fmt.Println("Starting workflow...")
		if err := startWorkflow(); err != nil {
			fmt.Printf("Error starting workflow: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run . run     - Start the Temporal worker")
	fmt.Println("  go run . worker  - Start the Temporal worker (alias)")
	fmt.Println("  go run . start   - Start a workflow execution")
	fmt.Println("  go run . workflow - Start a workflow execution (alias)")
}
