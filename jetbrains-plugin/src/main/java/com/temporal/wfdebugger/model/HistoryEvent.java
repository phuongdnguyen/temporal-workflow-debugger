package com.temporal.wfdebugger.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Instant;
import java.util.Map;

/**
 * Represents a single event in a Temporal workflow history.
 * This class models the structure of events from temporal workflow history JSON files.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class HistoryEvent {
    
    @JsonProperty("eventId")
    private long eventId;
    
    @JsonProperty("eventTime")
    private String eventTime;
    
    @JsonProperty("eventType")
    private String eventType;
    
    @JsonProperty("version")
    private long version;
    
    @JsonProperty("taskId")
    private long taskId;
    
    @JsonProperty("attributes")
    private Map<String, Object> attributes;
    
    // Plugin-specific fields (not from JSON)
    private boolean breakpointEnabled = false;
    private String displayName;
    private String description;
    
    public HistoryEvent() {}
    
    public HistoryEvent(long eventId, String eventTime, String eventType) {
        this.eventId = eventId;
        this.eventTime = eventTime;
        this.eventType = eventType;
        this.displayName = generateDisplayName();
        this.description = generateDescription();
    }
    
    // Getters and setters
    public long getEventId() {
        return eventId;
    }
    
    public void setEventId(long eventId) {
        this.eventId = eventId;
    }
    
    public String getEventTime() {
        return eventTime;
    }
    
    public void setEventTime(String eventTime) {
        this.eventTime = eventTime;
    }
    
    public String getEventType() {
        return eventType;
    }
    
    public void setEventType(String eventType) {
        this.eventType = eventType;
        this.displayName = generateDisplayName();
        this.description = generateDescription();
    }
    
    public long getVersion() {
        return version;
    }
    
    public void setVersion(long version) {
        this.version = version;
    }
    
    public long getTaskId() {
        return taskId;
    }
    
    public void setTaskId(long taskId) {
        this.taskId = taskId;
    }
    
    public Map<String, Object> getAttributes() {
        return attributes;
    }
    
    public void setAttributes(Map<String, Object> attributes) {
        this.attributes = attributes;
    }
    
    public boolean isBreakpointEnabled() {
        return breakpointEnabled;
    }
    
    public void setBreakpointEnabled(boolean breakpointEnabled) {
        this.breakpointEnabled = breakpointEnabled;
    }
    
    public String getDisplayName() {
        if (displayName == null) {
            displayName = generateDisplayName();
        }
        return displayName;
    }
    
    public String getDescription() {
        if (description == null) {
            description = generateDescription();
        }
        return description;
    }
    
    /**
     * Generate a human-readable display name for the event
     */
    private String generateDisplayName() {
        if (eventType == null) {
            return "ID: " + eventId + " - Event";
        }
        
        // Use the human-readable event type with ID first
        return "ID: " + eventId + " - " + getHumanReadableEventType();
    }
    
    /**
     * Generate a description based on event type and attributes
     */
    private String generateDescription() {
        StringBuilder desc = new StringBuilder();
        desc.append("Type: ").append(eventType != null ? eventType : "Unknown");
        
        if (eventTime != null) {
            desc.append(" | Time: ").append(eventTime);
        }
        
        if (taskId > 0) {
            desc.append(" | Task: ").append(taskId);
        }
        
        // Add some common attributes if present
        if (attributes != null) {
            if (attributes.containsKey("activityType")) {
                desc.append(" | Activity: ").append(attributes.get("activityType"));
            }
            if (attributes.containsKey("workflowType")) {
                desc.append(" | Workflow: ").append(attributes.get("workflowType"));
            }
            if (attributes.containsKey("reason")) {
                desc.append(" | Reason: ").append(attributes.get("reason"));
            }
        }
        
        return desc.toString();
    }
    
    /**
     * Get the timestamp as an Instant if possible
     */
    public Instant getEventTimeAsInstant() {
        if (eventTime == null || eventTime.isEmpty()) {
            return null;
        }
        
        try {
            return Instant.parse(eventTime);
        } catch (Exception e) {
            // If parsing fails, return null
            return null;
        }
    }
    
    /**
     * Check if this event is a workflow execution event
     */
    public boolean isWorkflowExecutionEvent() {
        return eventType != null && eventType.toLowerCase().contains("workflowexecution");
    }
    
    /**
     * Check if this event is an activity event
     */
    public boolean isActivityEvent() {
        return eventType != null && eventType.toLowerCase().contains("activity");
    }
    
    /**
     * Check if this event is a timer event
     */
    public boolean isTimerEvent() {
        return eventType != null && eventType.toLowerCase().contains("timer");
    }
    
    /**
     * Convert technical event type to human-readable name
     */
    public String getHumanReadableEventType() {
        if (eventType == null) {
            return "Unknown Event";
        }
        
        // Handle common Temporal event types
        switch (eventType) {
            // Workflow events
            case "EVENT_TYPE_WORKFLOW_EXECUTION_STARTED":
                return "Workflow Started";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED":
                return "Workflow Completed";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_FAILED":
                return "Workflow Failed";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT":
                return "Workflow Timed Out";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED":
                return "Workflow Terminated";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED":
                return "Workflow Canceled";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW":
                return "Workflow Continued As New";
                
            // Workflow task events
            case "EVENT_TYPE_WORKFLOW_TASK_SCHEDULED":
                return "Workflow Task Scheduled";
            case "EVENT_TYPE_WORKFLOW_TASK_STARTED":
                return "Workflow Task Started";
            case "EVENT_TYPE_WORKFLOW_TASK_COMPLETED":
                return "Workflow Task Completed";
            case "EVENT_TYPE_WORKFLOW_TASK_FAILED":
                return "Workflow Task Failed";
            case "EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT":
                return "Workflow Task Timed Out";
                
            // Activity events
            case "EVENT_TYPE_ACTIVITY_TASK_SCHEDULED":
                return "Activity Scheduled";
            case "EVENT_TYPE_ACTIVITY_TASK_STARTED":
                return "Activity Started";
            case "EVENT_TYPE_ACTIVITY_TASK_COMPLETED":
                return "Activity Completed";
            case "EVENT_TYPE_ACTIVITY_TASK_FAILED":
                return "Activity Failed";
            case "EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT":
                return "Activity Timed Out";
            case "EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED":
                return "Activity Cancel Requested";
            case "EVENT_TYPE_ACTIVITY_TASK_CANCELED":
                return "Activity Canceled";
                
            // Timer events
            case "EVENT_TYPE_TIMER_STARTED":
                return "Timer Started";
            case "EVENT_TYPE_TIMER_FIRED":
                return "Timer Fired";
            case "EVENT_TYPE_TIMER_CANCELED":
                return "Timer Canceled";
                
            // Signal events
            case "EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED":
                return "Signal Received";
            case "EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED":
                return "External Signal Sent";
                
            // Child workflow events
            case "EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED":
                return "Child Workflow Started";
            case "EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED":
                return "Child Workflow Running";
            case "EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED":
                return "Child Workflow Completed";
            case "EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED":
                return "Child Workflow Failed";
                
            // Marker events
            case "EVENT_TYPE_MARKER_RECORDED":
                return "Marker Recorded";
                
            // Request cancel events
            case "EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED":
                return "Cancel Request Sent";
            case "EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_CANCEL_REQUESTED":
                return "Cancel Request Received";
                
            // Version marker events
            case "EVENT_TYPE_VERSION_MARKER":
                return "Version Marker";
                
            // Query events
            case "EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ADMITTED":
                return "Update Admitted";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED":
                return "Update Accepted";
            case "EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED":
                return "Update Completed";
                
            default:
                // Fallback: try to make a reasonable human-readable version
                return formatGenericEventType(eventType);
        }
    }
    
    /**
     * Format unknown event types to be more readable
     */
    private String formatGenericEventType(String eventType) {
        // Remove EVENT_TYPE_ prefix
        String formatted = eventType.replaceFirst("^EVENT_TYPE_", "");
        
        // Convert UPPER_CASE to Title Case
        String[] words = formatted.toLowerCase().split("_");
        StringBuilder result = new StringBuilder();
        
        for (String word : words) {
            if (result.length() > 0) {
                result.append(" ");
            }
            if (!word.isEmpty()) {
                result.append(Character.toUpperCase(word.charAt(0)));
                if (word.length() > 1) {
                    result.append(word.substring(1));
                }
            }
        }
        
        return result.toString();
    }
    
    @Override
    public String toString() {
        return "HistoryEvent{" +
                "eventId=" + eventId +
                ", eventType='" + eventType + '\'' +
                ", eventTime='" + eventTime + '\'' +
                ", breakpointEnabled=" + breakpointEnabled +
                '}';
    }
    
    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        
        HistoryEvent that = (HistoryEvent) o;
        return eventId == that.eventId;
    }
    
    @Override
    public int hashCode() {
        return Long.hashCode(eventId);
    }
} 