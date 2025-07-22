package com.temporal.wfdebugger.model;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * State model for the Temporal Workflow Debugger plugin.
 * This stores user configuration and session state.
 */
public class WfDebuggerState {
    
    // Configuration settings
    public String debugDirectory = "";
    public String tdlvBinaryPath = "tdlv"; // Default to PATH lookup
    
    // Session state
    public List<HistoryEvent> loadedEvents = new ArrayList<>();
    public Set<Long> enabledBreakpoints = new HashSet<>();
    
    // UI state
    public boolean historyPanelVisible = true;
    public int historyPanelHeight = 300;
    
    // Debug session state
    public boolean debugSessionActive = false;
    public String lastDebugSessionWorkingDir = "";
    public String lastDebugSessionArgs = ""; // Stores the last used additional tdlv arguments
    
    public WfDebuggerState() {
        // Default constructor
    }
    
    // Utility methods
    
    /**
     * Add a breakpoint for the given event ID
     */
    public void addBreakpoint(long eventId) {
        enabledBreakpoints.add(eventId);
    }
    
    /**
     * Remove a breakpoint for the given event ID
     */
    public void removeBreakpoint(long eventId) {
        enabledBreakpoints.remove(eventId);
    }
    
    /**
     * Check if a breakpoint is enabled for the given event ID
     */
    public boolean hasBreakpoint(long eventId) {
        return enabledBreakpoints.contains(eventId);
    }
    
    /**
     * Toggle breakpoint for the given event ID
     * @return true if breakpoint is now enabled, false if disabled
     */
    public boolean toggleBreakpoint(long eventId) {
        if (enabledBreakpoints.contains(eventId)) {
            enabledBreakpoints.remove(eventId);
            return false;
        } else {
            enabledBreakpoints.add(eventId);
            return true;
        }
    }
    
    /**
     * Clear all breakpoints
     */
    public void clearAllBreakpoints() {
        enabledBreakpoints.clear();
    }
    
    /**
     * Get count of enabled breakpoints
     */
    public int getBreakpointCount() {
        return enabledBreakpoints.size();
    }
    
    /**
     * Update loaded events and sync breakpoint states
     */
    public void setLoadedEvents(List<HistoryEvent> events) {
        this.loadedEvents = new ArrayList<>(events);
        
        // Update breakpoint states on the events
        for (HistoryEvent event : this.loadedEvents) {
            event.setBreakpointEnabled(enabledBreakpoints.contains(event.getEventId()));
        }
    }
    
    /**
     * Get loaded events with current breakpoint states
     */
    public List<HistoryEvent> getLoadedEvents() {
        // Ensure breakpoint states are up to date
        for (HistoryEvent event : loadedEvents) {
            event.setBreakpointEnabled(enabledBreakpoints.contains(event.getEventId()));
        }
        return new ArrayList<>(loadedEvents);
    }
    
    /**
     * Check if configuration is valid for starting a debug session
     */
    public boolean isValidConfiguration() {
        return debugDirectory != null && !debugDirectory.trim().isEmpty() &&
               tdlvBinaryPath != null && !tdlvBinaryPath.trim().isEmpty();
    }
    
    /**
     * Check if a history file is loaded
     */
    public boolean hasHistoryLoaded() {
        return !loadedEvents.isEmpty();
    }
    
    /**
     * Get a summary of the current state
     */
    public String getStateSummary() {
        StringBuilder summary = new StringBuilder();
        summary.append("Debug Directory: ").append(debugDirectory.isEmpty() ? "Not set" : debugDirectory).append("\n");
        summary.append("Loaded Events: ").append(loadedEvents.size()).append("\n");
        summary.append("Active Breakpoints: ").append(enabledBreakpoints.size()).append("\n");
        summary.append("Debug Session: ").append(debugSessionActive ? "Active" : "Inactive");
        if (debugSessionActive && !lastDebugSessionArgs.isEmpty()) {
            summary.append(" (").append(lastDebugSessionArgs).append(")");
        }
        return summary.toString();
    }
    
    @Override
    public String toString() {
        return "WfDebuggerState{" +
                "debugDirectory='" + debugDirectory + '\'' +
                ", tdlvBinaryPath='" + tdlvBinaryPath + '\'' +
                ", loadedEvents=" + loadedEvents.size() +
                ", enabledBreakpoints=" + enabledBreakpoints.size() +
                ", debugSessionActive=" + debugSessionActive +
                (debugSessionActive ? ", lastDebugSessionArgs='" + lastDebugSessionArgs + '\'' : "") +
                '}';
    }
} 