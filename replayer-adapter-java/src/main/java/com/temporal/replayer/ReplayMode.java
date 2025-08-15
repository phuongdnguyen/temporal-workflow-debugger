package com.temporal.replayer;

/**
 * Defines the mode of workflow replay for debugging.
 */
public enum ReplayMode {
    /**
     * Standalone mode: replay with history file and local breakpoints
     */
    STANDALONE("standalone"),
    
    /**
     * IDE mode: replay with debugger UI integration
     */
    IDE("ide");
    
    private final String value;
    
    ReplayMode(String value) {
        this.value = value;
    }
    
    public String getValue() {
        return value;
    }
    
    @Override
    public String toString() {
        return value;
    }
}
