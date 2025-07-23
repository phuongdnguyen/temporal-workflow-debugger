package com.temporal.wfdebugger.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.intellij.openapi.components.PersistentStateComponent;
import com.intellij.openapi.components.Service;
import com.intellij.openapi.components.State;
import com.intellij.openapi.components.Storage;
import com.intellij.openapi.diagnostic.Logger;
import com.intellij.util.xmlb.XmlSerializerUtil;
import com.temporal.wfdebugger.model.HistoryEvent;
import com.temporal.wfdebugger.model.WfDebuggerState;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;

import java.io.File;
import java.io.IOException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.HashMap;

/**
 * Core service for the Temporal Workflow Debugger plugin.
 * Handles history file loading, breakpoint management, and debugging coordination.
 */
@Service
@State(
    name = "WfDebuggerService",
    storages = @Storage("wf-debugger.xml")
)
public final class WfDebuggerService implements PersistentStateComponent<WfDebuggerState> {
    
    private static final Logger LOG = Logger.getInstance(WfDebuggerService.class);
    private final ObjectMapper objectMapper;
    private WfDebuggerState state;
    
    // Persistent storage for history files
    private static final String PERSISTENT_HISTORY_FILENAME = "current-history.json";
    private static final String PERSISTENT_BREAKPOINTS_FILENAME = "current-breakpoints.json";
    
    public WfDebuggerService() {
        this.objectMapper = new ObjectMapper();
        this.state = new WfDebuggerState();
        LOG.info("WfDebuggerService initialized");
        
        // Start the history server immediately when the service is created
        startHistoryServer();
        
        // Load persisted history if available and ensure UI/server consistency
        loadPersistedHistoryOnStartup();
        
        // Load persisted breakpoint state if available
        loadPersistedBreakpointState();
        
        // Load persisted history file hash
        loadPersistedHistoryFileHash();
    }
    
    @Override
    public @Nullable WfDebuggerState getState() {
        return state;
    }
    
    @Override
    public void loadState(@NotNull WfDebuggerState state) {
        XmlSerializerUtil.copyBean(state, this.state);
        LOG.info("Loaded WfDebuggerService state: " + this.state.toString());
    }
    
    /**
     * Load a workflow history JSON file and parse the events
     * @param filePath Path to the history JSON file
     * @return Number of events loaded
     * @throws IOException If file cannot be read or parsed
     */
    public int loadHistoryFile(String filePath) throws IOException {
        LOG.info("Loading history file: " + filePath);
        
        File historyFile = new File(filePath);
        if (!historyFile.exists()) {
            throw new IOException("History file does not exist: " + filePath);
        }
        
        // Read raw bytes - this maintains compatibility with both JSON and protobuf formats
        // The Go adapter (replayer.go) tries protobuf first, then falls back to JSON
        // VSCode extension always converts to protobuf, but we support JSON input for flexibility
        byte[] rawBytes = Files.readAllBytes(historyFile.toPath());

        // Calculate hash of the new file to determine if breakpoints should be cleared
        String newFileHash = calculateFileHash(rawBytes);
        boolean shouldClearBreakpoints = shouldClearBreakpointsForNewFile(newFileHash);
        
        if (shouldClearBreakpoints) {
            int clearedBreakpoints = state.getBreakpointCount();
            state.clearAllBreakpoints();
            LOG.info("Cleared " + clearedBreakpoints + " breakpoints due to different history file (hash changed from " 
                + currentHistoryFileHash + " to " + newFileHash + ")");
        } else {
            LOG.info("Same history file detected (hash: " + newFileHash + "), keeping existing breakpoints");
        }
        
        // Update stored hash
        currentHistoryFileHash = newFileHash;

        // Parse the JSON file for UI display (this doesn't affect the /history endpoint)
        JsonNode rootNode = objectMapper.readTree(rawBytes);
        List<HistoryEvent> events = parseHistoryEvents(rootNode);
        
        // Update state with loaded data
        state.setLoadedEvents(events);
        
        // Sync breakpoint enabled flags with service state
        for (HistoryEvent event : events) {
            event.setBreakpointEnabled(state.hasBreakpoint(event.getEventId()));
        }
        
        // Update the history server with raw bytes
        // Note: /history endpoint now returns raw bytes directly (VSCode compatible)
        // instead of JSON-wrapped response
        updateHistoryServerData(rawBytes);
        
        // Persist the history file for future sessions
        persistHistoryFile(rawBytes);
        
        // Persist the new hash and breakpoint state
        persistHistoryFileHash(newFileHash);
        persistBreakpointState();

        LOG.info("Successfully loaded " + events.size() + " events from history file, updated history server, and persisted data");
        return events.size();
    }
    
    /**
     * Persist the current history data to the file system for future sessions.
     * This ensures the history server always has the same data across restarts.
     */
    private void persistHistoryFile(byte[] historyBytes) {
        try {
            Path persistentPath = Paths.get(PERSISTENT_HISTORY_FILENAME);
            Files.write(persistentPath, historyBytes);
            LOG.info("Persisted history data to: " + persistentPath.toAbsolutePath());
        } catch (IOException e) {
            LOG.error("Failed to persist history file: " + e.getMessage());
        }
    }
    
