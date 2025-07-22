package com.temporal.wfdebugger.ui;

import com.intellij.icons.AllIcons;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.ui.components.JBLabel;
import com.intellij.ui.components.JBList;
import com.intellij.ui.components.JBScrollPane;
import com.intellij.util.ui.FormBuilder;
import com.intellij.util.ui.UIUtil;
import com.intellij.ui.JBColor;
import com.temporal.wfdebugger.model.HistoryEvent;
import com.temporal.wfdebugger.service.WfDebuggerService;

import javax.swing.*;
import javax.swing.border.EmptyBorder;
import java.awt.*;
import java.awt.event.ActionEvent;
import java.awt.event.ActionListener;
import java.awt.event.MouseAdapter;
import java.awt.event.MouseEvent;
import java.util.List;

/**
 * Panel that displays workflow history events with breakpoint toggle functionality.
 * Users can click on events to toggle breakpoints similar to IntelliJ's breakpoint interface.
 */
public class HistoryPanel {
    
    private final Project project;
    private final WfDebuggerService debuggerService;
    
    private JPanel mainPanel;
    private JBList<HistoryEventListItem> eventsList;
    private DefaultListModel<HistoryEventListItem> listModel;
    private JBLabel statusLabel;
    private JButton refreshButton;
    private JButton clearAllBreakpointsButton;
    private JButton startDebugButton;
    private JButton debugStateButton;
    
    public HistoryPanel(Project project) {
        this.project = project;
        this.debuggerService = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        
        createUI();
        refreshEventsList();
    }
    
