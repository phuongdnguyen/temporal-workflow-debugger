package com.temporal.replayer.example;

import com.temporal.replayer.ReplayMode;
import com.temporal.replayer.ReplayOptions;
import com.temporal.replayer.ReplayerAdapter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Arrays;

/**
 * Example usage of the Temporal Java Replayer Adapter.
 * 
 * This example demonstrates both standalone and IDE modes for workflow debugging.
 */
public class Example {
    
    private static final Logger logger = LoggerFactory.getLogger(Example.class);
    
    public static void main(String[] args) {
        try {
            // Example 1: Standalone Mode
            runStandaloneExample();
            
            // Example 2: IDE Mode
            runIdeExample();
            
        } catch (Exception e) {
            logger.error("Example execution failed", e);
        }
    }
    
    /**
     * Example of running replay in standalone mode with local breakpoints.
     */
    private static void runStandaloneExample() throws Exception {
        logger.info("Running Standalone Mode Example");
        
        // Set breakpoints at specific event IDs
        ReplayerAdapter.setBreakpoints(Arrays.asList(1, 5, 10, 15));
        
        // Set standalone mode
        ReplayerAdapter.setReplayMode(ReplayMode.STANDALONE);
        
        // Configure replay options with history file
        ReplayOptions opts = new ReplayOptions.Builder()
            .setHistoryFilePath("/path/to/your/workflow-history.json")
            .build();
        
        // Replay workflow
        // Note: Replace ExampleWorkflow.class with your actual workflow interface
        ReplayerAdapter.replay(opts, ExampleWorkflow.class);
        
        logger.info("Standalone mode example completed");
    }
    
    /**
     * Example of running replay in IDE mode with debugger integration.
     */
    private static void runIdeExample() throws Exception {
        logger.info("Running IDE Mode Example");
        
        // Set IDE mode
        ReplayerAdapter.setReplayMode(ReplayMode.IDE);
        
        // Configure replay options (history will be fetched from IDE)
        ReplayOptions opts = new ReplayOptions.Builder()
            .build();
        
        // Replay workflow with IDE integration
        // Note: Replace ExampleWorkflow.class with your actual workflow interface
        ReplayerAdapter.replay(opts, ExampleWorkflow.class);
        
        logger.info("IDE mode example completed");
    }
    
    /**
     * Example workflow interface.
     * Replace this with your actual workflow interface.
     */
    public interface ExampleWorkflow {
        // TODO: Define your workflow methods here
        String runWorkflow(String input);
    }
    
    /**
     * Example workflow implementation.
     * Replace this with your actual workflow implementation.
     */
    public static class ExampleWorkflowImpl implements ExampleWorkflow {
        
        @Override
        public String runWorkflow(String input) {
            // TODO: Implement your workflow logic here
            logger.info("Executing workflow with input: {}", input);
            return "Result: " + input;
        }
    }
}
