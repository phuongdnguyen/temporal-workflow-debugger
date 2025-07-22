package com.temporal.wfdebugger.ui;

import com.intellij.icons.AllIcons;
import com.intellij.openapi.application.ApplicationManager;
import com.intellij.openapi.fileChooser.FileChooser;
import com.intellij.openapi.fileChooser.FileChooserDescriptor;
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.ui.Messages;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.ui.components.JBLabel;
import com.intellij.ui.components.JBList;
import com.intellij.ui.components.JBScrollPane;
import com.intellij.util.ui.FormBuilder;
import com.intellij.util.ui.JBUI;
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
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Main panel for the Temporal Workflow Debugger tool window.
 * Shows upload interface initially, then displays workflow history events with breakpoint controls after upload.
 */
public class WorkflowDebuggerPanel {
    
    private final Project project;
    private final WfDebuggerService debuggerService;
    
    private JPanel mainPanel;
    private CardLayout cardLayout;
    
    // Upload panel components
    private JPanel uploadPanel;
    private JButton uploadButton;
    private JBLabel uploadStatusLabel;
    
    // History panel components
    private JPanel historyPanel;
    private JBList<HistoryEventListItem> eventsList;
    private DefaultListModel<HistoryEventListItem> listModel;
    private JBLabel historyStatusLabel;
    private JButton refreshButton;
    private JButton clearAllBreakpointsButton;
    private JButton startDebugButton;
    private JButton backToUploadButton;
    
    private static final String UPLOAD_CARD = "upload";
    private static final String HISTORY_CARD = "history";
    
    public WorkflowDebuggerPanel(Project project) {
        this.project = project;
        this.debuggerService = ApplicationManager.getApplication().getService(WfDebuggerService.class);
        
        createUI();
        updateUI();
    }
    
    private void createUI() {
        cardLayout = new CardLayout();
        mainPanel = new JPanel(cardLayout);
        
        createUploadPanel();
        createHistoryPanel();
        
        mainPanel.add(uploadPanel, UPLOAD_CARD);
        mainPanel.add(historyPanel, HISTORY_CARD);
        
        // Show upload panel initially
        cardLayout.show(mainPanel, UPLOAD_CARD);
    }
    
    private void createUploadPanel() {
        uploadPanel = new JPanel(new BorderLayout());

        // Cool gradient banner at the top
        BannerPanel banner = new BannerPanel();
        banner.setPreferredSize(new Dimension(1000, 140));
        uploadPanel.add(banner, BorderLayout.NORTH);

        uploadPanel.setBorder(new EmptyBorder(20, 40, 20, 40));
        
        // Create center content
        JPanel centerPanel = new JPanel();
        centerPanel.setLayout(new BoxLayout(centerPanel, BoxLayout.Y_AXIS));
        centerPanel.setAlignmentX(Component.CENTER_ALIGNMENT);
        
        // Title
        JBLabel titleLabel = new JBLabel("Temporal Workflow Debugger", AllIcons.Actions.Lightning, SwingConstants.CENTER);
        titleLabel.setFont(titleLabel.getFont().deriveFont(Font.BOLD, 16f));
        titleLabel.setAlignmentX(Component.LEFT_ALIGNMENT);
        
        // Description
        JBLabel descLabel = new JBLabel("<html><center>Upload a Temporal workflow history JSON file to begin debugging.<br/>You can set breakpoints and analyze workflow execution.</center></html>");
        descLabel.setAlignmentX(Component.LEFT_ALIGNMENT);
        descLabel.setBorder(JBUI.Borders.empty(10, 0));
        
        // Upload button
        uploadButton = new JButton("Select History File", AllIcons.Actions.Upload);
        uploadButton.setAlignmentX(Component.LEFT_ALIGNMENT);
        uploadButton.setPreferredSize(new Dimension(200, 40));
        uploadButton.addActionListener(new UploadActionListener());
        
        // Status label
        uploadStatusLabel = new JBLabel(" ");
        uploadStatusLabel.setAlignmentX(Component.LEFT_ALIGNMENT);
        uploadStatusLabel.setBorder(JBUI.Borders.empty(10, 0));
        
        centerPanel.add(Box.createVerticalGlue());
        centerPanel.add(titleLabel);
        centerPanel.add(Box.createVerticalStrut(10));
        centerPanel.add(descLabel);
        centerPanel.add(Box.createVerticalStrut(20));
        centerPanel.add(uploadButton);
        centerPanel.add(uploadStatusLabel);
        centerPanel.add(Box.createVerticalGlue());
        
        uploadPanel.add(centerPanel, BorderLayout.CENTER);
    }