    private void createUI() {
        // Initialize components
        listModel = new DefaultListModel<>();
        eventsList = new JBList<>(listModel);
        statusLabel = new JBLabel("No events loaded");
        refreshButton = new JButton("Refresh", AllIcons.Actions.Refresh);
        clearAllBreakpointsButton = new JButton("Clear All Breakpoints", AllIcons.Debugger.Db_invalid_breakpoint);
        startDebugButton = new JButton("Start Debug", AllIcons.Actions.StartDebugger);
        debugStateButton = new JButton("Debug State", AllIcons.Actions.Show);
        startDebugButton.setEnabled(false);
        
        // Configure events list
        eventsList.setCellRenderer(new HistoryEventCellRenderer());
        eventsList.setSelectionMode(ListSelectionModel.SINGLE_SELECTION);
        
        // Add mouse listener for breakpoint toggling
        eventsList.addMouseListener(new MouseAdapter() {
            @Override
            public void mouseClicked(MouseEvent e) {
                if (e.getClickCount() == 1) {
                    handleEventClick(e);
                }
            }
        });
        
        // Configure button actions
        refreshButton.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                refreshEventsList();
            }
        });
        
        clearAllBreakpointsButton.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                clearAllBreakpoints();
            }
        });

        // Start debug action
        startDebugButton.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                triggerStartDebug();
            }
        });
        
        // Debug state action - shows current breakpoint state
        debugStateButton.addActionListener(new ActionListener() {
            @Override
            public void actionPerformed(ActionEvent e) {
                debugCurrentState();
            }
        });
        
        // Create toolbar
        JPanel toolbar = new JPanel(new FlowLayout(FlowLayout.LEFT));
        toolbar.add(refreshButton);
        toolbar.add(clearAllBreakpointsButton);
        toolbar.add(startDebugButton);
        toolbar.add(debugStateButton);
        
        // Create main panel
        mainPanel = FormBuilder.createFormBuilder()
            .addComponent(toolbar)
            .addComponentFillVertically(new JBScrollPane(eventsList), 1)
            .addComponent(statusLabel)
            .getPanel();
    }
    
    private void handleEventClick(MouseEvent e) {
        int index = eventsList.locationToIndex(e.getPoint());
        if (index >= 0) {
            HistoryEventListItem item = listModel.getElementAt(index);
            
            // Very precise click detection - only the icon area
            Rectangle cellBounds = eventsList.getCellBounds(index, index);
            if (cellBounds != null) {
                // Only respond to clicks in the first 16 pixels (actual icon size)
                // and within the vertical bounds of the icon
                int iconSize = 16; // Standard icon size
                int iconPadding = 2; // Small padding
                
                if (e.getX() >= iconPadding && e.getX() <= (iconPadding + iconSize)) {
                    // Also check vertical bounds to be more precise
                    int cellHeight = cellBounds.height;
                    int iconY = (cellHeight - iconSize) / 2; // Center vertically
                    int relativeY = e.getY() - cellBounds.y;
                    
                    if (relativeY >= iconY && relativeY <= (iconY + iconSize)) {
                        toggleBreakpoint(item);
                    }
                }
                // Clicking anywhere else on the row does nothing
            }
        }
    }
    
    private void toggleBreakpoint(HistoryEventListItem item) {
        HistoryEvent event = item.getEvent();
        long eventId = event.getEventId();
        
        // Simple toggle - flip the current state
        boolean newState = !item.isBreakpointEnabled();
        
        // Update the service state
        if (newState) {
            debuggerService.addBreakpoint(eventId);
        } else {
            debuggerService.removeBreakpoint(eventId);
        }
        
        // Update the UI item state
        item.setBreakpointEnabled(newState);
        event.setBreakpointEnabled(newState);
        
        // Repaint just this row
        int index = listModel.indexOf(item);
        if (index >= 0) {
            Rectangle cellBounds = eventsList.getCellBounds(index, index);
            if (cellBounds != null) {
                eventsList.repaint(cellBounds);
            }
        }
        
        // Update status
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        updateStatus("Loaded " + listModel.getSize() + " events, " + breakpointCount + " breakpoints set");
    }
    
    private void clearAllBreakpoints() {
        int count = debuggerService.getState().getBreakpointCount();
        if (count == 0) {
            Messages.showInfoMessage(mainPanel, "No breakpoints to clear.", "No Breakpoints");
            return;
        }
        
        int result = Messages.showYesNoDialog(
            mainPanel,
            "Clear all " + count + " breakpoints?",
            "Clear Breakpoints",
            Messages.getQuestionIcon()
        );
        
        if (result == Messages.YES) {
            debuggerService.clearAllBreakpoints();
            refreshEventsList();
            updateStatus("All breakpoints cleared");
        }
    }
    
    private void refreshEventsList() {
        listModel.clear();
        
        // Check if we have history data
        if (!debuggerService.hasHistoryServerData()) {
            updateStatus("No history loaded - upload a history file first");
            startDebugButton.setEnabled(false);
            return;
        }
        
        List<HistoryEvent> events = debuggerService.getState().getLoadedEvents();
        
        // Add events to list model
        for (HistoryEvent event : events) {
            // Check if this event has a breakpoint set
            boolean hasBreakpoint = debuggerService.getState().hasBreakpoint(event.getEventId());
            event.setBreakpointEnabled(hasBreakpoint);
            
            HistoryEventListItem item = new HistoryEventListItem(event);
            listModel.addElement(item);
        }
        
        // Update status
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        updateStatus("Loaded " + events.size() + " events, " + breakpointCount + " breakpoints set");

        // Enable start button if configuration is ready
        startDebugButton.setEnabled(debuggerService.canStartDebugSession());
        
        // Repaint the list
        eventsList.repaint();
    }
    
    private void updateStatus(String message) {
        statusLabel.setText(message);
    }

    private void triggerStartDebug() {
        com.intellij.openapi.actionSystem.ActionManager am = com.intellij.openapi.actionSystem.ActionManager.getInstance();
        com.intellij.openapi.actionSystem.AnAction action = am.getAction("com.temporal.wfdebugger.StartWfDebug");
        if (action != null) {
            com.intellij.openapi.actionSystem.AnActionEvent event = com.intellij.openapi.actionSystem.AnActionEvent.createFromAnAction(
                    action,
                    null,
                    "HistoryPanel",
                    com.intellij.ide.DataManager.getInstance().getDataContext(mainPanel)
            );
            action.actionPerformed(event);
        }
    }
    
    private void debugCurrentState() {
        // Simple debug info
        java.util.Set<Long> serviceBreakpoints = debuggerService.getState().enabledBreakpoints;
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        
        StringBuilder report = new StringBuilder();
        report.append("Total Events: ").append(listModel.getSize()).append("\n");
        report.append("Breakpoints Set: ").append(breakpointCount).append("\n");
        report.append("Breakpoint IDs: ").append(serviceBreakpoints).append("\n");
        
        Messages.showInfoMessage(mainPanel, report.toString(), "Debug Info");
    }
    
    public JComponent getComponent() {
        return mainPanel;
    }
    
    /**
     * Refresh the events list to update highlighting
     * This should be called when the highlighted event changes
     */
    public void refreshHighlighting() {
        // Trigger a repaint of the list to update highlighting
        if (eventsList != null) {
            eventsList.repaint();
        }
    }
    
    /**
     * Wrapper class for HistoryEvent to display in the list
     */
    private static class HistoryEventListItem {
        private final HistoryEvent event;
        private boolean breakpointEnabled;
        
        public HistoryEventListItem(HistoryEvent event) {
            this.event = event;
            this.breakpointEnabled = event.isBreakpointEnabled();
        }
        
        public HistoryEvent getEvent() {
            return event;
        }
        
        public boolean isBreakpointEnabled() {
            return breakpointEnabled;
        }
        
        public void setBreakpointEnabled(boolean enabled) {
            this.breakpointEnabled = enabled;
            this.event.setBreakpointEnabled(enabled);
        }
        
        @Override
        public String toString() {
            return event.getDisplayName();
        }
    }
    
    /**
     * Custom cell renderer for history events with breakpoint icons and highlighting
     */
    private static class HistoryEventCellRenderer extends DefaultListCellRenderer {
        
        @Override
        public Component getListCellRendererComponent(
                JList<?> list, Object value, int index,
                boolean isSelected, boolean cellHasFocus) {
            
            super.getListCellRendererComponent(list, value, index, isSelected, cellHasFocus);
            
            if (value instanceof HistoryEventListItem) {
                HistoryEventListItem item = (HistoryEventListItem) value;
                HistoryEvent event = item.getEvent();
                
                // Check if this event is currently highlighted (being debugged)
                WfDebuggerService debuggerService = com.intellij.openapi.application.ApplicationManager.getApplication().getService(WfDebuggerService.class);
                boolean isHighlighted = debuggerService.isEventHighlighted(event.getEventId());
                
                // Simple icon based on breakpoint state
                if (item.isBreakpointEnabled()) {
                    setIcon(AllIcons.Debugger.Db_set_breakpoint); // Red filled circle
                } else {
                    setIcon(AllIcons.General.Remove); // Empty circle
                }
                
                // Human-readable text display with ID first
                String displayText = "ID: " + event.getEventId() + " - " + event.getHumanReadableEventType();
                
                // Add event type hints for context
                if (event.isActivityEvent()) {
                    displayText += " - Activity";
                } else if (event.isTimerEvent()) {
                    displayText += " - Timer";
                } else if (event.isWorkflowExecutionEvent()) {
                    displayText += " - Workflow";
                }
                
                // Add highlight indicator to text
                if (isHighlighted) {
                    displayText = "▶ " + displayText + " ◀";
                }
                
                setText(displayText);
                setToolTipText(String.format(
                    "<html>" +
                    "<b>%s</b><br/>" +
                    "Technical Type: %s<br/>" +
                    "Event ID: %d<br/>" +
                    "Time: %s<br/>" +
                    "%s" +
                    "<i>Click the icon to toggle breakpoint</i>" +
                    "</html>",
                    event.getHumanReadableEventType(),
                    event.getEventType(),
                    event.getEventId(),
                    event.getEventTime() != null ? event.getEventTime() : "N/A",
                    isHighlighted ? "<b>⚡ Currently being debugged</b><br/>" : ""
                ));
                
                // Apply highlighting background and colors
                if (isHighlighted) {
                    if (!isSelected) {
                        setBackground(JBColor.namedColor("Debugger.currentExecutionPointBackground", 
                            new JBColor(0xFFFF9680, 0x4C4C0080))); // Theme-aware highlight
                        setOpaque(true);
                        setForeground(UIUtil.getListForeground()); // Use theme foreground
                        
                        // Add a thicker border for highlighted items
                        setBorder(javax.swing.BorderFactory.createCompoundBorder(
                            javax.swing.BorderFactory.createLineBorder(JBColor.namedColor("Component.focusColor", 
                                new JBColor(0xFFA500, 0xFF8C00)), 2), // Theme-aware border
                            new EmptyBorder(2, 5, 2, 5)
                        ));
                    } else {
                        // Keep selection colors but add border
                        setBackground(UIUtil.getListSelectionBackground(true));
                        setForeground(UIUtil.getListSelectionForeground(true));
                        setBorder(javax.swing.BorderFactory.createCompoundBorder(
                            javax.swing.BorderFactory.createLineBorder(JBColor.namedColor("Component.focusColor", 
                                new JBColor(0xFFA500, 0xFF8C00)), 2),
                            new EmptyBorder(2, 5, 2, 5)
                        ));
                    }
                } else {
                    // Standard colors for non-highlighted items
                    if (!isSelected) {
                        setBackground(UIUtil.getListBackground());
                        setOpaque(false);
                        
                        if (event.isWorkflowExecutionEvent()) {
                            setForeground(JBColor.namedColor("Component.infoForeground", 
                                new JBColor(0x0066CC, 0x5394EC))); // Blue
                        } else if (event.isActivityEvent()) {
                            setForeground(JBColor.namedColor("Component.validForeground", 
                                new JBColor(0x009933, 0x62C554))); // Green  
                        } else if (event.isTimerEvent()) {
                            setForeground(JBColor.namedColor("Component.warningForeground", 
                                new JBColor(0xCC6600, 0xE1A336))); // Orange
                        } else {
                            setForeground(UIUtil.getListForeground());
                        }
                    } else {
                        setBackground(UIUtil.getListSelectionBackground(true));
                        setForeground(UIUtil.getListSelectionForeground(true));
                    }
                    
                    setBorder(new EmptyBorder(2, 5, 2, 5));
                }
            }
            
            return this;
        }
    }
} 