    /**
     * Persist the current breakpoint state to the file system for future sessions.
     * This ensures breakpoint state is maintained across plugin restarts.
     */
    private void persistBreakpointState() {
        try {
            // Convert breakpoint set to JSON
            Map<String, Object> breakpointData = new HashMap<>();
            breakpointData.put("breakpoints", new ArrayList<>(state.enabledBreakpoints));
            breakpointData.put("timestamp", System.currentTimeMillis());
            breakpointData.put("count", state.enabledBreakpoints.size());
            
            String jsonData = objectMapper.writeValueAsString(breakpointData);
            byte[] breakpointBytes = jsonData.getBytes(java.nio.charset.StandardCharsets.UTF_8);
            
            Path persistentPath = Paths.get(PERSISTENT_BREAKPOINTS_FILENAME);
            Files.write(persistentPath, breakpointBytes);
            LOG.info("Persisted breakpoint state to: " + persistentPath.toAbsolutePath() + " (" + state.enabledBreakpoints.size() + " breakpoints)");
            LOG.info("Persisted breakpoints: " + state.enabledBreakpoints);
        } catch (IOException e) {
            LOG.error("Failed to persist breakpoint state: " + e.getMessage());
        }
    }
    
    /**
     * Load persisted breakpoint state from the file system if available.
     */
    private void loadPersistedBreakpointState() {
        try {
            Path breakpointPath = Paths.get(PERSISTENT_BREAKPOINTS_FILENAME);
            if (Files.exists(breakpointPath)) {
                LOG.info("Loading persisted breakpoint state from: " + breakpointPath.toAbsolutePath());
                byte[] breakpointBytes = Files.readAllBytes(breakpointPath);
                String jsonData = new String(breakpointBytes, java.nio.charset.StandardCharsets.UTF_8);
                
                JsonNode rootNode = objectMapper.readTree(jsonData);
                if (rootNode.has("breakpoints")) {
                    JsonNode breakpointsArray = rootNode.get("breakpoints");
                    if (breakpointsArray.isArray()) {
                        state.enabledBreakpoints.clear();
                        for (JsonNode breakpointNode : breakpointsArray) {
                            if (breakpointNode.isNumber()) {
                                state.enabledBreakpoints.add(breakpointNode.asLong());
                            }
                        }
                        LOG.info("Successfully loaded " + state.enabledBreakpoints.size() + " persisted breakpoints: " + state.enabledBreakpoints);
                    }
                }
            } else {
                LOG.info("No persisted breakpoint state found at: " + breakpointPath.toAbsolutePath());
            }
        } catch (IOException | RuntimeException e) {
            LOG.error("Failed to load persisted breakpoint state: " + e.getMessage());
            // Clear potentially corrupted breakpoint state
            state.enabledBreakpoints.clear();
            
            // Try to clean up corrupted persistent file
            try {
                clearPersistedBreakpointState();
                LOG.info("Cleared corrupted persistent breakpoint file");
            } catch (Exception cleanupException) {
                LOG.error("Failed to clean up corrupted persistent breakpoint file: " + cleanupException.getMessage());
            }
        }
    }
    
    /**
     * Clear persisted breakpoint state from the file system
     */
    private void clearPersistedBreakpointState() {
        try {
            Path persistentPath = Paths.get(PERSISTENT_BREAKPOINTS_FILENAME);
            if (Files.exists(persistentPath)) {
                Files.delete(persistentPath);
                LOG.info("Cleared persisted breakpoint state file: " + persistentPath.toAbsolutePath());
            }
        } catch (IOException e) {
            LOG.error("Failed to clear persisted breakpoint state file: " + e.getMessage());
        }
    }
    
    /**
     * Parse history events from the JSON structure
     */
    private List<HistoryEvent> parseHistoryEvents(JsonNode rootNode) throws IOException {
        List<HistoryEvent> events = new ArrayList<>();
        
        // Handle different possible JSON structures
        JsonNode eventsNode = null;
        
        if (rootNode.has("events")) {
            eventsNode = rootNode.get("events");
        } else if (rootNode.has("history")) {
            eventsNode = rootNode.get("history");
        } else if (rootNode.isArray()) {
            eventsNode = rootNode;
        } else {
            throw new IOException("Invalid history file format - expected 'events' or 'history' array");
        }
        
        if (!eventsNode.isArray()) {
            throw new IOException("Events node is not an array");
        }
        
        // Parse each event
        for (JsonNode eventNode : eventsNode) {
            try {
                HistoryEvent event = parseHistoryEvent(eventNode);
                if (event != null) {
                    events.add(event);
                }
            } catch (Exception e) {
                LOG.warn("Failed to parse event: " + e.getMessage());
                // Continue parsing other events
            }
        }
        
        return events;
    }
    