    /**
     * Simple colourful banner similar to JetBrains AI Assistant splash.
     */
    private static class BannerPanel extends JPanel {
        @Override
        protected void paintComponent(Graphics g) {
            super.paintComponent(g);
            Graphics2D g2d = (Graphics2D) g.create();
            int w = getWidth();
            int h = getHeight();
            // Smooth gradient background
            GradientPaint gp = new GradientPaint(0, 0, new Color(0x00C9FF), w, h, new Color(0x9B00FF));
            g2d.setPaint(gp);
            g2d.fillRect(0, 0, w, h);

            // Draw a swirl-like ellipse overlay
            g2d.setComposite(AlphaComposite.getInstance(AlphaComposite.SRC_OVER, 0.15f));
            g2d.setColor(Color.WHITE);
            int ellW = (int) (h * 2.5);
            g2d.fillOval(-ellW / 4, -h / 2, ellW, h * 2);
            g2d.dispose();
        }
    }
    
    private void createHistoryPanel() {
        historyPanel = new JPanel(new BorderLayout());
        
        // Initialize components
        listModel = new DefaultListModel<>();
        eventsList = new JBList<>(listModel);
        eventsList.setCellRenderer(new EnhancedHistoryEventCellRenderer());
        eventsList.setSelectionMode(ListSelectionModel.SINGLE_SELECTION);
        
        // Enable tooltips for the list
        ToolTipManager.sharedInstance().registerComponent(eventsList);
        
        // Override getToolTipText to show event details on hover
        eventsList = new JBList<HistoryEventListItem>(listModel) {
            @Override
            public String getToolTipText(MouseEvent event) {
                int index = locationToIndex(event.getPoint());
                if (index >= 0) {
                    HistoryEventListItem item = getModel().getElementAt(index);
                    if (item != null) {
                        // Generate tooltip directly without casting renderer
                        WfDebuggerService debuggerService = ApplicationManager.getApplication().getService(WfDebuggerService.class);
                        boolean isHighlighted = debuggerService.isEventHighlighted(item.getEvent().getEventId());
                        return generateDetailedTooltip(item.getEvent(), isHighlighted);
                    }
                }
                return super.getToolTipText(event);
            }
        };
        eventsList.setModel(listModel);
        eventsList.setCellRenderer(new EnhancedHistoryEventCellRenderer());
        eventsList.setSelectionMode(ListSelectionModel.SINGLE_SELECTION);
        
        // Add mouse listener for breakpoint toggling with precise gutter detection
        eventsList.addMouseListener(new MouseAdapter() {
            @Override
            public void mouseClicked(MouseEvent e) {
                if (e.getClickCount() == 1) {
                    handleBreakpointClick(e);
                }
            }
        });
        
        // Add mouse motion listener for hover effects
        eventsList.addMouseMotionListener(new MouseAdapter() {
            @Override
            public void mouseMoved(MouseEvent e) {
                handleMouseHover(e);
            }
            
            @Override
            public void mouseExited(MouseEvent e) {
                // Clear hover state when mouse leaves the list
                clearHoverState();
            }
        });
        
        JBScrollPane scrollPane = new JBScrollPane(eventsList);
        scrollPane.setPreferredSize(new Dimension(400, 300));
        
        // Create toolbar
        JPanel toolbar = new JPanel(new FlowLayout(FlowLayout.LEFT));
        
        backToUploadButton = new JButton("New File", AllIcons.Actions.Upload);
        backToUploadButton.addActionListener(e -> {
            // Open file chooser dialog directly without switching panels
            FileChooserDescriptor descriptor = FileChooserDescriptorFactory.createSingleFileDescriptor("json");
            descriptor.setTitle("Select Workflow History JSON File");
            descriptor.setDescription("Choose a Temporal workflow history JSON file to load");
            
            VirtualFile selectedFile = FileChooser.chooseFile(descriptor, project, null);
            if (selectedFile == null) {
                return; // User cancelled
            }
            
            try {
                // Disable the button while loading
                backToUploadButton.setEnabled(false);
                historyStatusLabel.setText("Loading file...");
                
                // Load the history file
                int eventCount = debuggerService.loadHistoryFile(selectedFile.getPath());
                
                // Refresh the events list immediately to show the new data
                refreshEventsList();
                
                // Show success message in status
                historyStatusLabel.setText("Successfully loaded " + eventCount + " events from " + selectedFile.getName());
                
                // Re-enable the button after a brief delay
                Timer timer = new Timer(1000, event -> {
                    backToUploadButton.setEnabled(true);
                });
                timer.setRepeats(false);
                timer.start();
                
            } catch (Exception ex) {
                backToUploadButton.setEnabled(true);
                historyStatusLabel.setText("Error loading file: " + ex.getMessage());
                Messages.showErrorDialog(
                    project,
                    "Failed to load history file: " + ex.getMessage(),
                    "Load Failed"
                );
            }
        });
        
        refreshButton = new JButton("Refresh", AllIcons.Actions.Refresh);
        refreshButton.addActionListener(e -> refreshEventsList());
        
        clearAllBreakpointsButton = new JButton("Clear All Breakpoints", AllIcons.Actions.GC);
        clearAllBreakpointsButton.addActionListener(e -> clearAllBreakpoints());

        startDebugButton = new JButton("Run", AllIcons.Actions.StartDebugger);
        startDebugButton.setEnabled(false);
        startDebugButton.addActionListener(e -> triggerStartDebug());
        
        toolbar.add(backToUploadButton);
        toolbar.add(refreshButton);
        toolbar.add(clearAllBreakpointsButton);
        toolbar.add(startDebugButton);
        
        // Status area
        historyStatusLabel = new JBLabel("No events loaded");
        historyStatusLabel.setBorder(new EmptyBorder(5, 10, 5, 10));
        
        historyPanel.add(toolbar, BorderLayout.NORTH);
        historyPanel.add(scrollPane, BorderLayout.CENTER);
        historyPanel.add(historyStatusLabel, BorderLayout.SOUTH);
    }
    
