# User Guide

## Overview

The plugin integrates Temporal workflow debugging into GoLand.

## Prerequisites

- GoLand with Go plugin.
- Go 1.19+.
- tdlv binary.

## Installation

1. Clone repository.
2. Build plugin: `./gradlew buildPlugin` in jetbrains-plugin/.
3. Install in GoLand via Settings > Plugins > Install from disk.

## Usage

1. Open tool window: View > Tool Windows > Temporal Workflow Debugger.
2. Upload history: Click 'Upload Workflow History', select JSON file.
3. Configure: Set debug directory and tdlv path if needed.
4. Set breakpoints: Click gutter icons in history list.
5. Start debug: Click 'Run'.

## Features

- Event history view with breakpoints.
- Tooltips with event details.
- Integration with GoLand debugger via tdlv.

## Tips

- Use 'Clear All Breakpoints' to reset.
- View event categories and times in tooltips. 