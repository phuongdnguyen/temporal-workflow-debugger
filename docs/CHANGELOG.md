# Release Notes

This document tracks user-facing changes, new features, and important updates to the Temporal Workflow Debugger.

For detailed technical changes, see [Developer Changelog](./developer-changelog.md).

## [Unreleased] - 2025-01-15

### 🎯 Enhanced

#### Improved Step-Over Experience
**What's New**: Enhanced visual feedback when stepping over code that goes through adapter layers.

**Benefits**:
- **Better Visual Feedback**: When you click "Step Over", the debugger cursor now visibly moves to the next line
- **Intuitive Debugging**: Step-over operations feel more natural and provide clear visual confirmation
- **Seamless Experience**: Auto-stepping through internal code now provides better user feedback

**How It Works**: After automatically stepping through Temporal SDK code, the debugger takes one additional step forward in your workflow code (only for step-over commands) to ensure you can see the debugging action has taken effect.

### 🐛 Fixed

#### More Reliable Command Detection
**What Changed**: Fixed issues where continue vs step-over commands were sometimes misidentified.

**Benefits**:
- **Accurate Behavior**: Continue commands now correctly stop at breakpoint locations without extra stepping
- **Consistent Experience**: Step-over commands consistently provide visual feedback
- **Better Auto-stepping**: Improved detection of user intent for better auto-stepping behavior

#### Enhanced Cross-Platform Support
**What Changed**: Improved JSON-RPC message parsing for better compatibility across different operating systems.

**Benefits**:
- **Windows Compatibility**: Better handling of line endings on Windows systems
- **Robust Communication**: More reliable message parsing between IDE and debugger
- **Fewer Connection Issues**: Reduced parsing errors that could cause debugging sessions to fail

---

## [v0.9.0] - 2024-12-20

### 🚀 New Features

#### VS Code Debug Adapter Protocol (DAP) Support
**What's New**: Full support for debugging in VS Code using the Debug Adapter Protocol.

**Benefits**:
- **VS Code Compatibility**: Native debugging support for VS Code users
- **Multi-IDE Support**: Works with both GoLand/IntelliJ IDEA and VS Code
- **Consistent Experience**: Same debugging features across different IDEs

**Setup**: Add the provided `launch.json` configuration to your VS Code workspace.

#### Smart Stack Trace Filtering
**What's New**: Automatically hides internal Temporal SDK and adapter code from call stacks.

**Benefits**:
- **Clean Call Stacks**: See only your workflow code and relevant runtime frames
- **Reduced Confusion**: No more seeing internal `replayer.go` or SDK files in stack traces
- **Better Focus**: Concentrate on debugging your workflow logic, not internal implementation

#### Automatic Variable Evaluation
**What's New**: Variable inspection works correctly even with filtered stack traces.

**Benefits**:
- **Accurate Variable Values**: Hover over variables to see correct values
- **Working Watch Expressions**: Add variables to watch lists without issues
- **Proper Frame Context**: Variable evaluation uses the correct stack frame context

### 🎯 Enhanced

#### Auto-stepping Through Adapter Code
**What's New**: Debugger automatically steps through internal Temporal code to reach your workflow logic.

**Benefits**:
- **Seamless Debugging**: No manual stepping through internal code
- **Time Saving**: Automatically skips irrelevant code sections
- **Better Focus**: Stops only at your workflow code and important runtime points

#### Multi-Protocol Architecture
**What's New**: Single proxy supports both JSON-RPC (GoLand) and DAP (VS Code) protocols.

**Benefits**:
- **Unified Experience**: Same features regardless of IDE choice
- **Easy Switching**: Use different IDEs for the same project without reconfiguration
- **Future-Proof**: Easy to add support for additional IDEs and protocols

---

## [v0.8.0] - 2024-11-15

### 🚀 New Features

#### JetBrains Plugin
**What's New**: Enhanced IDE integration for GoLand and IntelliJ IDEA users.

