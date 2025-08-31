.PHONY: help clean compile test run package worker starter

# Default target
help:
	@echo "Available targets:"
	@echo "  clean     - Clean compiled classes and artifacts"
	@echo "  compile   - Compile the project"
	@echo "  test      - Run tests"
	@echo "  run       - Run the example (requires history.json)"
	@echo "  package   - Create executable JAR"
	@echo "  worker    - Start the workflow worker"
	@echo "  starter   - Start a new workflow execution"
	@echo "  help      - Show this help message"

# Clean compiled classes and artifacts
clean:
	mvn clean

# Compile the project
compile: clean
	mvn compile

# Run tests
test: compile
	mvn test

# Run the example (requires history.json file)
run: compile
	@if [ ! -f "history.json" ]; then \
		echo "Error: history.json file not found. Please place a history file in the project root."; \
		exit 1; \
	fi
	mvn exec:java -Dexec.mainClass="com.temporal.example.SimpleWorkflowMain"

# Create executable JAR
package: compile
	mvn package

# Start the workflow worker
worker: compile
	@echo "Starting SimpleWorkflow worker..."
	@echo "Make sure Temporal server is running (e.g., temporal server start-dev)"
	@echo "Press Ctrl+C to stop the worker"
	mvn exec:java -Dexec.mainClass="com.temporal.example.SimpleWorkflowWorker"

# Start a new workflow execution
starter: compile
	@echo "Starting a new SimpleWorkflow execution..."
	@echo "Make sure Temporal server is running and worker is started"
	mvn exec:java -Dexec.mainClass="com.temporal.example.SimpleWorkflowStarter"

# Install dependencies
deps:
	mvn dependency:resolve

# Show project info
info:
	@echo "Project: Simple Workflow Example"
	@echo "Java version: $(shell java -version 2>&1 | head -n 1)"
	@echo "Maven version: $(shell mvn -version 2>&1 | head -n 1)"
	@echo "Project structure:"
	@find src -name "*.java" | sort
