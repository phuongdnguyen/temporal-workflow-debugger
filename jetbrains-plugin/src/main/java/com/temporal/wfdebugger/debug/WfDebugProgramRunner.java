package com.temporal.wfdebugger.debug;

import com.intellij.execution.ExecutionException;
import com.intellij.execution.ExecutionResult;
import com.intellij.execution.configurations.RunProfile;
import com.intellij.execution.configurations.RunProfileState;
import com.intellij.execution.runners.ExecutionEnvironment;
import com.intellij.execution.runners.GenericProgramRunner;
import com.intellij.execution.ui.RunContentDescriptor;
import com.intellij.openapi.diagnostic.Logger;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

/**
 * Program runner for Temporal Workflow Debug configurations.
 * This handles the actual execution of our debug configuration.
 */
public class WfDebugProgramRunner extends GenericProgramRunner {
    
    private static final Logger LOG = Logger.getInstance(WfDebugProgramRunner.class);
    
    @NotNull
    @Override
    public String getRunnerId() {
        return "TemporalWorkflowDebugRunner";
    }
    
    @Override
    public boolean canRun(@NotNull String executorId, @NotNull RunProfile profile) {
        return profile instanceof WfDebugRunConfiguration;
    }
    
    @Nullable
    @Override
    protected RunContentDescriptor doExecute(
            @NotNull RunProfileState state,
            @NotNull ExecutionEnvironment environment) throws ExecutionException {
        
        LOG.info("Executing Temporal Workflow Debug configuration");
        
        ExecutionResult executionResult = state.execute(environment.getExecutor(), this);
        if (executionResult == null) {
            return null;
        }
        
        return new RunContentDescriptor(
            executionResult.getExecutionConsole(),
            executionResult.getProcessHandler(),
            executionResult.getExecutionConsole().getComponent(),
            environment.getRunProfile().getName(),
            null
        );
    }
} 