package com.temporal.wfdebugger.actions;

import com.intellij.openapi.actionSystem.ActionUpdateThread;
import com.intellij.openapi.actionSystem.AnAction;
import com.intellij.openapi.actionSystem.AnActionEvent;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.fileChooser.FileChooser;
import com.intellij.openapi.fileChooser.FileChooserDescriptor;
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.openapi.wm.ToolWindow;
import com.intellij.openapi.wm.ToolWindowManager;
import com.temporal.wfdebugger.service.WfDebuggerService;
import org.jetbrains.annotations.NotNull;

/**
 * Action to load a Temporal workflow history JSON file.
 * This allows users to quickly load history files from the Tools menu.
 */
public class LoadHistoryAction extends AnAction {
    
    @Override
    public @NotNull ActionUpdateThread getActionUpdateThread() {
        return ActionUpdateThread.BGT;
    }
    
    @Override
    public void actionPerformed(@NotNull AnActionEvent e) {
        Project project = e.getProject();
        if (project == null) {
            return;
        }
        
        // Show file chooser for JSON files
        FileChooserDescriptor descriptor = FileChooserDescriptorFactory.createSingleFileDescriptor("json");
        descriptor.setTitle("Select Workflow History JSON File");
        descriptor.setDescription("Choose a Temporal workflow history JSON file to load");
        
        VirtualFile selectedFile = FileChooser.chooseFile(descriptor, project, null);
        if (selectedFile == null) {
            return; // User cancelled
        }
        
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        
        try {
            // Load the history file
            int eventCount = service.loadHistoryFile(selectedFile.getPath());
            
            // Show success message
            Messages.showInfoMessage(
                project,
                "Successfully loaded " + eventCount + " events from " + selectedFile.getName() + 
                "\n\nYou can now set breakpoints in the Temporal Workflow History tool window.",
                "History Loaded"
            );
            
            // Show the history tool window
            showHistoryToolWindow(project);
            
        } catch (Exception ex) {
            Messages.showErrorDialog(
                project,
                "Failed to load history file: " + ex.getMessage(),
                "Load Failed"
            );
        }
    }
    
    @Override
    public void update(@NotNull AnActionEvent e) {
        // Always enable this action when a project is available
        e.getPresentation().setEnabled(e.getProject() != null);
        
        // Update description based on current state
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        if (service.getState().hasHistoryLoaded()) {
            e.getPresentation().setDescription("Load a new workflow history JSON file (replaces current history)");
        } else {
            e.getPresentation().setDescription("Load a temporal workflow history JSON file");
        }
    }
    
    /**
     * Show and activate the Temporal Workflow Debugger tool window
     */
    private void showHistoryToolWindow(Project project) {
        ToolWindowManager toolWindowManager = ToolWindowManager.getInstance(project);
        ToolWindow toolWindow = toolWindowManager.getToolWindow("Temporal Workflow Debugger");
        
        if (toolWindow != null) {
            toolWindow.show();
            toolWindow.activate(null);
        }
    }
} 