    // State tracking for hover effects
    private int hoveredRowIndex = -1;
    private boolean isHoveringInGutter = false;
    
    // Gutter configuration
    private static final int GUTTER_WIDTH = 20;
    private static final int ICON_SIZE = 12;
    private static final int ICON_MARGIN = 4;
    
    /**
     * Handle precise breakpoint clicking only within the circular icon area
     */
    private void handleBreakpointClick(MouseEvent e) {
        int index = eventsList.locationToIndex(e.getPoint());
        if (index >= 0) {
            Rectangle cellBounds = eventsList.getCellBounds(index, index);
            if (cellBounds != null) {
                // Calculate the center and radius of the breakpoint icon
                int centerX = GUTTER_WIDTH / 2;
                int centerY = cellBounds.y + cellBounds.height / 2;
                int radius = ICON_SIZE / 2;
                
                // Calculate distance from click point to center of icon
                int clickX = e.getX();
                int clickY = e.getY();
                double distance = Math.sqrt(Math.pow(clickX - centerX, 2) + Math.pow(clickY - centerY, 2));
                
                // Only toggle if click is within the circular icon area
                if (distance <= radius) {
                    // Request focus if the list doesn't have it
                    if (!eventsList.hasFocus()) {
                        eventsList.requestFocusInWindow();
                        // Give the focus change a moment to process, then toggle breakpoint
                        SwingUtilities.invokeLater(() -> {
                            HistoryEventListItem item = listModel.getElementAt(index);
                            toggleBreakpoint(item);
                        });
                    } else {
                        // List already has focus, toggle immediately
                        HistoryEventListItem item = listModel.getElementAt(index);
                        toggleBreakpoint(item);
                    }
                }
            }
        }
    }
    
