import * as vscode from "vscode"
import { HistoryDebuggerPanel } from "./panel"
import { registerDebugAdapterTrackerFactory } from "./node-debug-tracker"

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  // TODO: remove after done with debugging js-debug
  const debugAdapterTrackerFactory: vscode.Disposable = registerDebugAdapterTrackerFactory()

  const openCommand: vscode.Disposable = vscode.commands.registerCommand(
    "temporal.debugger-extension.open-panel",
    async () => {
      await HistoryDebuggerPanel.install(context.extensionUri, context.secrets)
    },
  )
  context.subscriptions.push(openCommand)
  // TODO: remove after done with debugging js-debug
  context.subscriptions.push(openCommand, debugAdapterTrackerFactory)
}
