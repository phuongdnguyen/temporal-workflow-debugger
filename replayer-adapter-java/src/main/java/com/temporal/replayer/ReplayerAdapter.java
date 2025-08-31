package com.temporal.replayer;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.temporal.replayer.interceptors.RunnerWorkerInterceptor;
import io.temporal.common.WorkflowExecutionHistory;
import io.temporal.testing.WorkflowReplayer;
import io.temporal.testing.TestWorkflowEnvironment;
import io.temporal.testing.TestEnvironmentOptions;
import io.temporal.worker.WorkerOptions;
import io.temporal.worker.WorkerFactoryOptions;
import io.temporal.workflow.WorkflowInfo;
import org.apache.hc.client5.http.classic.methods.HttpGet;
import org.apache.hc.client5.http.classic.methods.HttpPost;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.client5.http.impl.classic.HttpClients;
import org.apache.hc.core5.http.ClassicHttpResponse;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.apache.hc.core5.http.ContentType;
import org.apache.hc.core5.http.io.entity.StringEntity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import io.temporal.internal.common.HistoryJsonUtils;

/**
 * Main adapter class for debugging Temporal workflows by replaying execution history.
 * Supports both standalone mode (using local history files) and IDE mode (with debugger UI integration).
 */
public class ReplayerAdapter {
    private static final Logger logger = LoggerFactory.getLogger(ReplayerAdapter.class);
    
    private static final String DEFAULT_DEBUGGER_URL = "http://localhost:54578";
    private static final String ENV_DEBUGGER_URL = "TEMPORAL_DEBUGGER_PLUGIN_URL";
    
    private static ReplayMode mode = ReplayMode.STANDALONE;
    private static final Set<Integer> breakpoints = ConcurrentHashMap.newKeySet();
    private static String debuggerAddr = "";
    private static int lastNotifiedStartEvent = -1;
    
    private static final ObjectMapper objectMapper = new ObjectMapper();
    private static final CloseableHttpClient httpClient = HttpClients.createDefault();
    
    /**
     * Sets the event IDs where execution should pause during replay.
     * This function is only used in Standalone mode.
     * 
     * @param eventIds list of event IDs to set as breakpoints
     */
    public static void setBreakpoints(List<Integer> eventIds) {
        breakpoints.clear();
        breakpoints.addAll(eventIds);
        logger.info("Set breakpoints: {}", eventIds);
    }
    
    /**
     * Configures the replay mode (Standalone or IDE).
     * 
     * @param replayMode the replay mode to set
     */
    public static void setReplayMode(ReplayMode replayMode) {
        mode = replayMode;
        logger.info("Set replay mode: {}", mode);
    }
    
    /**
     * Executes workflow replay with the specified options and workflow class.
     * The behavior depends on the configured ReplayMode:
     * - STANDALONE: replays using the history file specified in opts.getHistoryFilePath()
     * - IDE: replays by fetching history from the IDE debugger interface
     * 
     * @param opts     replay configuration options
     * @param workflow the workflow class to replay
     * @throws Exception if replay fails
     */
    public static void replay(ReplayOptions opts, Class<?> workflow) throws Exception {
        logger.info("Replaying in mode: {}", mode);
        
        switch (mode) {
            case STANDALONE:
                replayWithJsonFile(opts.getWorkerOptions(), workflow, opts.getHistoryFilePath());
                break;
            case IDE:
                WorkflowExecutionHistory hist = getHistoryFromIDE();
                replayWithHistory(opts.getWorkerOptions(), hist, workflow);
                break;
            default:
                throw new IllegalArgumentException("Unknown replay mode: " + mode);
        }
    }
    
