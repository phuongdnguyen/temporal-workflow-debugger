package com.temporal.replayer;

import io.temporal.worker.WorkerOptions;

/**
 * Configuration options for workflow replay.
 */
public class ReplayOptions {
    private final WorkerOptions workerOptions;
    private final String historyFilePath;
    
    private ReplayOptions(Builder builder) {
        this.workerOptions = builder.workerOptions;
        this.historyFilePath = builder.historyFilePath;
    }
    
    public WorkerOptions getWorkerOptions() {
        return workerOptions;
    }
    
    public String getHistoryFilePath() {
        return historyFilePath;
    }
    
    public static class Builder {
        private WorkerOptions workerOptions;
        private String historyFilePath;
        
        public Builder() {
            this.workerOptions = WorkerOptions.getDefaultInstance();
        }
        
        /**
         * Sets the worker options for replay configuration.
         * 
         * @param workerOptions the worker options
         * @return this builder
         */
        public Builder setWorkerOptions(WorkerOptions workerOptions) {
            this.workerOptions = workerOptions;
            return this;
        }
        
        /**
         * Sets the path to the history file (required for STANDALONE mode).
         * 
         * @param historyFilePath absolute path to the history JSON file
         * @return this builder
         */
        public Builder setHistoryFilePath(String historyFilePath) {
            this.historyFilePath = historyFilePath;
            return this;
        }
        
        public ReplayOptions build() {
            return new ReplayOptions(this);
        }
    }
}
