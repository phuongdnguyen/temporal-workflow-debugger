package com.temporal.example;

import com.temporal.replayer.ReplayMode;
import com.temporal.replayer.ReplayOptions;
import com.temporal.replayer.ReplayerAdapter;
import io.temporal.worker.WorkerOptions;

import java.nio.file.Paths;

/**
 * Main class for the structured workflow example.
 * This demonstrates debugging a more complex workflow with child workflows and signals.
 */
public class StructuredWorkflowMain {
    
    public static void main(String[] args) {
        try {
            // Set breakpoints at specific event IDs for debugging
            ReplayerAdapter.setBreakpoints(java.util.Arrays.asList(5, 12, 20, 28));
            
            // Configure replay mode (STANDALONE or IDE)
            ReplayerAdapter.setReplayMode(ReplayMode.STANDALONE);
            
            // Create replay options using the builder pattern
            ReplayOptions options = new ReplayOptions.Builder()
                .setWorkerOptions(WorkerOptions.getDefaultInstance())
                .setHistoryFilePath(Paths.get("history.json").toAbsolutePath().toString())
                .build();
            
            // Replay the workflow
            ReplayerAdapter.replay(options, UserOnboardingWorkflow.class);
            
            System.out.println("Structured workflow replay completed successfully!");
            
        } catch (Exception e) {
            System.err.println("Error during workflow replay: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
