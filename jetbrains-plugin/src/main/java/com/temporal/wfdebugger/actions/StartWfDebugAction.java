package com.temporal.wfdebugger.actions;

import com.intellij.execution.RunManager;
import com.intellij.execution.RunnerAndConfigurationSettings;
import com.intellij.execution.executors.DefaultDebugExecutor;
import com.intellij.execution.runners.ExecutionUtil;
import com.intellij.openapi.actionSystem.ActionUpdateThread;
import com.intellij.openapi.actionSystem.AnAction;
import com.intellij.openapi.actionSystem.AnActionEvent;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.diagnostic.Logger;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.temporal.wfdebugger.debug.WfDebugConfigurationType;
import com.temporal.wfdebugger.debug.WfDebugRunConfiguration;
import com.temporal.wfdebugger.service.WfDebuggerService;
import org.jetbrains.annotations.NotNull;

/**
 * Action to start a Temporal Workflow debug session.
 * This creates and executes a debug configuration using the tdlv debugger.
 */
public class StartWfDebugAction extends AnAction {
    
    private static final Logger LOG = Logger.getInstance(StartWfDebugAction.class);
    
    @Override
    public @NotNull ActionUpdateThread getActionUpdateThread() {
        return ActionUpdateThread.BGT;
    }
    
    @Override
    public void actionPerformed(@NotNull AnActionEvent e) {
        Project project = e.getProject();
        if (project == null) return;

        // Get the debugger service
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        
        // Validate configuration
        String runnerDir = service.getDebugDirectory();
        if (runnerDir == null || runnerDir.trim().isEmpty()) {
            Messages.showErrorDialog(
                project,
                "Runner directory is not configured. Please set it in Settings → Tools → Temporal Workflow Debugger.",
                "Configuration Required"
            );
            return;
        }

        // Check if tdlv binary is configured and exists
        String tdlvPath = service.getTdlvBinaryPath();
        if (tdlvPath == null || tdlvPath.trim().isEmpty()) {
            Messages.showErrorDialog(
                project,
                "tdlv binary path is not configured. Please set it in Settings → Tools → Temporal Workflow Debugger.",
                "Configuration Required"
            );
            return;
        }

        // Check if there are any breakpoints set
        int breakpointCount = service.getState().getBreakpointCount();
        if (breakpointCount == 0) {
            int result = Messages.showYesNoDialog(
                project,
                "No breakpoints are currently set. Start debug session anyway?\n\n" +
                "You can set breakpoints by clicking the red dots next to events in the history panel.",
                "No Breakpoints Set",
                Messages.getQuestionIcon()
            );
            if (result != Messages.YES) {
                return;
            }
        }

        LOG.info("Starting Temporal Workflow debug session with " + breakpointCount + " breakpoints");

        // Show starting notification
        com.intellij.notification.NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification(
                "Temporal Workflow Debugger", 
                "🚀 Starting debug session...\nBreakpoints: " + breakpointCount,
                com.intellij.notification.NotificationType.INFORMATION
            )
            .notify(project);

        // Create and start the Temporal Workflow Debug configuration
        RunManager runManager = RunManager.getInstance(project);
        
        // Get the Temporal Workflow Debug configuration type
        WfDebugConfigurationType configType = WfDebugConfigurationType.getInstance();
        
        // Create a new configuration
        RunnerAndConfigurationSettings settings = runManager.createConfiguration(
            "Temporal Workflow Debug",
            configType.getConfigurationFactories()[0]
        );
        
        // Configure it
        WfDebugRunConfiguration config = (WfDebugRunConfiguration) settings.getConfiguration();
        config.setWorkingDirectory(runnerDir);
        // Proxy port is handled automatically - no need to set additional args
        
        // Add it to run manager
        runManager.addConfiguration(settings);
        runManager.setSelectedConfiguration(settings);
        
        // Start debugging
        ExecutionUtil.runConfiguration(settings, DefaultDebugExecutor.getDebugExecutorInstance());
    }
    
    @Override
    public void update(@NotNull AnActionEvent e) {
        Project project = e.getProject();
        boolean enabled = project != null;
        
        if (enabled) {
            WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
            
            // Update action text based on current state
            if (service.getState().debugSessionActive) {
                e.getPresentation().setText("Restart Temporal Workflow Debug");
                e.getPresentation().setDescription("Restart the temporal workflow debugging session");
            } else {
                e.getPresentation().setText("Start Temporal Workflow Debug");
                e.getPresentation().setDescription("Start debugging with Temporal Workflow Debugger");
            }
            
            // Enable only if basic configuration is available
            enabled = service.canStartDebugSession();
        }
        
        e.getPresentation().setEnabled(enabled);
    }
} 