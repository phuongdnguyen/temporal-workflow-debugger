package com.temporal.wfdebugger.debug;

import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory;
import com.intellij.openapi.options.ConfigurationException;
import com.intellij.openapi.options.SettingsEditor;
import com.intellij.openapi.ui.TextFieldWithBrowseButton;
import com.intellij.ui.components.JBLabel;
import com.intellij.ui.components.JBTextField;
import com.intellij.util.ui.FormBuilder;
import org.jetbrains.annotations.NotNull;

import javax.swing.*;

/**
 * Editor for Temporal Workflow Debug run configuration.
 * Allows users to configure working directory and additional arguments.
 * Users must provide a pre-built tdlv binary path.
 */
public class WfDebugRunConfigurationEditor extends SettingsEditor<WfDebugRunConfiguration> {

    private JPanel panel;
    private TextFieldWithBrowseButton workingDirectoryField;
    private JBTextField additionalArgsField;

    @Override
    protected void resetEditorFrom(@NotNull WfDebugRunConfiguration configuration) {
        workingDirectoryField.setText(configuration.getWorkingDirectory());
        additionalArgsField.setText(configuration.getAdditionalArgs());
    }

    @Override
    protected void applyEditorTo(@NotNull WfDebugRunConfiguration configuration) throws ConfigurationException {
        configuration.setWorkingDirectory(workingDirectoryField.getText().trim());
        configuration.setAdditionalArgs(additionalArgsField.getText().trim());
    }

    @Override
    protected @NotNull JComponent createEditor() {
        if (panel == null) {
            createUI();
        }
        return panel;
    }

    private void createUI() {
        workingDirectoryField = new TextFieldWithBrowseButton();
        additionalArgsField = new JBTextField();

        // Configure working directory field
        workingDirectoryField.addBrowseFolderListener(
                "Select Working Directory",
                "Choose the directory containing your workflow code to debug",
                null,
                FileChooserDescriptorFactory.createSingleFolderDescriptor());

        // Create informational label
        JBLabel infoLabel = new JBLabel(
                "<html><small>You must provide a pre-built tdlv binary path in Settings → Tools → Temporal Workflow Debugger.<br/>"
                        +
                        "To build tdlv, use the Makefile in your wf-debugger project that contains:<br/>" +
                        "• Makefile<br/>" +
                        "• custom-debugger/ directory<br/>" +
                        "• my-wf/ directory</small></html>");
        infoLabel.setForeground(javax.swing.UIManager.getColor("Label.disabledForeground"));

        // Build the form
        panel = FormBuilder.createFormBuilder()
                .addLabeledComponent(new JBLabel("Working Directory:"), workingDirectoryField, true)
                .addTooltip("Directory containing your workflow code to debug")
                .addComponent(infoLabel)
                .addSeparator(5)
                .addLabeledComponent(new JBLabel("Additional Arguments:"), additionalArgsField, true)
                .addTooltip("Additional command line arguments to pass to tdlv (optional)")
                .addComponent(new JBLabel("<html><small>Optional arguments:<br/>" +
                        "• -h: Show help<br/>" +
                        "• -v: Verbose logging<br/>" +
                        "Note: Proxy port is handled automatically</small></html>"))
                .addComponentFillVertically(new JPanel(), 0)
                .getPanel();
    }
}