**Benefits**:
- **Simplified Setup**: Automatic configuration of debug settings
- **Temporal Actions**: IDE actions specific to workflow debugging
- **Enhanced UI**: Better integration with JetBrains IDE features

**Installation**: Build and install the plugin from the `jetbrains-plugin` directory.

#### Transparent Debugging Proxy
**What's New**: Intelligent proxy that sits between your IDE and the Delve debugger.

**Benefits**:
- **Non-Invasive**: Works with unmodified Delve and IDEs
- **Protocol Transparency**: No changes needed to existing debugging workflows
- **Full Compatibility**: Supports all standard debugging operations

### 🎯 Enhanced

#### Improved Connection Stability
**What's New**: More robust handling of debugging connections and protocol communication.

**Benefits**:
- **Fewer Disconnections**: More stable debugging sessions
- **Better Error Recovery**: Graceful handling of connection issues
- **Reliable Communication**: Improved message parsing and protocol handling

#### Frame Context Preservation
**What's New**: Maintains accurate frame numbering for variable evaluation with filtered stacks.

**Benefits**:
- **Working Variable Inspection**: Hover and watch expressions work correctly
- **Accurate Debugging**: Stack frame operations use correct context
- **Seamless Experience**: No difference from normal Go debugging

---

## [v0.7.0] - 2024-10-01

### 🚀 New Features

#### Initial Release
**What's New**: First public release of the Temporal Workflow Debugger.

**Features**:
- **Workflow Debugging**: Set breakpoints and step through Temporal workflow code
- **GoLand Support**: Native integration with GoLand and IntelliJ IDEA
- **Stack Filtering**: Hide internal adapter implementation details
- **Variable Inspection**: Examine workflow variables and context

## 🔄 Upgrade Guide

### From v0.8.x to v0.9.x

**VS Code Users**:
1. Add the new `launch.json` configuration for DAP support
2. Install the Go extension if not already installed
3. Use F5 to start debugging sessions

**GoLand Users**:
- No changes required, existing configurations continue to work
- Optionally install the updated JetBrains plugin for enhanced features

**Breaking Changes**: None

### From v0.7.x to v0.8.x

**All Users**:
1. Update to the latest version: `git pull origin main`
2. Rebuild the delve wrapper: `cd custom-debugger && go build`
3. Restart your debugging sessions

**JetBrains Plugin Users**:
1. Build and install the new plugin: `cd jetbrains-plugin && ./gradlew buildPlugin`
2. Install from `build/distributions/` through IDE plugin manager

**Breaking Changes**: None

## 🐛 Known Issues

### Current Limitations

- **Single Workflow Focus**: Currently optimized for single workflow debugging
- **Path Dependencies**: Stack filtering assumes workflow code is in `my-wf/` directory
- **Protocol Overhead**: Small performance impact due to proxy layer

### Workarounds

- **Custom Workflow Paths**: Modify filtering rules in delve wrapper for different project structures
- **Performance Optimization**: Disable verbose logging for better performance in production debugging

## 🔮 Planned Features

### Upcoming Releases

- **Multi-Workflow Support**: Debug multiple workflows simultaneously
- **Custom Filtering Rules**: User-configurable stack filtering patterns
- **History Integration**: Debug workflow replays with full history context
- **Enhanced Visualization**: Real-time workflow state visualization
- **Cloud Integration**: Integration with Temporal Cloud debugging tools

### Community Requests

- **Docker Support**: Enhanced debugging in containerized environments
- **Language Extensions**: Support for other Temporal SDK languages
- **Testing Integration**: Integration with workflow testing frameworks

## 📞 Support

- **Bug Reports**: [GitHub Issues](https://github.com/temporalio/temporal-goland-plugin/issues)
- **Feature Requests**: [GitHub Discussions](https://github.com/temporalio/temporal-goland-plugin/discussions)
- **Documentation**: [User Guide](./user-guide.md)
- **Community**: [Temporal Community](https://community.temporal.io/) 