    /**
     * Handle mouse hover for gutter visual feedback - only within circular icon area
     */
    private void handleMouseHover(MouseEvent e) {
        int index = eventsList.locationToIndex(e.getPoint());
        boolean wasHovering = isHoveringInGutter;
        int previousHoveredRow = hoveredRowIndex;
        
        isHoveringInGutter = false;
        hoveredRowIndex = -1;
        
        if (index >= 0) {
            Rectangle cellBounds = eventsList.getCellBounds(index, index);
            if (cellBounds != null) {
                // Calculate the center and radius of the breakpoint icon
                int centerX = GUTTER_WIDTH / 2;
                int centerY = cellBounds.y + cellBounds.height / 2;
                int radius = ICON_SIZE / 2;
                
                // Calculate distance from mouse point to center of icon
                int mouseX = e.getX();
                int mouseY = e.getY();
                double distance = Math.sqrt(Math.pow(mouseX - centerX, 2) + Math.pow(mouseY - centerY, 2));
                
                // Only show hover if mouse is within the circular icon area
                if (distance <= radius) {
                    isHoveringInGutter = true;
                    hoveredRowIndex = index;
                }
            }
        }
        
        // Repaint if hover state changed
        if (wasHovering != isHoveringInGutter || previousHoveredRow != hoveredRowIndex) {
            if (previousHoveredRow >= 0 && previousHoveredRow != hoveredRowIndex) {
                // Repaint the previously hovered row
                Rectangle oldBounds = eventsList.getCellBounds(previousHoveredRow, previousHoveredRow);
                if (oldBounds != null) {
                    eventsList.repaint(oldBounds);
                }
            }
            if (hoveredRowIndex >= 0) {
                // Repaint the currently hovered row
                Rectangle newBounds = eventsList.getCellBounds(hoveredRowIndex, hoveredRowIndex);
                if (newBounds != null) {
                    eventsList.repaint(newBounds);
                }
            }
        }
    }
    
    /**
     * Clear hover state when mouse exits
     */
    private void clearHoverState() {
        if (hoveredRowIndex >= 0) {
            Rectangle bounds = eventsList.getCellBounds(hoveredRowIndex, hoveredRowIndex);
            if (bounds != null) {
                eventsList.repaint(bounds);
            }
        }
        hoveredRowIndex = -1;
        isHoveringInGutter = false;
    }
    
