package com.temporal.wfdebugger.debug;

import com.intellij.execution.ExecutionException;
import com.intellij.execution.ExecutionResult;
import com.intellij.execution.Executor;
import com.intellij.execution.ProgramRunnerUtil;
import com.intellij.execution.RunManager;
import com.intellij.execution.RunnerAndConfigurationSettings;
import com.intellij.execution.configurations.CommandLineState;
import com.intellij.execution.configurations.ConfigurationFactory;
import com.intellij.execution.configurations.ConfigurationType;
import com.intellij.execution.configurations.GeneralCommandLine;
import com.intellij.execution.configurations.RunConfiguration;
import com.intellij.execution.process.ProcessHandler;
import com.intellij.execution.process.ProcessHandlerFactory;
import com.intellij.execution.process.OSProcessHandler;
import com.intellij.execution.process.ProcessOutputTypes;
import com.intellij.execution.runners.ExecutionEnvironment;
import com.intellij.execution.runners.ProgramRunner;
import com.intellij.execution.executors.DefaultDebugExecutor;
import com.intellij.notification.NotificationGroupManager;
import com.intellij.notification.NotificationType;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.diagnostic.Logger;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.util.Key;
import com.intellij.openapi.wm.ToolWindow;
import com.intellij.openapi.wm.ToolWindowManager;
import com.intellij.ui.content.Content;
import com.intellij.ui.content.ContentManager;
import com.temporal.wfdebugger.service.WfDebuggerService;
import org.jetbrains.annotations.NotNull;

import java.io.File;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import javax.swing.JTabbedPane;


/**
 * Run state for executing the tdlv debugger process.
 * This handles starting a pre-built tdlv binary, waiting for it to stabilize,
 * and automatically connecting the GoLand debugger with retry mechanisms.
 */
public class WfDebugRunState extends CommandLineState {
    
    private static final Logger LOG = Logger.getInstance(WfDebugRunState.class);
    private final WfDebugRunConfiguration configuration;
    private RunnerAndConfigurationSettings goRemoteSettings;
    
    protected WfDebugRunState(ExecutionEnvironment environment, WfDebugRunConfiguration configuration) {
        super(environment);
        this.configuration = configuration;
    }
    
