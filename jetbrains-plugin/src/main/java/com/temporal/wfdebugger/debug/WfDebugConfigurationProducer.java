package com.temporal.wfdebugger.debug;

import com.intellij.execution.actions.ConfigurationContext;
import com.intellij.execution.actions.LazyRunConfigurationProducer;
import com.intellij.execution.configurations.ConfigurationFactory;
import com.intellij.openapi.util.Ref;
import com.intellij.psi.PsiElement;
import com.intellij.psi.PsiFile;
import org.jetbrains.annotations.NotNull;

import java.io.File;

/**
 * Configuration producer for Temporal Workflow Debug configurations.
 * Automatically detects wf-debugger projects and creates appropriate debug configurations.
 */
public class WfDebugConfigurationProducer extends LazyRunConfigurationProducer<WfDebugRunConfiguration> {
    
    @Override
    public @NotNull ConfigurationFactory getConfigurationFactory() {
        return WfDebugConfigurationType.getInstance().getConfigurationFactories()[0];
    }
    
    @Override
    protected boolean setupConfigurationFromContext(@NotNull WfDebugRunConfiguration configuration,
                                                     @NotNull ConfigurationContext context,
                                                     @NotNull Ref<PsiElement> sourceElement) {
        
        // Get the file that triggered this context
        PsiFile psiFile = context.getPsiLocation() != null ? context.getPsiLocation().getContainingFile() : null;
        if (psiFile == null) {
            return false;
        }
        
        // Get the file path to determine if we're in a wf-debugger project
        String filePath = psiFile.getVirtualFile() != null ? psiFile.getVirtualFile().getPath() : null;
        if (filePath == null) {
            return false;
        }
        
        // Find the wf-debugger project root
        File wfDebuggerRoot = findWfDebuggerRoot(new File(filePath));
        if (wfDebuggerRoot == null) {
            return false;
        }
        
        // Set up the configuration
        configuration.setWorkingDirectory(wfDebuggerRoot.getAbsolutePath());
        configuration.setName("Debug Temporal Workflow");
        
        // Proxy port is handled automatically - no additional args needed by default
        
        return true;
    }
    
    @Override
    public boolean isConfigurationFromContext(@NotNull WfDebugRunConfiguration configuration,
                                              @NotNull ConfigurationContext context) {
        
        // Get the file that triggered this context
        PsiFile psiFile = context.getPsiLocation() != null ? context.getPsiLocation().getContainingFile() : null;
        if (psiFile == null) {
            return false;
        }
        
        // Get the file path
        String filePath = psiFile.getVirtualFile() != null ? psiFile.getVirtualFile().getPath() : null;
        if (filePath == null) {
            return false;
        }
        
        // Find the wf-debugger project root
        File wfDebuggerRoot = findWfDebuggerRoot(new File(filePath));
        if (wfDebuggerRoot == null) {
            return false;
        }
        
        // Check if the configuration's working directory matches this project
        String configWorkingDir = configuration.getWorkingDirectory();
        return configWorkingDir != null && 
               (configWorkingDir.equals(wfDebuggerRoot.getAbsolutePath()) ||
                wfDebuggerRoot.getAbsolutePath().startsWith(configWorkingDir));
    }
    
    /**
     * Finds the wf-debugger project root by looking for the characteristic files/directories.
     */
    private File findWfDebuggerRoot(File startFile) {
        File current = startFile.isDirectory() ? startFile : startFile.getParentFile();
        
        while (current != null) {
            if (isWfDebuggerProject(current)) {
                return current;
            }
            current = current.getParentFile();
        }
        
        return null;
    }
    
    /**
     * Checks if a directory is a wf-debugger project root.
     */
    private boolean isWfDebuggerProject(File dir) {
        if (!dir.isDirectory()) {
            return false;
        }
        
        // Check for characteristic files/directories of a wf-debugger project
        boolean hasMakefile = new File(dir, "Makefile").exists();
        boolean hasDelveWrapper = new File(dir, "delve_wrapper").isDirectory();
        boolean hasMyWf = new File(dir, "my-wf").isDirectory();
        
        return hasMakefile && hasDelveWrapper && hasMyWf;
    }
} 