    /**
     * Generate a comprehensive tooltip with full event details
     */
    private String generateDetailedTooltip(HistoryEvent event, boolean isHighlighted) {
        StringBuilder tooltip = new StringBuilder();
        tooltip.append("<html><body>");
        
        // Header with event type
        tooltip.append("<b>").append(event.getHumanReadableEventType()).append("</b><br>");
        
        // Highlight indicator
        if (isHighlighted) {
            tooltip.append("<font color='red'>⚡ Currently being debugged</font><br>");
        }
        
        tooltip.append("<hr>");
        
        // Basic information
        tooltip.append("<b>Event ID:</b> ").append(event.getEventId()).append("<br>");
        tooltip.append("<b>Type:</b> ").append(event.getEventType()).append("<br>");
        
        if (event.getEventTime() != null) {
            tooltip.append("<b>Time:</b> ").append(event.getEventTime()).append("<br>");
        }
        
        if (event.getVersion() > 0) {
            tooltip.append("<b>Version:</b> ").append(event.getVersion()).append("<br>");
        }
        
        if (event.getTaskId() > 0) {
            tooltip.append("<b>Task ID:</b> ").append(event.getTaskId()).append("<br>");
        }
        
        // Event attributes/details
        if (event.getAttributes() != null && !event.getAttributes().isEmpty()) {
            tooltip.append("<br><b>Details:</b><br>");
            
            int count = 0;
            for (Map.Entry<String, Object> entry : event.getAttributes().entrySet()) {
                if (count >= 6) { // Limit to prevent tooltip from being too large
                    tooltip.append("... and ").append(event.getAttributes().size() - count).append(" more<br>");
                    break;
                }
                
                String key = entry.getKey();
                Object value = entry.getValue();
                String formattedValue = formatAttributeValue(value);
                
                tooltip.append("&nbsp;&nbsp;<b>").append(key).append(":</b> ").append(formattedValue).append("<br>");
                count++;
            }
        }
        
        // Category information
        tooltip.append("<br><b>Category:</b> ");
        if (event.isWorkflowExecutionEvent()) {
            tooltip.append("Workflow Execution");
        } else if (event.isActivityEvent()) {
            tooltip.append("Activity");
        } else if (event.isTimerEvent()) {
            tooltip.append("Timer");
        } else {
            tooltip.append("Other");
        }
        
        // Footer instructions
        tooltip.append("<hr>");
        tooltip.append("<i>Click the gutter area to toggle breakpoint</i>");
        
        tooltip.append("</body></html>");
        
        return tooltip.toString();
    }
    
    /**
     * Format attribute values for display in tooltip
     */
    private String formatAttributeValue(Object value) {
        if (value == null) {
            return "<i>null</i>";
        }
        
        String stringValue = value.toString();
        
        // Truncate very long values
        if (stringValue.length() > 100) {
            return stringValue.substring(0, 97) + "...";
        }
        
        // Escape HTML characters
        stringValue = stringValue.replace("<", "&lt;")
                               .replace(">", "&gt;")
                               .replace("&", "&amp;");
        
        return stringValue;
    }
    
    private void updateUI() {
        if (debuggerService.getState().hasHistoryLoaded()) {
            refreshEventsList();
            cardLayout.show(mainPanel, HISTORY_CARD);
        } else {
            cardLayout.show(mainPanel, UPLOAD_CARD);
        }
    }
    
    private void updateUploadStatus(String message) {
        uploadStatusLabel.setText(message);
    }
    
    private class UploadActionListener implements ActionListener {
        @Override
        public void actionPerformed(ActionEvent e) {
            FileChooserDescriptor descriptor = FileChooserDescriptorFactory.createSingleFileDescriptor("json");
            descriptor.setTitle("Select Workflow History JSON File");
            descriptor.setDescription("Choose a Temporal workflow history JSON file to load");
            
            VirtualFile selectedFile = FileChooser.chooseFile(descriptor, project, null);
            if (selectedFile == null) {
                return; // User cancelled
            }
            
            try {
                updateUploadStatus("Loading file...");
                uploadButton.setEnabled(false);
                
                // Load the history file
                int eventCount = debuggerService.loadHistoryFile(selectedFile.getPath());
                
                // Show success and switch to history view
                updateUploadStatus("Successfully loaded " + eventCount + " events from " + selectedFile.getName());
                
                // Delay to show the success message briefly
                Timer timer = new Timer(1000, event -> {
                    refreshEventsList();
                    cardLayout.show(mainPanel, HISTORY_CARD);
                    uploadButton.setEnabled(true);
                });
                timer.setRepeats(false);
                timer.start();
                
            } catch (Exception ex) {
                updateUploadStatus("Error: " + ex.getMessage());
                uploadButton.setEnabled(true);
                Messages.showErrorDialog(
                    project,
                    "Failed to load history file: " + ex.getMessage(),
                    "Load Failed"
                );
            }
        }
    }
    
