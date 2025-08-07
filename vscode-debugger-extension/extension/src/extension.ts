import * as vscode from "vscode"
import { HistoryDebuggerPanel } from "./panel"

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const openCommand: vscode.Disposable = vscode.commands.registerCommand(
    "temporal.debugger-extension.open-panel",
    async () => {
      await HistoryDebuggerPanel.install(context.extensionUri, context.secrets)
    },
  )
  context.subscriptions.push(openCommand)
}