    /**
     * Parse a single history event from JSON node
     */
    private HistoryEvent parseHistoryEvent(JsonNode eventNode) {
        HistoryEvent event = new HistoryEvent();
        
        // Extract basic event information with flexible field names
        if (eventNode.has("eventId")) {
            event.setEventId(eventNode.get("eventId").asLong());
        } else if (eventNode.has("id")) {
            event.setEventId(eventNode.get("id").asLong());
        }
        
        if (eventNode.has("eventType")) {
            event.setEventType(eventNode.get("eventType").asText());
        } else if (eventNode.has("type")) {
            event.setEventType(eventNode.get("type").asText());
        }
        
        if (eventNode.has("eventTime")) {
            event.setEventTime(eventNode.get("eventTime").asText());
        } else if (eventNode.has("timestamp")) {
            event.setEventTime(eventNode.get("timestamp").asText());
        } else if (eventNode.has("time")) {
            event.setEventTime(eventNode.get("time").asText());
        }
        
        if (eventNode.has("version")) {
            event.setVersion(eventNode.get("version").asLong());
        }
        
        if (eventNode.has("taskId")) {
            event.setTaskId(eventNode.get("taskId").asLong());
        }
        
        // Parse attributes/details
        JsonNode attributesNode = null;
        if (eventNode.has("attributes")) {
            attributesNode = eventNode.get("attributes");
        } else if (eventNode.has("details")) {
            attributesNode = eventNode.get("details");
        } else if (eventNode.has("data")) {
            attributesNode = eventNode.get("data");
        }
        
        if (attributesNode != null) {
            try {
                Map<String, Object> attributes = objectMapper.convertValue(
                    attributesNode, 
                    new TypeReference<Map<String, Object>>() {}
                );
                event.setAttributes(attributes);
            } catch (Exception e) {
                LOG.warn("Failed to parse attributes for event " + event.getEventId() + ": " + e.getMessage());
            }
        }
        
        return event;
    }
    
    /**
     * Toggle breakpoint for the given event ID (UI action).
     */
    public boolean toggleBreakpoint(long eventId) {
        boolean enabled = state.toggleBreakpoint(eventId);
        LOG.info("Breakpoint " + (enabled ? "enabled" : "disabled") + " for event " + eventId);
        
        // Explicitly trigger state persistence
        triggerStatePersistence();
        
        return enabled;
    }
    
    /**
     * Add a breakpoint for the given event ID
     */
    public void addBreakpoint(long eventId) {
        state.addBreakpoint(eventId);
        LOG.info("Breakpoint added for event " + eventId);
        
        // Explicitly trigger state persistence
        triggerStatePersistence();
    }
    
    /**
     * Remove a breakpoint for the given event ID
     */
    public void removeBreakpoint(long eventId) {
        state.removeBreakpoint(eventId);
        LOG.info("Breakpoint removed for event " + eventId);
        
        // Explicitly trigger state persistence
        triggerStatePersistence();
    }
    
    /**
     * Clear all breakpoints
     */
    public void clearAllBreakpoints() {
        int count = state.getBreakpointCount();
        state.clearAllBreakpoints();
        LOG.info("Cleared " + count + " breakpoints");
        
        // Update breakpoint states on loaded events
        for (HistoryEvent event : state.loadedEvents) {
            event.setBreakpointEnabled(false);
        }
        
        // Explicitly trigger state persistence
        triggerStatePersistence();
    }
    
    /**
     * Clear loaded history and all breakpoints
     */
    public void clearHistory() {
        int eventCount = state.loadedEvents.size();
        int breakpointCount = state.getBreakpointCount();
        
        state.loadedEvents.clear();
        state.enabledBreakpoints.clear();
        
        // Clear server data
        currentHistoryBytes = null;
        currentHistoryFileHash = null;
        
        // Clear persistent data
        clearPersistedHistory();
        clearPersistedBreakpointState();
        clearPersistedHistoryFileHash();
        
        LOG.info("Cleared history (" + eventCount + " events), breakpoints (" + breakpointCount + "), hash, server data, and persistent storage");
    }
    
    /**
     * Clear persisted history file from the file system
     */
    private void clearPersistedHistory() {
        try {
            Path persistentPath = Paths.get(PERSISTENT_HISTORY_FILENAME);
            if (Files.exists(persistentPath)) {
                Files.delete(persistentPath);
                LOG.info("Cleared persisted history file: " + persistentPath.toAbsolutePath());
            }
        } catch (IOException e) {
            LOG.error("Failed to clear persisted history file: " + e.getMessage());
        }
    }
    
    /**
     * Check if the history server has actual data available
     */
    public boolean hasHistoryServerData() {
        return currentHistoryBytes != null && currentHistoryBytes.length > 0;
    }
    
    /**
     * Check if UI state and server state are consistent
     */
    public boolean isHistoryStateConsistent() {
        boolean hasUIEvents = !state.loadedEvents.isEmpty();
        boolean hasServerData = hasHistoryServerData();
        return hasUIEvents == hasServerData;
    }
    
    /**
     * Get a summary of current history state for debugging
     */
    public String getHistoryStateSummary() {
        return String.format(
            "UI Events: %d, Server Data: %s, Consistent: %s, Breakpoints: %d",
            state.loadedEvents.size(),
            hasHistoryServerData() ? (currentHistoryBytes.length + " bytes") : "none",
            isHistoryStateConsistent() ? "✓" : "✗",
            state.getBreakpointCount()
        );
    }
    
