package com.temporal.wfdebugger.debug;

import com.intellij.execution.ExecutionException;
import com.intellij.execution.Executor;
import com.intellij.execution.configurations.*;
import com.intellij.execution.runners.ExecutionEnvironment;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.options.SettingsEditor;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.util.InvalidDataException;
import com.intellij.openapi.util.WriteExternalException;
import com.temporal.wfdebugger.service.WfDebuggerService;
import org.jdom.Element;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

import java.io.File;

/**
 * Run configuration for debugging Temporal workflows with tdlv.
 * This executes a pre-built tdlv debugger binary.
 */
public class WfDebugRunConfiguration extends RunConfigurationBase<WfDebugRunState> {

    private String workingDirectory = "";
    private String tdlvBinaryPath = ""; // Path to pre-built tdlv binary
    private String additionalArgs = "";

    protected WfDebugRunConfiguration(@NotNull Project project, @NotNull ConfigurationFactory factory,
            @NotNull String name) {
        super(project, factory, name);

        // Initialize with better defaults
        initializeDefaults(project);
    }

    private void initializeDefaults(@NotNull Project project) {
        // Try to set working directory to project base path or a reasonable default
        if (project.getBasePath() != null) {
            File projectDir = new File(project.getBasePath());

            // Check if we're already in a wf-debugger project
            if (isWfDebuggerProject(projectDir)) {
                this.workingDirectory = project.getBasePath();
            } else {
                // Look for wf-debugger subdirectory
                File wfDebuggerDir = new File(projectDir, "wf-debugger");
                if (wfDebuggerDir.exists() && isWfDebuggerProject(wfDebuggerDir)) {
                    this.workingDirectory = wfDebuggerDir.getAbsolutePath();
                } else {
                    // Default to project base path
                    this.workingDirectory = project.getBasePath();
                }
            }
        }

        // Initialize from plugin settings if available
        WfDebuggerService service = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        if (service.getState().debugDirectory != null && !service.getState().debugDirectory.isEmpty()) {
            this.workingDirectory = service.getState().debugDirectory;
        }

        // Set default additional args (empty - proxy port handled automatically)
        this.additionalArgs = "";
    }

    private boolean isWfDebuggerProject(File dir) {
        return new File(dir, "Makefile").exists() &&
                new File(dir, "tdlv").exists() &&
                new File(dir, "my-wf").exists();
    }

    @Override
    public @NotNull SettingsEditor<? extends RunConfiguration> getConfigurationEditor() {
        return new WfDebugRunConfigurationEditor();
    }

    @Override
    public void checkConfiguration() throws RuntimeConfigurationException {
        if (workingDirectory.isEmpty()) {
            throw new RuntimeConfigurationError("Working directory is not specified");
        }

        File workingDir = new File(workingDirectory);
        if (!workingDir.exists()) {
            throw new RuntimeConfigurationError("Working directory does not exist: " + workingDirectory);
        }

        // Check if we can find the wf-debugger project structure
        if (!findWfDebuggerRoot(workingDir)) {
            throw new RuntimeConfigurationWarning(
                    "Could not find wf-debugger project structure (Makefile, tdlv, my-wf) " +
                            "in or above the working directory. Make sure the working directory is within a wf-debugger project.");
        }
    }

    private boolean findWfDebuggerRoot(File startDir) {
        File current = startDir;
        while (current != null) {
            if (isWfDebuggerProject(current)) {
                return true;
            }
            current = current.getParentFile();
        }
        return false;
    }

    @Override
    public @Nullable RunProfileState getState(@NotNull Executor executor, @NotNull ExecutionEnvironment environment)
            throws ExecutionException {
        return new WfDebugRunState(environment, this);
    }

    @Override
    public void readExternal(@NotNull Element element) throws InvalidDataException {
        super.readExternal(element);

        workingDirectory = element.getAttributeValue("workingDirectory", "");
        tdlvBinaryPath = element.getAttributeValue("tdlvBinaryPath", "");
        additionalArgs = element.getAttributeValue("additionalArgs", "");
    }

    @Override
    public void writeExternal(@NotNull Element element) throws WriteExternalException {
        super.writeExternal(element);

        element.setAttribute("workingDirectory", workingDirectory);
        element.setAttribute("tdlvBinaryPath", tdlvBinaryPath);
        element.setAttribute("additionalArgs", additionalArgs);
    }

    // Getters and setters
    public String getWorkingDirectory() {
        return workingDirectory;
    }

    public void setWorkingDirectory(String workingDirectory) {
        this.workingDirectory = workingDirectory;
    }

    public String getTdlvBinaryPath() {
        return tdlvBinaryPath;
    }

    public void setTdlvBinaryPath(String tdlvBinaryPath) {
        this.tdlvBinaryPath = tdlvBinaryPath;
    }

    public String getAdditionalArgs() {
        return additionalArgs;
    }

    public void setAdditionalArgs(String additionalArgs) {
        this.additionalArgs = additionalArgs;
    }
}