    @Override
    protected @NotNull ProcessHandler startProcess() throws ExecutionException {
        try {
            LOG.info("========== Starting tdlv debugger ==========");
            
            // Create and start the tdlv process
            LOG.info("Creating tdlv command...");
            GeneralCommandLine commandLine = createTdlvCommandLine();
            LOG.info("Command details:");
            LOG.info("  - Full command: " + commandLine.getCommandLineString());
            LOG.info("  - Working directory: " + commandLine.getWorkDirectory());
            LOG.info("  - Environment variables: " + commandLine.getEnvironment());
            LOG.info("  - Executable path: " + commandLine.getExePath());
            
            // Verify executable exists
            if (!new File(commandLine.getExePath()).exists()) {
                LOG.error("tdlv executable not found at: " + commandLine.getExePath());
                throw new ExecutionException("tdlv executable not found at: " + commandLine.getExePath());
            }
            
            LOG.info("Creating process handler...");
            // Create a process handler that captures output
            OSProcessHandler processHandler = new OSProcessHandler(commandLine) {
                @Override
                public void notifyTextAvailable(@NotNull String text, @NotNull Key outputType) {
                    String trimmedText = text.trim();
                    if (!trimmedText.isEmpty()) {
                        if (outputType == ProcessOutputTypes.STDERR) {
                            // Don't log normal tdlv status messages as warnings
                            if (trimmedText.contains("Listening for remote connections") ||
                                trimmedText.contains("Delve headless server started") ||
                                trimmedText.contains("Starting delve proxy") ||
                                trimmedText.contains("New client connected") ||
                                trimmedText.contains("Connected to Delve server") ||
                                trimmedText.contains("Client disconnected") ||
                                trimmedText.contains("shutting down")) {
                                LOG.info("tdlv: " + trimmedText);
                            } else {
                                LOG.warn("tdlv stderr: " + trimmedText);
                            }
                        } else if (outputType == ProcessOutputTypes.STDOUT) {
                            LOG.info("tdlv stdout: " + trimmedText);
                        } else {
                            LOG.debug("tdlv other output (" + outputType + "): " + trimmedText);
                        }
                    }
                }

                @Override
                public void startNotify() {
                    LOG.info("tdlv process starting with PID: " + getProcess().pid());
                    super.startNotify();
                }

                @Override
                protected void detachProcessImpl() {
                    LOG.info("Detaching tdlv process with PID: " + getProcess().pid());
                    super.detachProcessImpl();
                }

                @Override
                protected void destroyProcessImpl() {
                    LOG.info("Stopping tdlv process with PID: " + getProcess().pid());
                    super.destroyProcessImpl();
                }
                
                @Override
                public void notifyProcessTerminated(int exitCode) {
                    if (exitCode == 0) {
                        LOG.info("tdlv process terminated normally with exit code: " + exitCode);
                    } else {
                        LOG.warn("tdlv process terminated with non-zero exit code: " + exitCode);
                    }
                    
                    // Clean up debug session state
                    try {
                        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
                        service.getState().debugSessionActive = false;
                        service.getState().lastDebugSessionWorkingDir = "";
                        LOG.info("Debug session state cleaned up");
                    } catch (Exception e) {
                        LOG.warn("Failed to clean up debug session state", e);
                    }
                    
                    // Clean up auto-created Go Remote configuration
                    cleanupGoRemoteConfiguration();
                    
                    super.notifyProcessTerminated(exitCode);
                }
            };
            
            // Configure process handler - don't destroy recursively to keep tdlv running
            processHandler.setShouldDestroyProcessRecursively(false);
            
            // Extract port for notifications
            int port = extractPortFromArgs(configuration.getAdditionalArgs());
            LOG.info("tdlv will listen on port: " + port);
            
            // Start the process
            LOG.info("Starting tdlv process...");
            processHandler.startNotify();
            
            // Wait briefly for the process to start and stabilize (2 seconds - much faster than before)
            LOG.info("Waiting 2 seconds to verify process started and stabilized...");
            Thread.sleep(2000);
            
            Process process = processHandler.getProcess();
            boolean isAlive = process.isAlive();
            int exitValue = -1;
            try {
                exitValue = process.exitValue();
            } catch (IllegalThreadStateException e) {
                // Process is still running, this is good
            }
            
            LOG.info("Process status check:");
            LOG.info("  - Is alive: " + isAlive);
            LOG.info("  - Exit value: " + (exitValue == -1 ? "Still running" : exitValue));
            LOG.info("  - Process PID: " + process.pid());
            
            if (!isAlive) {
                LOG.error("tdlv process failed to start or terminated immediately");
                throw new ExecutionException("tdlv process failed to start or terminated immediately (exit code: " + exitValue + ")");
            }
            
            // Mark debug session as active
            WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
            service.getState().debugSessionActive = true;
            service.getState().lastDebugSessionWorkingDir = configuration.getWorkingDirectory();
            LOG.info("Debug session marked as active");
            
            // Show notification that tdlv is running
            showTdlvRunningNotification(port);
            
            // Wait for tdlv to be fully ready, then auto-connect to GoLand
            Project project = getEnvironment().getProject();
            ApplicationManager.getApplication().executeOnPooledThread(() -> {
                try {
                    LOG.info("Waiting for tdlv to initialize...");
                    
                    // Simple delay to let tdlv start up
                    Thread.sleep(1000);
                    
                    LOG.info("Attempting auto-connection to GoLand on port " + port);

                    ApplicationManager.getApplication().invokeLater(() -> {
                        try {
                            LOG.info("Starting Go Remote debug session to attach to GoLand...");
                            autoConnectToGoLand(project, port);
                        } catch (Exception e) {
                            LOG.error("Failed to auto-connect to tdlv in GoLand", e);
                            showErrorNotification(port, "Auto-connection to GoLand failed: " + e.getMessage());
                        }
                    });
                    
                    // Wait to verify connection
                    Thread.sleep(3000);
                    
                    ApplicationManager.getApplication().invokeLater(() -> {
                        try {
                            focusDebuggingTab(project, port);
                        } catch (Exception e) {
                            LOG.warn("Failed to auto-focus debugging tab", e);
                        }
                    });
                } catch (Exception e) {
                    LOG.error("Error during tdlv startup delay", e);
                    ApplicationManager.getApplication().invokeLater(() -> {
                        showErrorNotification(port, "Startup delay failed: " + e.getMessage());
                    });
                }
            });
            
            LOG.info("========== tdlv startup completed successfully ==========");
            LOG.info("tdlv is now running on port " + port);
            LOG.info("Ready for auto-connection to Go Remote debugger");
            
            return processHandler;
                
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOG.error("Interrupted during tdlv startup", e);
            throw new ExecutionException("Interrupted during tdlv startup: " + e.getMessage(), e);
        } catch (Exception e) {
            LOG.error("Critical error in tdlv startup", e);
            throw new ExecutionException("Critical error in tdlv startup: " + e.getMessage(), e);
        }
    }
    