    private void refreshEventsList() {
        listModel.clear();
        
        // Only show events if the history server actually has data (ensures consistency)
        if (!debuggerService.hasHistoryServerData()) {
            historyStatusLabel.setText("No history loaded");
            startDebugButton.setEnabled(false);
            return;
        }
        
        // Check UI/server consistency
        if (!debuggerService.isHistoryStateConsistent()) {
            historyStatusLabel.setText("Warning: UI and server state inconsistent! " + debuggerService.getHistoryStateSummary());
            startDebugButton.setEnabled(false);
            return;
        }
        
        List<HistoryEvent> events = debuggerService.getState().getLoadedEvents();
        
        for (HistoryEvent event : events) {
            HistoryEventListItem item = new HistoryEventListItem(event);
            item.setBreakpointEnabled(event.isBreakpointEnabled());
            listModel.addElement(item);
        }
        
        // Update status
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        historyStatusLabel.setText("Loaded " + events.size() + " events, " + breakpointCount + " breakpoints set");

        // Enable Run button once history is loaded and consistent
        startDebugButton.setEnabled(true);
    }
    
    private void toggleBreakpoint(HistoryEventListItem item) {
        boolean wasEnabled = debuggerService.toggleBreakpoint(item.getEvent().getEventId());
        item.setBreakpointEnabled(!wasEnabled);
        
        // Refresh the list to update the display
        eventsList.repaint();
        
        // Update status
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        List<HistoryEvent> events = debuggerService.getState().getLoadedEvents();
        historyStatusLabel.setText("Loaded " + events.size() + " events, " + breakpointCount + " breakpoints set");
        
        System.out.println("Breakpoint " + (!wasEnabled ? "enabled" : "disabled") + " for event ID: " + item.getEvent().getEventId() + " - " + item.getEvent().getEventType());
    }
    
    private void clearAllBreakpoints() {
        debuggerService.clearAllBreakpoints();
        
        // Update all items in the list
        for (int i = 0; i < listModel.getSize(); i++) {
            HistoryEventListItem item = listModel.getElementAt(i);
            item.setBreakpointEnabled(false);
        }
        
        eventsList.repaint();
        refreshEventsList(); // Update status
        
        Messages.showInfoMessage(project, "All breakpoints have been cleared.", "Breakpoints Cleared");
    }