    /**
     * Get summary of current breakpoint persistence state for debugging
     */
    public String getBreakpointPersistenceStatus() {
        try {
            Path breakpointPath = Paths.get(PERSISTENT_BREAKPOINTS_FILENAME);
            boolean fileExists = Files.exists(breakpointPath);
            long fileSize = fileExists ? Files.size(breakpointPath) : 0;
            
            return String.format(
                "Breakpoints in memory: %d (%s), Persisted file: %s (%d bytes)",
                state.getBreakpointCount(),
                state.enabledBreakpoints.toString(),
                fileExists ? "exists" : "missing",
                fileSize
            );
        } catch (IOException e) {
            return "Error checking breakpoint persistence: " + e.getMessage();
        }
    }
    
    /**
     * Get current history file hash status for debugging
     */
    public String getHistoryFileHashStatus() {
        try {
            Path hashPath = Paths.get(PERSISTENT_HASH_FILENAME);
            boolean fileExists = Files.exists(hashPath);
            
            return String.format(
                "Current hash in memory: %s, Persisted hash file: %s",
                currentHistoryFileHash != null ? currentHistoryFileHash.substring(0, Math.min(8, currentHistoryFileHash.length())) + "..." : "null",
                fileExists ? "exists" : "missing"
            );
        } catch (Exception e) {
            return "Error checking hash status: " + e.getMessage();
        }
    }
    
    /**
     * Validate state consistency and attempt recovery if inconsistent.
     * This method can be called by UI components to ensure data integrity.
     * 
     * @return true if state is consistent or was successfully recovered, false otherwise
     */
    public boolean validateAndRecoverState() {
        if (isHistoryStateConsistent()) {
            return true;
        }
        
        LOG.warn("History state inconsistency detected: " + getHistoryStateSummary());
        
        // Attempt recovery strategies
        boolean hasUIEvents = !state.loadedEvents.isEmpty();
        boolean hasServerData = hasHistoryServerData();
        
        if (hasUIEvents && !hasServerData) {
            // UI has events but server doesn't - try to reload from persistent storage
            LOG.info("Attempting to recover server data from persistent storage...");
            try {
                Path persistentPath = Paths.get(PERSISTENT_HISTORY_FILENAME);
                if (Files.exists(persistentPath)) {
                    byte[] persistedBytes = Files.readAllBytes(persistentPath);
                    updateHistoryServerData(persistedBytes);
                    LOG.info("Successfully recovered server data from persistent storage");
                    return isHistoryStateConsistent();
                }
            } catch (IOException e) {
                LOG.error("Failed to recover from persistent storage: " + e.getMessage());
            }
            
            // Clear UI state to match server state
            LOG.info("Clearing UI state to match empty server state");
            state.loadedEvents.clear();
            state.enabledBreakpoints.clear();
            return true;
            
        } else if (!hasUIEvents && hasServerData) {
            // Server has data but UI doesn't - clear server data
            LOG.info("Clearing server data to match empty UI state");
            currentHistoryBytes = null;
            clearPersistedHistory();
            return true;
        }
        
        // If both are inconsistent in other ways, clear everything
        LOG.warn("Unable to recover state, clearing all data");
        clearHistory();
        return true;
    }
    
    /**
     * Check if configuration is valid for debugging
     */
    public boolean isConfigurationValid() {
        return state.isValidConfiguration();
    }
    
    /**
     * Get summary of current state for debugging
     */
    public String getStateSummary() {
        return state.getStateSummary() + " | " + getHistoryFileHashStatus();
    }
    
    /**
     * Check if a debug session can be started
     */
    public boolean canStartDebugSession() {
        return isConfigurationValid() && !state.debugDirectory.isEmpty();
    }
    
    /**
     * Check if the tdlv binary is available
     */
    public boolean isTdlvAvailable() {
        try {
            ProcessBuilder pb = new ProcessBuilder(state.tdlvBinaryPath, "--help");
            Process process = pb.start();
            int exitCode = process.waitFor();
            return exitCode == 0;
        } catch (Exception e) {
            LOG.warn("Failed to check tdlv availability: " + e.getMessage());
            return false;
        }
    }
    
    /**
     * Get the configured tdlv binary path
     */
    public String getTdlvBinaryPath() {
        return state.tdlvBinaryPath;
    }

    public void setTdlvBinaryPath(String path) {
        if (path != null) {
            state.tdlvBinaryPath = path;
        }
    }
    
    /**
     * Get the configured debug directory
     */
    public String getDebugDirectory() {
        return state.debugDirectory;
    }

    public void setDebugDirectory(String dir) {
        if (dir != null) {
            state.debugDirectory = dir;
        }
    }
    
    /**
     * Get list of events with breakpoints enabled
     */
    public List<HistoryEvent> getEventsWithBreakpoints() {
        List<HistoryEvent> eventsWithBreakpoints = new ArrayList<>();
        for (HistoryEvent event : state.loadedEvents) {
            if (state.hasBreakpoint(event.getEventId())) {
                eventsWithBreakpoints.add(event);
            }
        }
        return eventsWithBreakpoints;
    }

    // --------------------------------------------------------------------
    // History HTTP server (GET /history) - VSCode Extension Compatible
    // --------------------------------------------------------------------
    // This server mimics the behavior of the VSCode debugger extension:
    // - GET /history: Returns raw bytes (200) or error JSON (404) 
    // - GET /breakpoints: Returns JSON array of breakpoint IDs
    // - Content-Type: application/octet-stream for history, application/json for others
    // - The Go adapter (replayer.go) handles both protobuf and JSON formats automatically
    //
    // Using fixed port 54578 (instead of dynamic assignment) for:
    // - Predictable port for debugging and configuration
    // - Compatibility with Go adapter default port
    // - Easier firewall/network configuration

