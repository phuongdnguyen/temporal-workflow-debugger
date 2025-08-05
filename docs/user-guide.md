# User Guide

## Overview
There are 2 methods to use the debugger: standalone & ide integrated

## Tested language version
- Go 1.19+.
- NodeJS v22.17.0, Npm 10.9.2
- Python 3.12.11


## Installation
For both methods:
1. Install tdlv from [Github Release](https://github.com/phuongdnguyen/temporal-workflow-debugger/releases).
For Goland Plugin users:
1. Install the plugin from [jetbrains marketplace](https://plugins.jetbrains.com/search?excludeTags=internal&products=androidstudio&products=aqua&products=clion&products=dataspell&products=dbe&products=fleet&products=go&products=idea&products=idea_ce&products=mps&products=phpstorm&products=pycharm&products=rider&products=ruby&products=rust&products=webstorm&products=writerside&search=Temporal%20workflow%20debugger)

## Usage
For standalone mode users:
You will need a remote debugging configuration. We have examples provided in [example](../example)

For Goland Plugin users:
1. Open tool window: View > Tool Windows > Temporal Workflow Debugger.
2. Upload history: Click 'Upload Workflow History', select JSON file.
3. Configure: Set debug directory and tdlv path if needed.
4. Set breakpoints: Click gutter icons in history list.
5. Start debug: Click 'Run'.