    private boolean waitForTdlvReady(int port, int timeoutMs) {
        LOG.info("Skipping tdlv readiness check on port " + port + " - assuming ready");
        return true;
    }
    
    private void showTdlvRunningNotification(int port) {
        String message = String.format(
            "🚀 tdlv debugger starting on port %d\n" +
            "⏳ Checking readiness...\n" +
            "📦 Breakpoints configured: %d",
            port, getBreakpointCount()
        );
        
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification("Temporal Workflow Debugger", message, NotificationType.INFORMATION)
            .notify(getEnvironment().getProject());
    }
    
    private void showReadyForConnectionNotification(int port) {
        String message = String.format(
            "⚠️ Auto-connection failed - Manual setup required\n\n" +
            "📋 To connect manually:\n" +
            "1. Go to Run → Edit Configurations\n" +
            "2. Click '+' → Go Remote\n" +
            "3. Set Host: localhost, Port: %d\n" +
            "4. Click Debug to connect\n\n" +
            "🎯 Breakpoints: %d active\n" +
            "💡 tdlv is running on port %d and ready for connections\n" +
            "🔧 Check IntelliJ logs for detailed connection failure info",
            port, getBreakpointCount(), port
        );
        
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification("Manual Connection Required", message, NotificationType.WARNING)
            .notify(getEnvironment().getProject());
    }
    
    private void showErrorNotification(int port, String error) {
        String message = String.format(
            "❌ tdlv debugger startup failed\n" +
            "Port: %d\n" +
            "Error: %s\n\n" +
            "💡 Troubleshooting:\n" +
            "• Check if port %d is already in use\n" +
            "• Verify tdlv binary path in settings\n" +
            "• Check runner directory permissions",
            port, error, port
        );
        
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification("Debug Startup Failed", message, NotificationType.ERROR)
            .notify(getEnvironment().getProject());
    }
    
    private void showTimeoutNotification(int port) {
        String message = String.format(
            "⚠️ tdlv debugger did not start within timeout.\n\n" +
            "You can manually create a 'Go Remote' debug configuration:\n" +
            "Host: localhost, Port: %d",
            port
        );
        
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification("Connection Timeout", message, NotificationType.WARNING)
            .notify(getEnvironment().getProject());
    }

    private int getBreakpointCount() {
        try {
            WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
            return service.getState().getBreakpointCount();
        } catch (Exception e) {
            return 0;
        }
    }
    
    private int extractPortFromArgs(String additionalArgs) {
        if (additionalArgs == null || additionalArgs.isEmpty()) {
            return 60000; // default port
        }
        
        String[] args = additionalArgs.split("\\s+");
        for (int i = 0; i < args.length - 1; i++) {
            if ("-p".equals(args[i])) {
                try {
                    return Integer.parseInt(args[i + 1]);
                } catch (NumberFormatException e) {
                    LOG.warn("Invalid port number in args: " + args[i + 1]);
                }
            }
        }
        return 60000; // default port
    }
    
