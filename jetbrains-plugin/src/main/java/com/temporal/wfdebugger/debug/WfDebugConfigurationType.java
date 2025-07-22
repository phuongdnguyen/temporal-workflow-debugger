package com.temporal.wfdebugger.debug;

import com.intellij.execution.configurations.ConfigurationFactory;
import com.intellij.execution.configurations.ConfigurationType;
import com.intellij.execution.configurations.RunConfiguration;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.util.NotNullLazyValue;
import org.jetbrains.annotations.Nls;
import org.jetbrains.annotations.NonNls;
import org.jetbrains.annotations.NotNull;

import javax.swing.*;

/**
 * Configuration type for Temporal Workflow debugging using tdlv.
 * This integrates with IntelliJ's run/debug system.
 */
public class WfDebugConfigurationType implements ConfigurationType {
    
    private static final String ID = "TemporalWorkflowDebug";
    private static final String DISPLAY_NAME = "Temporal Workflow Debug";
    private static final WfDebugConfigurationType INSTANCE = new WfDebugConfigurationType();
    
    private final NotNullLazyValue<ConfigurationFactory> myFactory = NotNullLazyValue.lazy(() -> new WfDebugConfigurationFactory(this));
    
    public static WfDebugConfigurationType getInstance() {
        return INSTANCE;
    }
    
    @Override
    public @NotNull @Nls(capitalization = Nls.Capitalization.Title) String getDisplayName() {
        return DISPLAY_NAME;
    }
    
    @Override
    public @Nls(capitalization = Nls.Capitalization.Sentence) String getConfigurationTypeDescription() {
        return "Debug Temporal workflows using the tdlv debugger";
    }
    
    @Override
    public Icon getIcon() {
        // Use a debug icon from IntelliJ's built-in icons
        return com.intellij.icons.AllIcons.Actions.StartDebugger;
    }
    
    @Override
    public @NotNull @NonNls String getId() {
        return ID;
    }
    
    @Override
    public ConfigurationFactory[] getConfigurationFactories() {
        return new ConfigurationFactory[]{myFactory.getValue()};
    }
    
    /**
     * Factory for creating debug configurations
     */
    private static class WfDebugConfigurationFactory extends ConfigurationFactory {
        
        protected WfDebugConfigurationFactory(@NotNull ConfigurationType type) {
            super(type);
        }
        
        @Override
        public @NotNull RunConfiguration createTemplateConfiguration(@NotNull Project project) {
            return new WfDebugRunConfiguration(project, this, "Temporal Workflow Debug");
        }
        
        @Override
        public @NotNull @Nls(capitalization = Nls.Capitalization.Title) String getName() {
            return DISPLAY_NAME;
        }
        
        @Override
        public @NotNull String getId() {
            return "TemporalWorkflowDebugFactory";
        }
    }
} 