    private static final int FIXED_HISTORY_PORT = 54578; // Fixed port for history server (matches Go adapter default)
    private HttpServer historyServer;
    private volatile byte[] currentHistoryBytes;
    private volatile int historyPort = FIXED_HISTORY_PORT;
    private volatile Long currentHighlightedEventId = null; // Currently highlighted event in debugger
    private volatile String currentHistoryFileHash = null; // Hash of currently loaded history file

    /**
     * Start the history server without any data (called during plugin initialization)
     */
    private synchronized void startHistoryServer() {
        try {
            historyServer = HttpServer.create(new InetSocketAddress("127.0.0.1", FIXED_HISTORY_PORT), 0);
            
            // History endpoint
            historyServer.createContext("/history", new HttpHandler() {
                @Override
                public void handle(HttpExchange exchange) throws IOException {
                    LOG.info("Handling /history request from " + exchange.getRemoteAddress());
                    if (!"GET".equals(exchange.getRequestMethod())) {
                        LOG.warn("/history: Method not allowed: " + exchange.getRequestMethod());
                        exchange.sendResponseHeaders(405, -1);
                        return;
                    }
                    
                    if (currentHistoryBytes != null) {
                        LOG.info("/history: Sending loaded history data (" + currentHistoryBytes.length + " bytes)");
                        exchange.getResponseHeaders().add("Content-Type", "application/octet-stream");
                        exchange.sendResponseHeaders(200, currentHistoryBytes.length);
                        exchange.getResponseBody().write(currentHistoryBytes);
                        exchange.close();
                    } else {
                        // Return 404 with error JSON when no history is loaded (VSCode compatible)
                        LOG.info("/history: No history loaded, returning 404");
                        String errorResponse = "{\"error\":\"No current history available\"}";
                        byte[] errorBytes = errorResponse.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                        exchange.getResponseHeaders().add("Content-Type", "application/json");
                        exchange.sendResponseHeaders(404, errorBytes.length);
                        exchange.getResponseBody().write(errorBytes);
                        exchange.close();
                    }
                }
            });

            // Breakpoints endpoint
            historyServer.createContext("/breakpoints", new HttpHandler() {
                @Override
                public void handle(HttpExchange exchange) throws IOException {
                    LOG.info("Handling /breakpoints request from " + exchange.getRemoteAddress());
                    try {
                        if (!"GET".equals(exchange.getRequestMethod())) {
                            LOG.warn("/breakpoints: Method not allowed: " + exchange.getRequestMethod());
                            exchange.sendResponseHeaders(405, -1);
                            return;
                        }
                        
                        java.util.Set<Long> bps = state.enabledBreakpoints;
                        LOG.info("/breakpoints: Found " + bps.size() + " breakpoints: " + bps);
                        
                        StringBuilder sb = new StringBuilder();
                        sb.append("{\"breakpoints\":[");
                        boolean first = true;
                        for (Long id : bps) {
                            if (!first) sb.append(',');
                            sb.append(id);
                            first = false;
                        }
                        sb.append("]}");
                        
                        String jsonResponse = sb.toString();
                        LOG.info("/breakpoints: Sending response: " + jsonResponse);
                        
                        byte[] resp = jsonResponse.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                        exchange.getResponseHeaders().add("Content-Type", "application/json");
                        exchange.sendResponseHeaders(200, resp.length);
                        exchange.getResponseBody().write(resp);
                        exchange.close();
                        LOG.info("/breakpoints: Response sent successfully");
                    } catch (Exception e) {
                        LOG.error("/breakpoints: Error handling request", e);
                        try {
                            String errorMsg = "{\"error\":\"" + e.getMessage() + "\"}";
                            byte[] errorResp = errorMsg.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                            exchange.getResponseHeaders().add("Content-Type", "application/json");
                            exchange.sendResponseHeaders(500, errorResp.length);
                            exchange.getResponseBody().write(errorResp);
                            exchange.close();
                        } catch (Exception e2) {
                            LOG.error("Failed to send error response", e2);
                        }
                    }
                }
            });
            
            // Current event endpoint for highlighting
            historyServer.createContext("/current-event", new HttpHandler() {
                @Override
                public void handle(HttpExchange exchange) throws IOException {
                    LOG.info("Handling /current-event request from " + exchange.getRemoteAddress());
                    try {
                        if ("POST".equals(exchange.getRequestMethod())) {
                            // Read the event ID from request body
                            String requestBody = new String(exchange.getRequestBody().readAllBytes(), java.nio.charset.StandardCharsets.UTF_8);
                            LOG.info("/current-event: Received POST data: " + requestBody);
                            
                            try {
                                JsonNode payload = objectMapper.readTree(requestBody);
                                                                if (payload.has("eventId")) {
                                    long eventId = payload.get("eventId").asLong();
                                    Long previousEventId = currentHighlightedEventId;
                                    currentHighlightedEventId = eventId;
                                    LOG.info("/current-event: Set highlighted event from " + previousEventId + " to: " + eventId);
                                    
                                    // Trigger UI update
                                    WfDebuggerService.this.triggerHighlightUpdate(eventId);
                                    
                                    String response = "{\"status\":\"ok\",\"highlightedEventId\":" + eventId + "}";
                                    byte[] resp = response.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                                    exchange.getResponseHeaders().add("Content-Type", "application/json");
                                    exchange.sendResponseHeaders(200, resp.length);
                                    exchange.getResponseBody().write(resp);
                                    exchange.close();
                                } else {
                                    String errorResponse = "{\"error\":\"Missing eventId in payload\"}";
                                    byte[] errorBytes = errorResponse.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                                    exchange.getResponseHeaders().add("Content-Type", "application/json");
                                    exchange.sendResponseHeaders(400, errorBytes.length);
                                    exchange.getResponseBody().write(errorBytes);
                                    exchange.close();
                                }
                            } catch (Exception e) {
                                LOG.error("/current-event: Error parsing JSON: " + e.getMessage());
                                String errorResponse = "{\"error\":\"Invalid JSON payload\"}";
                                byte[] errorBytes = errorResponse.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                                exchange.getResponseHeaders().add("Content-Type", "application/json");
                                exchange.sendResponseHeaders(400, errorBytes.length);
                                exchange.getResponseBody().write(errorBytes);
                                exchange.close();
                            }
                            
                        } else if ("DELETE".equals(exchange.getRequestMethod())) {
                            // Clear highlight (when debugging continues)
                            Long previousEventId = currentHighlightedEventId;
                            LOG.info("/current-event: Clearing highlighted event (was: " + previousEventId + ")");
                            currentHighlightedEventId = null;
                            WfDebuggerService.this.triggerHighlightUpdate(null);
                            
                            String response = "{\"status\":\"ok\",\"highlightedEventId\":null}";
                            byte[] resp = response.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                            exchange.getResponseHeaders().add("Content-Type", "application/json");
                            exchange.sendResponseHeaders(200, resp.length);
                            exchange.getResponseBody().write(resp);
                            exchange.close();
                            
                        } else {
                            LOG.warn("/current-event: Method not allowed: " + exchange.getRequestMethod());
                            exchange.sendResponseHeaders(405, -1);
                        }
                    } catch (Exception e) {
                        LOG.error("/current-event: Error handling request", e);
                        try {
                            String errorMsg = "{\"error\":\"" + e.getMessage() + "\"}";
                            byte[] errorResp = errorMsg.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                            exchange.getResponseHeaders().add("Content-Type", "application/json");
                            exchange.sendResponseHeaders(500, errorResp.length);
                            exchange.getResponseBody().write(errorResp);
                            exchange.close();
                        } catch (Exception e2) {
                            LOG.error("Failed to send error response", e2);
                        }
                    }
                }
            });
            
            // Test endpoint to verify server is working
            historyServer.createContext("/test", new HttpHandler() {
                @Override
                public void handle(HttpExchange exchange) throws IOException {
                    LOG.info("Handling /test request from " + exchange.getRemoteAddress());
                    String response = "{\"status\":\"ok\",\"server\":\"wf-debugger-history\",\"port\":" + FIXED_HISTORY_PORT + "}";
                    byte[] resp = response.getBytes(java.nio.charset.StandardCharsets.UTF_8);
                    exchange.getResponseHeaders().add("Content-Type", "application/json");
                    exchange.sendResponseHeaders(200, resp.length);
                    exchange.getResponseBody().write(resp);
                    exchange.close();
                    LOG.info("/test: Response sent successfully");
                }
            });
            
            historyServer.setExecutor(null); // default executor
            historyServer.start();
            // Using fixed port instead of dynamic assignment
            LOG.info("History server started at http://127.0.0.1:" + FIXED_HISTORY_PORT);
            LOG.info("Available endpoints:");
            LOG.info("  - GET http://127.0.0.1:" + FIXED_HISTORY_PORT + "/history");
            LOG.info("  - GET http://127.0.0.1:" + FIXED_HISTORY_PORT + "/breakpoints");
            LOG.info("  - POST/DELETE http://127.0.0.1:" + FIXED_HISTORY_PORT + "/current-event");
            LOG.info("  - GET http://127.0.0.1:" + FIXED_HISTORY_PORT + "/test");
            
            // Wait a bit for server to fully initialize
            try {
                Thread.sleep(100);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
            
            // Verify server is actually listening
            try {
                java.net.URL testUrl = new java.net.URL("http://127.0.0.1:" + FIXED_HISTORY_PORT + "/test");
                java.net.HttpURLConnection testConn = (java.net.HttpURLConnection) testUrl.openConnection();
                testConn.setConnectTimeout(5000);
                testConn.setReadTimeout(5000);
                int responseCode = testConn.getResponseCode();
                LOG.info("Server verification: /test endpoint returned status " + responseCode);
                testConn.disconnect();
                
                if (responseCode != 200) {
                    LOG.error("Server verification failed: expected 200, got " + responseCode);
                }
            } catch (Exception e) {
                LOG.error("Server verification failed", e);
            }
            
        } catch (IOException e) {
            LOG.error("Failed to start history server", e);
        }
    }

    /**
     * Update the history server with new data when a history file is loaded
     */
    private synchronized void updateHistoryServerData(byte[] data) {
        this.currentHistoryBytes = data;
        LOG.info("Updated history server with new data (" + data.length + " bytes)");
        
        if (historyServer == null) {
            LOG.warn("History server not running, starting it now...");
            startHistoryServer();
        }
    }

    /**
     * Load persisted history from the file system if available and ensure UI/server consistency.
     * This method is called during service startup to load the last loaded history.
     */
    private void loadPersistedHistoryOnStartup() {
        try {
            Path historyFilePath = Paths.get(PERSISTENT_HISTORY_FILENAME);
            if (Files.exists(historyFilePath)) {
                LOG.info("Loading persisted history from: " + historyFilePath.toAbsolutePath());
                byte[] persistedHistoryBytes = Files.readAllBytes(historyFilePath);
                
                // Validate the persisted file is valid JSON
                JsonNode rootNode = objectMapper.readTree(persistedHistoryBytes);
                List<HistoryEvent> persistedEvents = parseHistoryEvents(rootNode);
                
                // Update state with persisted history
                state.setLoadedEvents(persistedEvents);
                
                // Update the history server with persisted data
                updateHistoryServerData(persistedHistoryBytes);
                
                // Validate consistency after loading
                if (!isHistoryStateConsistent()) {
                    LOG.error("History state inconsistent after loading persisted data: " + getHistoryStateSummary());
                }
                
                LOG.info("Successfully loaded " + persistedEvents.size() + " persisted events from " + PERSISTENT_HISTORY_FILENAME);
            } else {
                LOG.info("No persisted history file found at: " + historyFilePath.toAbsolutePath());
            }
        } catch (IOException | RuntimeException e) {
            LOG.error("Failed to load persisted history: " + e.getMessage(), e);
            // Clear potentially corrupted state
            state.loadedEvents.clear();
            currentHistoryBytes = null;
            
            // Try to clean up corrupted persistent file
            try {
                clearPersistedHistory();
                LOG.info("Cleared corrupted persistent history file");
            } catch (Exception cleanupException) {
                LOG.error("Failed to clean up corrupted persistent file: " + cleanupException.getMessage());
            }
        }
    }

    /**
     * Trigger state persistence using file-based mechanism.
     * This ensures breakpoint changes are saved to disk immediately and available to the history server.
     */
    private void triggerStatePersistence() {
        try {
            // Log the current state to verify it's updated
            LOG.info("Triggering breakpoint state persistence. Current state: " + state.enabledBreakpoints);
            LOG.info("Total breakpoints: " + state.getBreakpointCount());
            
            // Persist breakpoint state to file immediately
            persistBreakpointState();
            
            // The IntelliJ PersistentStateComponent will also automatically save the state
            
        } catch (Exception e) {
            LOG.error("Error during state persistence: " + e.getMessage());
        }
    }


    public int getHistoryPort() {
        return historyPort;
    }
    
    /**
     * Check if the history server is running
     */
    public boolean isHistoryServerRunning() {
        return historyServer != null && historyServer.getAddress() != null;
    }
    
    /**
     * Get the current server status for debugging
     */
    public String getServerStatus() {
        if (historyServer == null) {
            return "Server not started";
        }
        
        try {
            return String.format("Server running on http://127.0.0.1:%d (status: %s)", 
                FIXED_HISTORY_PORT, 
                historyServer.getAddress() != null ? "listening" : "not listening");
        } catch (Exception e) {
            return "Server error: " + e.getMessage();
        }
    }
    
    /**
     * Force restart the history server (for debugging)
     */
    public void restartHistoryServer() {
        LOG.info("Force restarting history server...");
        if (historyServer != null) {
            try {
                historyServer.stop(0);
                LOG.info("Stopped existing server");
            } catch (Exception e) {
                LOG.warn("Error stopping existing server", e);
            }
            historyServer = null;
        }
        
        if (currentHistoryBytes != null) {
            updateHistoryServerData(currentHistoryBytes);
        } else {
            LOG.warn("Cannot restart server - no history data available");
        }
    }
    
    // --------------------------------------------------------------------
    // Event Highlighting Support
    // --------------------------------------------------------------------
    
    /**
     * Get the currently highlighted event ID (the event being debugged)
     */
    public Long getCurrentHighlightedEventId() {
        return currentHighlightedEventId;
    }
    
    /**
     * Set the currently highlighted event ID
     */
    public void setCurrentHighlightedEventId(Long eventId) {
        Long previousEventId = currentHighlightedEventId;
        currentHighlightedEventId = eventId;
        LOG.info("Highlighted event changed from " + previousEventId + " to " + eventId);
        triggerHighlightUpdate(eventId);
    }
    
    /**
     * Clear the currently highlighted event
     */
    public void clearHighlightedEvent() {
        setCurrentHighlightedEventId(null);
    }
    
    /**
     * Check if a specific event is currently highlighted
     */
    public boolean isEventHighlighted(long eventId) {
        return currentHighlightedEventId != null && currentHighlightedEventId == eventId;
    }
    
    /**
     * Trigger UI update when highlight changes
     * This method will be called to notify UI components that the highlighted event has changed
     */
    private void triggerHighlightUpdate(Long eventId) {
        // Use ApplicationManager to run UI updates on the proper thread
        com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater(() -> {
            try {
                LOG.info("Triggering UI highlight update for event: " + eventId);
                
                // Find and refresh all open history panels
                com.intellij.openapi.project.Project[] openProjects = com.intellij.openapi.project.ProjectManager.getInstance().getOpenProjects();
                for (com.intellij.openapi.project.Project project : openProjects) {
                    try {
                        // Find the Temporal Workflow Debugger tool window
                        com.intellij.openapi.wm.ToolWindowManager toolWindowManager = com.intellij.openapi.wm.ToolWindowManager.getInstance(project);
                        com.intellij.openapi.wm.ToolWindow toolWindow = toolWindowManager.getToolWindow("Temporal Workflow Debugger");
                        
                        if (toolWindow != null && toolWindow.isVisible()) {
                            // Get the content and refresh it
                            com.intellij.ui.content.ContentManager contentManager = toolWindow.getContentManager();
                            com.intellij.ui.content.Content[] contents = contentManager.getContents();
                            
                            for (com.intellij.ui.content.Content content : contents) {
                                javax.swing.JComponent component = content.getComponent();
                                if (component != null) {
                                    // Force repaint of all components
                                    component.repaint();
                                    
                                    // Also try to find and refresh specific history panels
                                    refreshHistoryPanelsInComponent(component);
                                }
                            }
                            
                            LOG.info("Refreshed Temporal Workflow Debugger tool window for highlight update");
                        }
                    } catch (Exception e) {
                        LOG.warn("Error refreshing tool window for project " + project.getName(), e);
                    }
                }
                
            } catch (Exception e) {
                LOG.error("Error during highlight UI update", e);
            }
        });
    }
    
    /**
     * Recursively refresh history panels within a component tree
     */
    private void refreshHistoryPanelsInComponent(java.awt.Component component) {
        if (component instanceof javax.swing.JList) {
            // Repaint JList components (our event lists)
            component.repaint();
        }
        
        if (component instanceof java.awt.Container) {
            java.awt.Container container = (java.awt.Container) component;
            for (java.awt.Component child : container.getComponents()) {
                refreshHistoryPanelsInComponent(child);
            }
        }
    }
    
    // --------------------------------------------------------------------
    // File Hash Management for Breakpoint Clearing
    // --------------------------------------------------------------------
    
    private static final String PERSISTENT_HASH_FILENAME = "current-history-hash.txt";
    
    /**
     * Calculate SHA-256 hash of file content
     */
    private String calculateFileHash(byte[] fileContent) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashBytes = digest.digest(fileContent);
            
            // Convert to hex string
            StringBuilder hexString = new StringBuilder();
            for (byte b : hashBytes) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) {
                    hexString.append('0');
                }
                hexString.append(hex);
            }
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            LOG.error("SHA-256 algorithm not available", e);
            // Fallback to simple content length + timestamp
            return "fallback_" + fileContent.length + "_" + System.currentTimeMillis();
        }
    }
    
    /**
     * Determine if breakpoints should be cleared for a new file
     */
    private boolean shouldClearBreakpointsForNewFile(String newFileHash) {
        if (currentHistoryFileHash == null) {
            // First time loading a file, don't clear breakpoints
            return false;
        }
        
        if (newFileHash == null) {
            // Unable to calculate hash, be conservative and don't clear
            return false;
        }
        
        // Clear breakpoints if the hash is different
        return !newFileHash.equals(currentHistoryFileHash);
    }
    
    /**
     * Persist the history file hash to disk
     */
    private void persistHistoryFileHash(String hash) {
        try {
            if (hash != null) {
                Path hashPath = Paths.get(PERSISTENT_HASH_FILENAME);
                Files.write(hashPath, hash.getBytes(java.nio.charset.StandardCharsets.UTF_8));
                LOG.info("Persisted history file hash: " + hash);
            }
        } catch (IOException e) {
            LOG.error("Failed to persist history file hash: " + e.getMessage());
        }
    }
    
    /**
     * Load persisted history file hash from disk
     */
    private void loadPersistedHistoryFileHash() {
        try {
            Path hashPath = Paths.get(PERSISTENT_HASH_FILENAME);
            if (Files.exists(hashPath)) {
                byte[] hashBytes = Files.readAllBytes(hashPath);
                currentHistoryFileHash = new String(hashBytes, java.nio.charset.StandardCharsets.UTF_8);
                LOG.info("Loaded persisted history file hash: " + currentHistoryFileHash);
            } else {
                LOG.info("No persisted history file hash found");
            }
        } catch (IOException e) {
            LOG.error("Failed to load persisted history file hash: " + e.getMessage());
            currentHistoryFileHash = null;
        }
    }
    
    /**
     * Clear persisted history file hash
     */
    private void clearPersistedHistoryFileHash() {
        try {
            Path hashPath = Paths.get(PERSISTENT_HASH_FILENAME);
            if (Files.exists(hashPath)) {
                Files.delete(hashPath);
                LOG.info("Cleared persisted history file hash");
            }
        } catch (IOException e) {
            LOG.error("Failed to clear persisted history file hash: " + e.getMessage());
        }
    }
} 