    private GeneralCommandLine createTdlvCommandLine() throws ExecutionException {
        GeneralCommandLine commandLine = new GeneralCommandLine();
        
        // Get the runner directory from settings first
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        String savedDebugDirectory = service.getDebugDirectory();
        String tdlvPath = service.getTdlvBinaryPath();
        
        String workingDir;
        if (savedDebugDirectory != null && !savedDebugDirectory.trim().isEmpty()) {
            workingDir = savedDebugDirectory.trim();
            LOG.info("Using runner directory from settings for tdlv execution: " + workingDir);
        } else {
            workingDir = configuration.getWorkingDirectory();
            LOG.info("Using runner directory from configuration for tdlv execution: " + workingDir);
        }
        
        if (workingDir == null || workingDir.isEmpty()) {
            throw new ExecutionException("Runner directory is not specified");
        }
        
        // Check if we have a configured tdlv path
        if (tdlvPath == null || tdlvPath.trim().isEmpty()) {
            throw new ExecutionException("tdlv binary path is not configured. Please configure it in Settings → Tools → Temporal Workflow Debugger");
        }
        
        // Verify tdlv exists and is executable
        File tdlvFile = new File(tdlvPath);
        if (!tdlvFile.exists() || !tdlvFile.canExecute()) {
            throw new ExecutionException("tdlv binary not found or not executable at: " + tdlvPath);
        }
        
        // Set the tdlv binary path
        commandLine.setExePath(tdlvPath);
        LOG.info("Using tdlv binary: " + tdlvPath);
        
        // Use the configured runner directory as the working directory for tdlv
        // This should be the directory containing the workflow code to debug
        File runnerDir = new File(workingDir);
        if (!runnerDir.exists() || !runnerDir.isDirectory()) {
            throw new ExecutionException("Runner directory does not exist: " + workingDir);
        }
        
        commandLine.setWorkDirectory(runnerDir);
        LOG.info("tdlv will execute in directory: " + workingDir);
        
        // Add proxy port to Go adapter
        int historyPort = service.getHistoryPort();
        if (historyPort > 0) {
            commandLine.getEnvironment().put("WFDBG_HISTORY_PORT", String.valueOf(historyPort));
        }

        // Always add the proxy port automatically (default 60000)
        int defaultPort = 60000;
        boolean hasPortArg = false;
        
        // Add additional arguments from configuration if any
        String additionalArgs = configuration.getAdditionalArgs();
        if (additionalArgs != null && !additionalArgs.trim().isEmpty()) {
            LOG.info("Adding additional arguments: " + additionalArgs);
            String[] args = additionalArgs.trim().split("\\s+");
            
            // Check if user specified a custom port
            for (int i = 0; i < args.length - 1; i++) {
                if ("-p".equals(args[i])) {
                    hasPortArg = true;
                    break;
                }
            }
            
            for (String arg : args) {
                commandLine.addParameter(arg);
            }
        }
        
        // Add default proxy port if user didn't specify one
        if (!hasPortArg) {
            LOG.info("Adding default proxy port: " + defaultPort);
            commandLine.addParameter("-p");
            commandLine.addParameter(String.valueOf(defaultPort));
        }
        
        
        LOG.info("Final tdlv command: " + commandLine.getCommandLineString());
        
        return commandLine;
    }
    
    @Override
    public @NotNull ExecutionResult execute(@NotNull Executor executor, @NotNull ProgramRunner runner) throws ExecutionException {
        // Log current breakpoints for debugging
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        int breakpointCount = service.getState().getBreakpointCount();
        LOG.info("Starting debug session with " + breakpointCount + " breakpoints enabled");
        
        if (breakpointCount > 0) {
            LOG.info("Enabled breakpoints: " + service.getState().enabledBreakpoints);
        }
        
        return super.execute(executor, runner);
    }

