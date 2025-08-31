package com.temporal.example;

import com.temporal.replayer.ReplayMode;
import com.temporal.replayer.ReplayOptions;
import com.temporal.replayer.ReplayerAdapter;
import io.temporal.worker.WorkerOptions;

import java.nio.file.Paths;

/**
 * Main class demonstrating how to use the Temporal workflow debugger replayer adapter.
 * This example shows both standalone mode (using local history files) and IDE mode.
 */
public class SimpleWorkflowMain {
    
    public static void main(String[] args) {
        try {
            // Set breakpoints at specific event IDs for debugging
            ReplayerAdapter.setBreakpoints(java.util.Arrays.asList(3, 9, 15));
            
            // Configure replay mode (STANDALONE or IDE)
            ReplayerAdapter.setReplayMode(ReplayMode.STANDALONE);
            
            // Create replay options using the builder pattern
            ReplayOptions options = new ReplayOptions.Builder()
                .setWorkerOptions(WorkerOptions.getDefaultInstance())
                .setHistoryFilePath(Paths.get("history.json").toAbsolutePath().toString())
                .build();
            
            // Replay the workflow - use the concrete implementation class, not the interface
            ReplayerAdapter.replay(options, SimpleWorkflowImpl.class);
            
            System.out.println("Workflow replay completed successfully!");
            
        } catch (Exception e) {
            System.err.println("Error during workflow replay: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
