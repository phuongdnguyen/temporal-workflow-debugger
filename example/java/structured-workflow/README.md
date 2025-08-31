# Java Structured Workflow Example

This example demonstrates a more complex Temporal workflow with child workflows, signals, and queries using the Java replayer adapter.

## Overview

The structured workflow example includes:
- A main user onboarding workflow with multiple steps
- Child workflow for account setup
- Signal methods for updating preferences during execution
- Query methods for checking workflow status
- Complex activity orchestration
- Error handling and retry logic

## Project Structure

```
src/main/java/com/temporal/example/
├── StructuredWorkflowMain.java      # Main class with replayer setup
├── UserOnboardingWorkflow.java      # Main workflow interface
├── UserOnboardingWorkflowImpl.java  # Main workflow implementation
├── AccountSetupWorkflow.java        # Child workflow interface
├── OnboardingActivities.java        # Activity interface
├── OnboardingResult.java            # Result data class
├── UserPreferences.java             # Preferences data class
└── OnboardingStatus.java            # Status data class
```

## Workflow Flow

The user onboarding workflow follows this sequence:

1. **User Validation** - Validates the user ID
2. **Profile Creation** - Creates a user profile
3. **Account Setup** - Executes a child workflow for account setup
4. **Preferences Configuration** - Configures user preferences (if provided via signal)
5. **Finalization** - Completes the onboarding process

## Features Demonstrated

### Child Workflows
- The main workflow spawns a child workflow for account setup
- Demonstrates workflow composition and coordination

### Signals
- `updatePreferences()` signal allows updating user preferences during execution
- Shows how to handle external input during workflow execution

### Queries
- `getStatus()` query provides current workflow status
- Useful for monitoring workflow progress

### Error Handling
- Comprehensive error handling with status updates
- Retry logic for activities
- Graceful failure handling

## Usage

### Building
```bash
mvn clean compile
```

### Running
```bash
mvn exec:java -Dexec.mainClass="com.temporal.example.StructuredWorkflowMain"
```

### Configuration

Set breakpoints at specific event IDs:
```java
ReplayerAdapter.setBreakpoints(Arrays.asList(5, 12, 20, 28));
```

## Integration Points

This example integrates with:
- Temporal workflow debugger replayer adapter
- Support for both standalone and IDE modes
- Breakpoint debugging at specific event IDs
- History replay from local files or IDE sources

## Advanced Patterns

- **Workflow State Management**: Maintains workflow state across multiple steps
- **Activity Orchestration**: Coordinates multiple activities with proper error handling
- **Child Workflow Execution**: Demonstrates workflow composition
- **Signal and Query Handling**: Shows reactive workflow patterns
- **Progress Tracking**: Real-time status updates during execution

## Debugging Features

The replayer adapter provides:
- Event-level breakpoints for debugging
- Workflow execution monitoring
- History replay capabilities
- IDE integration support
- Comprehensive logging and error reporting
