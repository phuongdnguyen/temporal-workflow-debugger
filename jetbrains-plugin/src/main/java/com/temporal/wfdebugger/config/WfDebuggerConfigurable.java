package com.temporal.wfdebugger.config;

import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory;
import com.intellij.openapi.options.Configurable;
import com.intellij.openapi.options.ConfigurationException;
import com.intellij.openapi.ui.TextFieldWithBrowseButton;
import com.intellij.ui.components.JBLabel;
import com.intellij.ui.components.JBTextField;
import com.intellij.util.ui.FormBuilder;
import com.temporal.wfdebugger.model.WfDebuggerState;
import com.temporal.wfdebugger.service.WfDebuggerService;
import org.jetbrains.annotations.Nls;
import org.jetbrains.annotations.Nullable;

import javax.swing.*;
import java.io.File;

/**
 * Configuration panel for the Temporal Workflow Debugger plugin.
 * Allows users to set up debug directory and binary path.
 */
public class WfDebuggerConfigurable implements Configurable {

    private JPanel mainPanel;
    private TextFieldWithBrowseButton debugDirectoryField;
    private JBTextField tdlvBinaryPathField;
    
    private WfDebuggerService debuggerService;

    public WfDebuggerConfigurable() {
        this.debuggerService = ApplicationManager.getApplication().getService(WfDebuggerService.class);
    }

    @Nls(capitalization = Nls.Capitalization.Title)
    @Override
    public String getDisplayName() {
        return "Temporal Workflow Debugger";
    }

    @Nullable
    @Override
    public JComponent createComponent() {
        if (mainPanel == null) {
            createUI();
        }
        return mainPanel;
    }

    private void createUI() {
        // Initialize components
        debugDirectoryField = new TextFieldWithBrowseButton();
        tdlvBinaryPathField = new JBTextField();

        // Configure debug directory field
        debugDirectoryField.addBrowseFolderListener(
            "Select Debug Directory",
            "Choose the directory where your workflow code is located",
            null,
            FileChooserDescriptorFactory.createSingleFolderDescriptor()
        );

        // Build the form
        mainPanel = FormBuilder.createFormBuilder()
            .addLabeledComponent(new JBLabel("Runner Directory:"), debugDirectoryField, true)
            .addTooltip("The directory containing your workflow code to debug")
            .addLabeledComponent(new JBLabel("tdlv Binary Path:"), tdlvBinaryPathField, true)
            .addTooltip("Path to the tdlv debugger binary (leave empty to use PATH)")
            .addComponentFillVertically(new JPanel(), 0)
            .getPanel();

        // Load current state
        reset();
    }

    @Override
    public boolean isModified() {
        WfDebuggerState state = debuggerService.getState();
        
        return !state.debugDirectory.equals(debugDirectoryField.getText().trim()) ||
               !state.tdlvBinaryPath.equals(tdlvBinaryPathField.getText().trim());
    }

    @Override
    public void apply() throws ConfigurationException {
        // Validate configuration
        String debugDir = debugDirectoryField.getText().trim();
        if (!debugDir.isEmpty()) {
            File dir = new File(debugDir);
            if (!dir.exists() || !dir.isDirectory()) {
                throw new ConfigurationException("Debug directory does not exist or is not a directory: " + debugDir);
            }
        }

        String binaryPath = tdlvBinaryPathField.getText().trim();
        if (binaryPath.isEmpty()) {
            binaryPath = "tdlv"; // Default to PATH lookup
        }

        // Save configuration
        WfDebuggerState state = debuggerService.getState();
        state.debugDirectory = debugDir;
        state.tdlvBinaryPath = binaryPath;
    }

    @Override
    public void reset() {
        WfDebuggerState state = debuggerService.getState();
        
        debugDirectoryField.setText(state.debugDirectory);
        tdlvBinaryPathField.setText(state.tdlvBinaryPath);
    }

    @Override
    public void disposeUIResources() {
        mainPanel = null;
        debugDirectoryField = null;
        tdlvBinaryPathField = null;
    }
} 