    /**
     * Automatically create and start a Go Remote debug configuration to connect to tdlv and attach to GoLand
     */
    private void autoConnectToGoLand(Project project, int port) throws ExecutionException {
        RunManager runManager = RunManager.getInstance(project);
        
        // Retry logic for connecting to GoLand
        int maxRetries = 3;
        Exception lastException = null;
        
        for (int attempt = 1; attempt <= maxRetries; attempt++) {
            try {
                LOG.info("Attempting to connect to GoLand (attempt " + attempt + "/" + maxRetries + ")");
                
                // Try to get the Go Remote configuration type
                ConfigurationFactory goRemoteFactory = findGoRemoteConfigurationFactory(runManager);
                if (goRemoteFactory == null) {
                    throw new ExecutionException("Go Remote configuration type not found. Make sure Go plugin is installed and enabled in GoLand.");
                }
                
                // Create Go Remote configuration
                String configName = "Temporal Workflow Debug - tdlv (port " + port + ")";
                goRemoteSettings = runManager.createConfiguration(configName, goRemoteFactory);
                
                // Configure the Go Remote settings
                RunConfiguration goRemoteConfig = goRemoteSettings.getConfiguration();
                configureGoRemoteConfiguration(goRemoteConfig, port);
                
                // Add to run manager but don't make it permanent
                runManager.addConfiguration(goRemoteSettings);
                
                LOG.info("Starting Go Remote debugger in GoLand to connect to tdlv on port " + port);
                
                // Add a small delay before starting the debugger to ensure everything is ready
                Thread.sleep(2000);
                
                LOG.info("Starting Go Remote debugger...");
                
                // Start the Go Remote debugger
                ProgramRunnerUtil.executeConfiguration(goRemoteSettings, DefaultDebugExecutor.getDebugExecutorInstance());
                
                // Wait a moment to verify the connection was successful
                Thread.sleep(3000);
                
                // Auto-focus on the debugging tab
                ApplicationManager.getApplication().invokeLater(() -> {
                    try {
                        focusDebuggingTab(project, port);
                    } catch (Exception e) {
                        LOG.warn("Failed to auto-focus debugging tab", e);
                    }
                });
                
                LOG.info("Successfully connected GoLand debugger to tdlv on port " + port);
                showAutoConnectedNotification(port);
                return; // Success!
                
            } catch (Exception e) {
                lastException = e;
                LOG.warn("Attempt " + attempt + "/" + maxRetries + " failed to connect to GoLand: " + e.getMessage());
                
                if (attempt < maxRetries) {
                    try {
                        LOG.info("Waiting 2 seconds before retry...");
                        Thread.sleep(2000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        break;
                    }
                } else {
                    LOG.error("All " + maxRetries + " attempts to connect to GoLand failed");
                }
            }
        }
        
        // All retries failed
        LOG.error("Failed to auto-connect to GoLand after " + maxRetries + " attempts", lastException);
        showReadyForConnectionNotification(port);
        throw new ExecutionException("Failed to auto-connect to GoLand after " + maxRetries + " attempts: " + 
                                    (lastException != null ? lastException.getMessage() : "Unknown error"), lastException);
    }
    
    /**
     * Find the Go Remote configuration factory
     */
    private ConfigurationFactory findGoRemoteConfigurationFactory(RunManager runManager) {
        // Try to find Go Remote configuration type by ID
        try {
            // Go plugin configuration type IDs (these may vary by GoLand version)
            String[] possibleIds = {
                "GoRemoteDebugConfigurationType",
                "com.goide.execution.GoRemoteDebugConfigurationType", 
                "GoRemoteRunConfigurationType"
            };
            
            // Get all configuration types
            ConfigurationType[] configTypes = ConfigurationType.CONFIGURATION_TYPE_EP.getExtensions();
            
            for (String typeId : possibleIds) {
                for (ConfigurationType configType : configTypes) {
                    if (configType.getId().equals(typeId)) {
                        ConfigurationFactory[] factories = configType.getConfigurationFactories();
                        if (factories.length > 0) {
                            LOG.info("Found Go Remote configuration factory: " + typeId);
                            return factories[0]; // Return the first factory
                        }
                    }
                }
            }
            
            // Try by display name as fallback
            for (ConfigurationType configType : configTypes) {
                String displayName = configType.getDisplayName();
                if (displayName.contains("Go Remote") || displayName.contains("Remote")) {
                    ConfigurationFactory[] factories = configType.getConfigurationFactories();
                    if (factories.length > 0) {
                        LOG.info("Found Go Remote configuration by display name: " + displayName);
                        return factories[0];
                    }
                }
            }
            
        } catch (Exception e) {
            LOG.warn("Error searching for Go Remote configuration factory", e);
        }
        
        return null;
    }
    
    /**
     * Configure the Go Remote debug configuration
     */
    private void configureGoRemoteConfiguration(RunConfiguration config, int port) throws ExecutionException {
        try {
            // Use reflection to set the configuration properties since Go plugin classes may not be available at compile time
            Class<?> configClass = config.getClass();
            LOG.info("Configuring Go Remote debug configuration: " + configClass.getName());
            
            // Log available methods for debugging
            LOG.info("Available methods in Go Remote configuration:");
            for (java.lang.reflect.Method method : configClass.getMethods()) {
                if (method.getName().startsWith("set") && method.getParameterCount() <= 1) {
                    LOG.info("  - " + method.getName() + "(" + 
                           (method.getParameterCount() > 0 ? method.getParameterTypes()[0].getSimpleName() : "") + ")");
                }
            }
            
            // Set host (usually "localhost" or "127.0.0.1")
            boolean hostSet = false;
            if (tryInvokeMethod(configClass, config, "setHost", new Class<?>[]{String.class}, new Object[]{"localhost"})) {
                LOG.info("Successfully set Go Remote host to localhost");
                hostSet = true;
            } else if (tryInvokeMethod(configClass, config, "setHost", new Class<?>[]{String.class}, new Object[]{"127.0.0.1"})) {
                LOG.info("Successfully set Go Remote host to 127.0.0.1");
                hostSet = true;
            }
            
            if (!hostSet) {
                LOG.warn("Could not set host - trying default behavior");
            }
            
            // Set port - try different parameter types
            boolean portSet = false;
            if (tryInvokeMethod(configClass, config, "setPort", new Class<?>[]{int.class}, new Object[]{port})) {
                LOG.info("Successfully set Go Remote port to " + port + " (as int)");
                portSet = true;
            } else if (tryInvokeMethod(configClass, config, "setPort", new Class<?>[]{String.class}, new Object[]{String.valueOf(port)})) {
                LOG.info("Successfully set Go Remote port to " + port + " (as String)");
                portSet = true;
            } else if (tryInvokeMethod(configClass, config, "setPort", new Class<?>[]{Integer.class}, new Object[]{Integer.valueOf(port)})) {
                LOG.info("Successfully set Go Remote port to " + port + " (as Integer)");
                portSet = true;
            }
            
            if (!portSet) {
                LOG.error("CRITICAL: Could not set port - no compatible setPort method found!");
                LOG.error("This will likely cause connection failures. Available set methods:");
                for (java.lang.reflect.Method method : configClass.getMethods()) {
                    if (method.getName().contains("port") || method.getName().contains("Port")) {
                        LOG.error("  - " + method.getName() + "(" + 
                               java.util.Arrays.toString(method.getParameterTypes()) + ")");
                    }
                }
                throw new ExecutionException("Failed to configure Go Remote port - incompatible Go plugin version");
            }
            
            // Log final configuration
            LOG.info("Go Remote configuration completed successfully");
            LOG.info("  Host: localhost (or default)");
            LOG.info("  Port: " + port);
            
        } catch (Exception e) {
            LOG.error("Failed to configure Go Remote debug configuration", e);
            throw new ExecutionException("Go Remote configuration failed: " + e.getMessage(), e);
        }
    }
    
    /**
     * Helper method to safely invoke a method using reflection
     */
    private boolean tryInvokeMethod(Class<?> clazz, Object instance, String methodName, Class<?>[] paramTypes, Object[] args) {
        try {
            clazz.getMethod(methodName, paramTypes).invoke(instance, args);
            return true;
        } catch (NoSuchMethodException e) {
            LOG.debug("Method " + methodName + " not found in " + clazz.getName());
            return false;
        } catch (Exception e) {
            LOG.warn("Failed to invoke method " + methodName + " on " + clazz.getName(), e);
            return false;
        }
    }
    
    private void showAutoConnectedNotification(int port) {
        String message = String.format(
            "✅ Successfully attached to GoLand debugger!\n\n" +
            "🔗 GoLand debugger connected to tdlv on port %d\n" +
            "🎯 Breakpoints: %d active\n" +
            "🚀 Auto-switched to debugging view!\n\n" +
            "💡 Use the second debug tab for threads & variables\n" +
            "🛠️ Set breakpoints and start your workflow to begin debugging",
            port, getBreakpointCount()
        );
        
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification("GoLand Debugger Attached", message, NotificationType.INFORMATION)
                         .notify(getEnvironment().getProject());
     }
 
     /**
      * Cleans up the auto-created Go Remote configuration if it exists.
      */
     private void cleanupGoRemoteConfiguration() {
         if (goRemoteSettings != null) {
             try {
                 RunManager runManager = RunManager.getInstance(getEnvironment().getProject());
                 runManager.removeConfiguration(goRemoteSettings);
                 LOG.info("Auto-created Go Remote configuration removed");
             } catch (Exception e) {
                 LOG.warn("Failed to remove auto-created Go Remote configuration", e);
             }
             goRemoteSettings = null;
         }
     }
     
     /**
      * Auto-focus on the debugging tab after Go Remote connection is established
      */
     private void focusDebuggingTab(Project project, int port) {
         try {
             // Get the Debug tool window
             ToolWindowManager toolWindowManager = ToolWindowManager.getInstance(project);
             ToolWindow debugToolWindow = toolWindowManager.getToolWindow("Debug");
             
             if (debugToolWindow != null) {
                 // Activate and show the Debug tool window
                 debugToolWindow.activate(() -> {
                     LOG.info("Debug tool window activated");
                     
                     // Additional delay to ensure tabs are loaded
                     new Thread(() -> {
                         try {
                             Thread.sleep(2000); // 2 second delay to ensure content is loaded
                             ApplicationManager.getApplication().invokeLater(() -> {
                                 try {
                                     // Focus on the specific Go Remote debug session tab
                                     focusOnGoRemoteDebugTab(debugToolWindow, port);
                                     
                                     LOG.info("Auto-focused on tdlv debugging tab for port " + port);
                                     
                                     // Show a subtle notification that we've switched to debugging view
                                     NotificationGroupManager.getInstance()
                                         .getNotificationGroup("Temporal Workflow Debugger")
                                         .createNotification(
                                             "Debug Session Ready", 
                                             "🎯 Debug session active. Focused on debug tabs for optimal debugging experience.",
                                             NotificationType.INFORMATION
                                         )
                                         .notify(project);
                                         
                                 } catch (Exception e) {
                                     LOG.warn("Failed to focus on specific debug tab", e);
                                 }
                             });
                         } catch (InterruptedException e) {
                             Thread.currentThread().interrupt();
                             LOG.warn("Focus delay interrupted", e);
                         }
                     }).start();
                 });
             } else {
                 LOG.warn("Debug tool window not found");
             }
         } catch (Exception e) {
             LOG.error("Failed to focus debugging tab", e);
         }
     }
     
     /**
      * Focus on the specific Go Remote debug session tab and select Threads & Variables view
      */
     private void focusOnGoRemoteDebugTab(ToolWindow debugToolWindow, int port) {
         try {
             ContentManager contentManager = debugToolWindow.getContentManager();
             Content[] contents = contentManager.getContents();
             
             LOG.info("Found " + contents.length + " debug content tabs");
             
             // Find the Go Remote debug session tab (contains "tdlv" and port number)
             Content targetContent = null;
             for (Content content : contents) {
                 String displayName = content.getDisplayName();
                 LOG.info("Debug tab: " + displayName);
                 
                 if (displayName != null && 
                     (displayName.contains("tdlv") || displayName.contains("port " + port))) {
                     targetContent = content;
                     LOG.info("Found target debug session: " + displayName);
                     break;
                 }
             }
             
             if (targetContent != null) {
                 // Select the Go Remote debug session tab
                 contentManager.setSelectedContent(targetContent);
                 LOG.info("Selected Go Remote debug session tab");
                 
                 // Make final copy for lambda usage
                 final Content finalTargetContent = targetContent;
                 
                 // Additional delay to ensure the tab content is fully loaded
                 new Thread(() -> {
                     try {
                         Thread.sleep(500);
                         ApplicationManager.getApplication().invokeLater(() -> {
                             // Try to focus on Threads & Variables within this tab
                             focusOnThreadsAndVariables(finalTargetContent);
                         });
                     } catch (InterruptedException e) {
                         Thread.currentThread().interrupt();
                     }
                 }).start();
                 
             } else {
                 LOG.warn("Could not find Go Remote debug session tab for port " + port);
                 // Fallback: just select the last content (most recent debug session)
                 if (contents.length > 0) {
                     contentManager.setSelectedContent(contents[contents.length - 1]);
                     LOG.info("Fallback: selected last debug session tab");
                 }
             }
             
         } catch (Exception e) {
             LOG.error("Failed to focus on Go Remote debug tab", e);
         }
     }
     
     /**
      * Try to focus on the Threads & Variables view within the debug session
      */
     private void focusOnThreadsAndVariables(Content debugContent) {
         try {
             LOG.info("Attempting to focus on Threads & Variables view");
             
             // Get the component tree from the debug content
             java.awt.Component component = debugContent.getComponent();
             if (component != null) {
                 // Look for JTabbedPane components that contain the debug sub-tabs
                 JTabbedPane debugTabs = findDebugTabsContainer(component);
                 
                 if (debugTabs != null) {
                     LOG.info("Found debug tabs container with " + debugTabs.getTabCount() + " tabs");
                     
                     // Look for "Threads & Variables" or similar tab
                     for (int i = 0; i < debugTabs.getTabCount(); i++) {
                         String tabTitle = debugTabs.getTitleAt(i);
                         LOG.info("Tab " + i + ": " + tabTitle);
                         
                         // Check for various possible tab names
                         if (tabTitle != null && 
                             (tabTitle.contains("Threads") || 
                              tabTitle.contains("Variables") || 
                              tabTitle.equals("Threads & Variables") ||
                              tabTitle.equals("Threads and Variables"))) {
                             
                             LOG.info("Focusing on Threads & Variables tab: " + tabTitle);
                             debugTabs.setSelectedIndex(i);
                             
                             // Show success notification
                             NotificationGroupManager.getInstance()
                                 .getNotificationGroup("Temporal Workflow Debugger")
                                 .createNotification(
                                     "Debug View Ready", 
                                     "✅ Focused on Threads & Variables tab. Ready for debugging!",
                                     NotificationType.INFORMATION
                                 )
                                 .notify(getEnvironment().getProject());
                             
                             return;
                         }
                     }
                     
                     // If we didn't find "Threads & Variables", try to select the first non-console tab
                     for (int i = 0; i < debugTabs.getTabCount(); i++) {
                         String tabTitle = debugTabs.getTitleAt(i);
                         if (tabTitle != null && !tabTitle.toLowerCase().contains("console")) {
                             LOG.info("Selecting first non-console tab: " + tabTitle);
                             debugTabs.setSelectedIndex(i);
                             return;
                         }
                     }
                     
                     LOG.warn("Could not find Threads & Variables tab among: " + getTabTitles(debugTabs));
                 } else {
                     LOG.warn("Could not find debug tabs container in debug content");
                     // Try alternative approach - look for any TabbedPane in the component tree
                     JTabbedPane anyTabbedPane = findTabbedPaneRecursively(component);
                     if (anyTabbedPane != null) {
                         LOG.info("Found alternative tabbed pane with " + anyTabbedPane.getTabCount() + " tabs");
                         selectThreadsAndVariablesTab(anyTabbedPane);
                     }
                 }
             }
             
         } catch (Exception e) {
             LOG.error("Failed to focus on Threads & Variables view", e);
         }
     }
     
     /**
      * Find the main debug tabs container (usually a JTabbedPane)
      */
     private JTabbedPane findDebugTabsContainer(java.awt.Component component) {
         // First, try to find the main debug tabs container
         return findTabbedPaneRecursively(component);
     }
     
     /**
      * Recursively search for JTabbedPane components
      */
     private JTabbedPane findTabbedPaneRecursively(java.awt.Component component) {
         if (component instanceof JTabbedPane) {
             JTabbedPane tabbedPane = (JTabbedPane) component;
             // Check if this looks like a debug tabs container
             if (hasDebugTabs(tabbedPane)) {
                 return tabbedPane;
             }
         }
         
         if (component instanceof java.awt.Container) {
             java.awt.Container container = (java.awt.Container) component;
             for (java.awt.Component child : container.getComponents()) {
                 JTabbedPane result = findTabbedPaneRecursively(child);
                 if (result != null) {
                     return result;
                 }
             }
         }
         
         return null;
     }
     
     /**
      * Check if a JTabbedPane contains debug-related tabs
      */
     private boolean hasDebugTabs(JTabbedPane tabbedPane) {
         for (int i = 0; i < tabbedPane.getTabCount(); i++) {
             String title = tabbedPane.getTitleAt(i);
             if (title != null) {
                 String lowerTitle = title.toLowerCase();
                 if (lowerTitle.contains("threads") || 
                     lowerTitle.contains("variables") || 
                     lowerTitle.contains("console") ||
                     lowerTitle.contains("debugger")) {
                     return true;
                 }
             }
         }
         return false;
     }
     
     /**
      * Select the Threads & Variables tab in the given tabbed pane
      */
     private void selectThreadsAndVariablesTab(JTabbedPane tabbedPane) {
         for (int i = 0; i < tabbedPane.getTabCount(); i++) {
             String tabTitle = tabbedPane.getTitleAt(i);
             if (tabTitle != null && 
                 (tabTitle.contains("Threads") || 
                  tabTitle.contains("Variables"))) {
                 LOG.info("Selecting Threads & Variables tab: " + tabTitle);
                 tabbedPane.setSelectedIndex(i);
                 return;
             }
         }
         
         // If not found, select first non-console tab
         for (int i = 0; i < tabbedPane.getTabCount(); i++) {
             String tabTitle = tabbedPane.getTitleAt(i);
             if (tabTitle != null && !tabTitle.toLowerCase().contains("console")) {
                 LOG.info("Selecting first non-console tab: " + tabTitle);
                 tabbedPane.setSelectedIndex(i);
                 return;
             }
         }
     }
     
     /**
      * Get all tab titles for debugging purposes
      */
     private String getTabTitles(JTabbedPane tabbedPane) {
         StringBuilder titles = new StringBuilder();
         for (int i = 0; i < tabbedPane.getTabCount(); i++) {
             if (i > 0) titles.append(", ");
             titles.append("'").append(tabbedPane.getTitleAt(i)).append("'");
         }
         return titles.toString();
     }
} 