    /**
     * Checks if the given event ID is a breakpoint.
     * 
     * @param eventId the event ID to check
     * @return true if the event ID is a breakpoint, false otherwise
     */
    public static boolean isBreakpoint(int eventId) {
        switch (mode) {
            case STANDALONE:
                logger.debug("Standalone checking breakpoints: {}, eventId: {}", breakpoints, eventId);
                boolean isHit = breakpoints.contains(eventId);
                if (isHit) {
                    logger.info("Hit breakpoint at eventId: {}", eventId);
                }
                return isHit;
                
            case IDE:
                if (debuggerAddr.isEmpty()) {
                    return false;
                }
                
                try {
                    HttpGet request = new HttpGet(debuggerAddr + "/breakpoints");
                    ClassicHttpResponse response = httpClient.execute(request);
                    
                    try {
                        String responseBody = EntityUtils.toString(response.getEntity());
                        JsonNode jsonNode = objectMapper.readTree(responseBody);
                        JsonNode breakpointsNode = jsonNode.get("breakpoints");
                        
                        if (breakpointsNode != null && breakpointsNode.isArray()) {
                            for (JsonNode breakpoint : breakpointsNode) {
                                if (breakpoint.asInt() == eventId) {
                                    return true;
                                }
                            }
                        }
                    } finally {
                        response.close();
                    }
                } catch (IOException | org.apache.hc.core5.http.ParseException e) {
                    logger.warn("Could not get breakpoints from IDE: {}", e.getMessage());
                }
                return false;
                
            default:
                return false;
        }
    }
    