    private void triggerStartDebug() {
        // Ensure runner directory is set
        if (debuggerService.getDebugDirectory() == null || debuggerService.getDebugDirectory().isEmpty()) {
            com.intellij.openapi.fileChooser.FileChooserDescriptor descriptor = com.intellij.openapi.fileChooser.FileChooserDescriptorFactory.createSingleFolderDescriptor();
            descriptor.setTitle("Select Workflow Entrypoint Directory");
            com.intellij.openapi.vfs.VirtualFile dir = com.intellij.openapi.fileChooser.FileChooser.chooseFile(descriptor, project, null);
            if (dir == null) {
                return; // user cancelled
            }
            debuggerService.setDebugDirectory(dir.getPath());
        }

        // Ensure tdlv binary path set
        if (debuggerService.getTdlvBinaryPath() == null || debuggerService.getTdlvBinaryPath().trim().isEmpty()) {
            com.intellij.openapi.fileChooser.FileChooserDescriptor binDesc = com.intellij.openapi.fileChooser.FileChooserDescriptorFactory.createSingleFileDescriptor();
            binDesc.setTitle("Select tdlv Binary");
            com.intellij.openapi.vfs.VirtualFile bin = com.intellij.openapi.fileChooser.FileChooser.chooseFile(binDesc, project, null);
            if (bin == null) {
                return;
            }
            debuggerService.setTdlvBinaryPath(bin.getPath());
            // Inform user
            com.intellij.openapi.ui.Messages.showInfoMessage(project,
                    "tdlv binary set to: " + bin.getPath(),
                    "tdlv Path Configured");
        }

        // Show non-blocking notification instead of modal dialog
        int breakpointCount = debuggerService.getState().getBreakpointCount();
        com.intellij.notification.NotificationGroupManager.getInstance()
            .getNotificationGroup("Temporal Workflow Debugger")
            .createNotification(
                "Temporal Workflow Debugger", 
                "🚀 Starting debug session...\nBreakpoints: " + breakpointCount + 
                "\nWorkflow: " + debuggerService.getDebugDirectory(),
                com.intellij.notification.NotificationType.INFORMATION
            )
            .notify(project);

        com.intellij.openapi.actionSystem.ActionManager am = com.intellij.openapi.actionSystem.ActionManager.getInstance();
        com.intellij.openapi.actionSystem.AnAction action = am.getAction("com.temporal.wfdebugger.StartWfDebug");
        if (action != null) {
            com.intellij.openapi.actionSystem.AnActionEvent event = com.intellij.openapi.actionSystem.AnActionEvent.createFromAnAction(
                    action,
                    null,
                    "WorkflowDebuggerPanel",
                    com.intellij.ide.DataManager.getInstance().getDataContext(mainPanel)
            );
            action.actionPerformed(event);
        }
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
    
    // Inner classes for list rendering (reusing existing logic)
    private static class HistoryEventListItem {
        private final HistoryEvent event;
        private boolean breakpointEnabled;
        
        public HistoryEventListItem(HistoryEvent event) {
            this.event = event;
            this.breakpointEnabled = false;
        }
        
        public HistoryEvent getEvent() {
            return event;
        }
        
        public boolean isBreakpointEnabled() {
            return breakpointEnabled;
        }
        
        public void setBreakpointEnabled(boolean enabled) {
            this.breakpointEnabled = enabled;
        }
        
        @Override
        public String toString() {
            return "ID: " + event.getEventId() + " - " + event.getHumanReadableEventType();
        }
    }
    
    private class EnhancedHistoryEventCellRenderer extends JPanel implements ListCellRenderer<HistoryEventListItem> {
        private JLabel textLabel;
        private final Color BREAKPOINT_COLOR = JBColor.namedColor("Debugger.breakpointIcon", new JBColor(0xDB5C5C, 0xE74848));
        private final Color HOVER_COLOR = JBColor.namedColor("Debugger.breakpointHover", new JBColor(0xFFB6C1B4, 0xFF6B6B99));
        private final Color GUTTER_BACKGROUND = UIUtil.getPanelBackground();
        
        public EnhancedHistoryEventCellRenderer() {
            setLayout(new BorderLayout());
            textLabel = new JLabel();
            // Add extra left padding to account for the gutter area (GUTTER_WIDTH + some margin)
            textLabel.setBorder(new EmptyBorder(2, GUTTER_WIDTH + 8, 2, 5));
            add(textLabel, BorderLayout.CENTER);
        }
        
        @Override
        public Component getListCellRendererComponent(JList<? extends HistoryEventListItem> list, 
                                                    HistoryEventListItem value, int index, 
                                                    boolean isSelected, boolean cellHasFocus) {
            
            if (value == null) {
                return this;
            }
            
            HistoryEvent event = value.getEvent();
            
            // Check if this event is currently highlighted (being debugged)
            WfDebuggerService debuggerService = ApplicationManager.getApplication().getService(WfDebuggerService.class);
            boolean isHighlighted = debuggerService.isEventHighlighted(event.getEventId());
            boolean hasBreakpoint = value.isBreakpointEnabled();
            boolean isHovered = (index == hoveredRowIndex && isHoveringInGutter);
            
            // Format the text with event ID first
            String eventInfo = String.format("ID: %d - %s", 
                event.getEventId(), event.getHumanReadableEventType());
            
            // Add event type information if available
            if (event.isActivityEvent()) {
                eventInfo += " - Activity";
            } else if (event.isWorkflowExecutionEvent()) {
                eventInfo += " - Workflow";
            } else if (event.isTimerEvent()) {
                eventInfo += " - Timer";
            }
            
            // Add highlight indicator to text
            if (isHighlighted) {
                eventInfo = "▶ " + eventInfo + " ◀";
            }
            
            textLabel.setText(eventInfo);
            
            // Tooltip is now handled by the JList override
            
            // Set colors based on state
            if (isHighlighted) {
                setBackground(JBColor.namedColor("Debugger.currentExecutionPointBackground", 
                    new JBColor(0xFFFF8080, 0x4C4C0080))); // Theme-aware highlight
                textLabel.setForeground(UIUtil.getListForeground());
            } else if (isSelected) {
                setBackground(UIUtil.getListSelectionBackground(true));
                textLabel.setForeground(UIUtil.getListSelectionForeground(true));
            } else {
                setBackground(UIUtil.getListBackground());
                
                // Set text color based on event type using theme-aware colors
                if (event.isWorkflowExecutionEvent()) {
                    textLabel.setForeground(JBColor.namedColor("Component.infoForeground", 
                        new JBColor(0x0066CC, 0x5394EC))); // Blue
                } else if (event.isActivityEvent()) {
                    textLabel.setForeground(JBColor.namedColor("Component.validForeground", 
                        new JBColor(0x009933, 0x62C554))); // Green  
                } else if (event.isTimerEvent()) {
                    textLabel.setForeground(JBColor.namedColor("Component.warningForeground", 
                        new JBColor(0xCC6600, 0xE1A336))); // Orange
                } else {
                    textLabel.setForeground(UIUtil.getListForeground());
                }
            }
            
            setOpaque(true);
            
            // Store state for custom painting
            putClientProperty("hasBreakpoint", hasBreakpoint);
            putClientProperty("isHovered", isHovered);
            putClientProperty("eventIndex", index);
            
            return this;
        }
        
        @Override
        protected void paintComponent(Graphics g) {
            super.paintComponent(g);
            
            Graphics2D g2d = (Graphics2D) g.create();
            g2d.setRenderingHint(RenderingHints.KEY_ANTIALIASING, RenderingHints.VALUE_ANTIALIAS_ON);
            
            // Draw gutter background
            g2d.setColor(GUTTER_BACKGROUND);
            g2d.fillRect(0, 0, GUTTER_WIDTH, getHeight());
            
            // Draw gutter separator line
            g2d.setColor(JBColor.namedColor("Component.borderColor", UIUtil.getBoundsColor()));
            g2d.drawLine(GUTTER_WIDTH - 1, 0, GUTTER_WIDTH - 1, getHeight());
            
            // Get state from client properties
            Boolean hasBreakpoint = (Boolean) getClientProperty("hasBreakpoint");
            Boolean isHovered = (Boolean) getClientProperty("isHovered");
            
            if (hasBreakpoint != null && hasBreakpoint) {
                // Draw solid red breakpoint circle
                drawBreakpointIcon(g2d, BREAKPOINT_COLOR, true);
            } else if (isHovered != null && isHovered) {
                // Draw faded red circle on hover
                drawBreakpointIcon(g2d, HOVER_COLOR, false);
            }
            
            g2d.dispose();
        }
        
        private void drawBreakpointIcon(Graphics2D g2d, Color color, boolean filled) {
            int centerX = GUTTER_WIDTH / 2;
            int centerY = getHeight() / 2;
            int radius = ICON_SIZE / 2;
            
            g2d.setColor(color);
            
            if (filled) {
                // Draw filled circle for active breakpoint
                g2d.fillOval(centerX - radius, centerY - radius, ICON_SIZE, ICON_SIZE);
                
                // Add a subtle border
                g2d.setColor(color.darker());
                g2d.drawOval(centerX - radius, centerY - radius, ICON_SIZE, ICON_SIZE);
            } else {
                // Draw hollow circle for hover state
                g2d.setStroke(new BasicStroke(2.0f));
                g2d.drawOval(centerX - radius, centerY - radius, ICON_SIZE, ICON_SIZE);
            }
        }
        

    }
} 