    /**
     * Raises a sentinel breakpoint for debugging - called from interceptors.
     * 
     * @param caller the name of the calling operation
     * @param info   workflow info containing the current history length (can be null for activities)
     */
    public static void raiseSentinelBreakpoint(String caller, WorkflowInfo info) {
        int eventId = -1;
        
        if (info != null) {
            try {
                // Get current history length from WorkflowInfo
                eventId = (int) info.getHistoryLength();
            } catch (Exception e) {
                logger.debug("Could not get current history length: {}", e.getMessage());
                return;
            }
        }
        
        if (eventId <= lastNotifiedStartEvent) {
            return;
        }
        
        lastNotifiedStartEvent = eventId;
        logger.info("Runner notified at {} by {}, eventId: {}", System.currentTimeMillis(), caller, eventId);
        
        if (isBreakpoint(eventId)) {
            logger.info("Pause at event {}", eventId);
            
            if (mode == ReplayMode.IDE) {
                highlightCurrentEventInIDE(eventId);
            }
            
            // Java equivalent of runtime.Breakpoint() - trigger debugger breakpoint
            // This is a no-op in production but can be caught by debuggers
            logger.info("BREAKPOINT: Paused at event {}", eventId);
            
            // Optional: Add a way to pause execution for interactive debugging
            try {
                Thread.sleep(100); // Brief pause to allow debugger to catch
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
    }
    
    /**
     * Sends a POST request to highlight the current event being debugged in the IDE.
     * 
     * @param eventId the event ID to highlight
     */
    private static void highlightCurrentEventInIDE(int eventId) {
        if (debuggerAddr.isEmpty()) {
            logger.warn("debuggerAddr is empty, cannot send highlight request");
            return;
        }
        
        logger.info("Sending highlight request for event {} to {}/current-event", eventId, debuggerAddr);
        
        try {
            Map<String, Integer> payload = Collections.singletonMap("eventId", eventId);
            String jsonPayload = objectMapper.writeValueAsString(payload);
            logger.debug("Highlight payload: {}", jsonPayload);
            
            HttpPost request = new HttpPost(debuggerAddr + "/current-event");
            request.setEntity(new StringEntity(jsonPayload, ContentType.APPLICATION_JSON));
            
            ClassicHttpResponse response = httpClient.execute(request);
            try {
                String responseBody = EntityUtils.toString(response.getEntity());
                logger.info("Highlight response status: {}, body: {}", response.getCode(), responseBody);
                
                if (response.getCode() == 200) {
                    logger.info("✓ Successfully highlighted event {} in debugger UI", eventId);
                } else {
                    logger.warn("Highlight request failed with status: {}, response: {}", response.getCode(), responseBody);
                }
            } finally {
                response.close();
            }
        } catch (IOException | org.apache.hc.core5.http.ParseException e) {
            logger.warn("Failed to send highlight request: {}", e.getMessage());
        }
    }
    
    /**
     * Fetches workflow history from the IDE debugger interface.
     * 
     * @return the workflow history
     * @throws Exception if history cannot be fetched
     */
    private static WorkflowExecutionHistory getHistoryFromIDE() throws Exception {
        String addr = System.getenv(ENV_DEBUGGER_URL);
        if (addr == null || addr.isEmpty()) {
            addr = DEFAULT_DEBUGGER_URL;
        }
        debuggerAddr = addr;
        
        logger.info("Fetching history from IDE at: {}/history", debuggerAddr);
        
        HttpGet request = new HttpGet(debuggerAddr + "/history");
        ClassicHttpResponse response = httpClient.execute(request);
        
        try {
            if (response.getCode() != 200) {
                throw new RuntimeException("Could not get history from IDE: HTTP " + response.getCode());
            }
            
            byte[] bodyBytes = EntityUtils.toByteArray(response.getEntity());
            
            // Parse binary protobuf using Temporal SDK
            io.temporal.api.history.v1.History historyProto = io.temporal.api.history.v1.History.parseFrom(bodyBytes);
            logger.info("Successfully fetched history with {} events", historyProto.getEventsCount());
            
            // Convert to WorkflowExecutionHistory format expected by the replayer
            return new WorkflowExecutionHistory(historyProto);
        } finally {
            response.close();
        }
    }
    
    /**
     * Replays workflow with history data.
     * 
     * @param workerOptions worker configuration options
     * @param hist          the workflow history
     * @param workflow      the workflow class to replay
     * @throws Exception if replay fails
     */
    private static void replayWithHistory(WorkerOptions workerOptions, WorkflowExecutionHistory hist, Class<?> workflow) throws Exception {
        // 1. Configure interceptors at the factory level
        WorkerFactoryOptions factoryOptions = WorkerFactoryOptions.newBuilder()
            .setWorkerInterceptors(new RunnerWorkerInterceptor())
            .build();

        // 2. Create TestWorkflowEnvironment with interceptors
        TestEnvironmentOptions testEnvOptions = TestEnvironmentOptions.newBuilder()
            .setWorkerFactoryOptions(factoryOptions)
            .build();

        TestWorkflowEnvironment testEnv = TestWorkflowEnvironment.newInstance(testEnvOptions);

        try {
            // 3. Use WorkflowReplayer static methods with the configured environment
            WorkflowReplayer.replayWorkflowExecution(hist, testEnv, workflow);
        } finally {
            testEnv.close();
        }
        
        logger.info("Completed replay with history");
    }
    
    /**
     * Replays workflow with history from JSON file.
     * 
     * @param workerOptions  worker configuration options
     * @param workflow       the workflow class to replay
     * @param jsonFileName   path to the JSON history file
     * @throws Exception if file cannot be read or replay fails
     */
    private static void replayWithJsonFile(WorkerOptions workerOptions, Class<?> workflow, String jsonFileName) throws Exception {
        if (jsonFileName == null || jsonFileName.isEmpty()) {
            throw new IllegalArgumentException("History file path is required for STANDALONE mode");
        }
        
        logger.info("Reading history from file: {}", jsonFileName);
        
        // 1. Configure interceptors at the factory level
        WorkerFactoryOptions factoryOptions = WorkerFactoryOptions.newBuilder()
            .setWorkerInterceptors(new RunnerWorkerInterceptor())
            .build();

        // 2. Create TestWorkflowEnvironment with interceptors
        TestEnvironmentOptions testEnvOptions = TestEnvironmentOptions.newBuilder()
            .setWorkerFactoryOptions(factoryOptions)
            .build();

        TestWorkflowEnvironment testEnv = TestWorkflowEnvironment.newInstance(testEnvOptions);

        try {
            // 3. Read JSON file and convert from simplified format to protobuf format
            String jsonContent = new String(Files.readAllBytes(Paths.get(jsonFileName)));
            logger.info("Original JSON content length: {}", jsonContent.length());
            
            // Check if eventType field is present in the first event
            if (jsonContent.contains("\"eventType\"")) {
                logger.info("Original JSON contains eventType field");
                // Extract the first eventType value for debugging
                int eventTypeIndex = jsonContent.indexOf("\"eventType\"");
                int startQuote = jsonContent.indexOf("\"", eventTypeIndex + 12);
                int endQuote = jsonContent.indexOf("\"", startQuote + 1);
                if (startQuote > 0 && endQuote > startQuote) {
                    String firstEventType = jsonContent.substring(startQuote + 1, endQuote);
                    logger.info("First eventType in original JSON: {}", firstEventType);
                }
            } else {
                logger.warn("Original JSON does NOT contain eventType field!");
            }
            
            // Determine if we need to convert the JSON format
            String protoJson;
            if (jsonContent.contains("EVENT_TYPE_")) {
                // JSON is already in protobuf format, no conversion needed
                logger.info("JSON is already in protobuf format, skipping conversion");
                protoJson = jsonContent;
            } else {
                // Convert from simplified format (e.g., "WorkflowExecutionStarted") to protobuf format (e.g., "EVENT_TYPE_WORKFLOW_EXECUTION_STARTED")
                logger.info("Converting from simplified format to protobuf format");
                protoJson = HistoryJsonUtils.historyFormatJsonToProtoJson(jsonContent);
            }
            
            logger.info("Final JSON length: {}", protoJson.length());
            
            // Check if eventType field is present in the final JSON
            if (protoJson.contains("\"eventType\"")) {
                logger.info("Final JSON contains eventType field");
                // Extract the first eventType value for debugging
                int eventTypeIndex = protoJson.indexOf("\"eventType\"");
                int startQuote = protoJson.indexOf("\"", eventTypeIndex + 12);
                int endQuote = protoJson.indexOf("\"", startQuote + 1);
                if (startQuote > 0 && endQuote > startQuote) {
                    String firstEventType = protoJson.substring(startQuote + 1, endQuote);
                    logger.info("First eventType in final JSON: {}", firstEventType);
                }
            } else {
                logger.warn("Final JSON does NOT contain eventType field!");
            }
            
            // Parse the JSON using protobuf parser
            com.google.protobuf.util.JsonFormat.Parser parser = com.google.protobuf.util.JsonFormat.parser().ignoringUnknownFields();
            io.temporal.api.history.v1.History.Builder historyBuilder = io.temporal.api.history.v1.History.newBuilder();
            
            try {
                parser.merge(protoJson, historyBuilder);
            } catch (com.google.protobuf.InvalidProtocolBufferException e) {
                throw new RuntimeException("Failed to parse JSON to protobuf: " + e.getMessage(), e);
            }
            
            io.temporal.api.history.v1.History historyProto = historyBuilder.build();
            logger.info("Successfully parsed history with {} events", historyProto.getEventsCount());
            
            // Debug: Check the first event's eventType
            if (historyProto.getEventsCount() > 0) {
                io.temporal.api.history.v1.HistoryEvent firstEvent = historyProto.getEvents(0);
                logger.info("First event eventType: {}", firstEvent.getEventType());
                logger.info("First event has workflowExecutionStartedEventAttributes: {}", firstEvent.hasWorkflowExecutionStartedEventAttributes());
            }
            
            // Create WorkflowExecutionHistory from the parsed protobuf
            WorkflowExecutionHistory history = new WorkflowExecutionHistory(historyProto);
            
            // 4. Replay the workflow
            WorkflowReplayer.replayWorkflowExecution(history, testEnv, workflow);
        } finally {
            testEnv.close();
        }
        
        logger.info("Completed replay with JSON file");
    }
}
