import * as vscode from "vscode"
import os from "node:os"
import path from "node:path"
import http from "node:http"
import { historyFromJSON } from "@temporalio/common/lib/proto-utils"
import { temporal } from "@temporalio/proto"
import { Connection, LOCAL_TARGET } from "@temporalio/client"
import { Server } from "./server"
import { getBaseConfiguration, getCurrentLanguage } from "./get-base-configuration"

interface StartFromId {
  namespace?: string
  workflowId: string
  runId?: string
}

interface Settings {
  address: string
  tls: boolean
  clientCert?: Uint8Array
  clientPrivateKey?: Uint8Array
}

interface EncodedSettings {
  address: string
  tls: boolean
  base64ClientCert?: string
  base64ClientPrivateKey?: string
}

export class HistoryDebuggerPanel {
  protected static _instance?: Promise<HistoryDebuggerPanel>

  static async install(extensionUri: vscode.Uri, secretStorage: vscode.SecretStorage): Promise<void> {
    if (this._instance === undefined) {
      this._instance = Server.create().then((server: Server) => {
        console.log(`Server listening on ${server.url}`)
        return new this(extensionUri, secretStorage, server)
      })
    } else {
      const instance = await this._instance
      instance.show()
    }
  }

  static get instance(): Promise<HistoryDebuggerPanel> {
    if (this._instance === undefined) {
      throw new ReferenceError("HistoryDebuggerPanel not installed")
    }
    return this._instance
  }

  public currentHistoryBuffer?: Buffer

  public static readonly viewType = "temporal-debugger-plugin"

  private readonly panel: vscode.WebviewPanel
  private disposables: vscode.Disposable[] = []
  private updateWorkflowTaskHasBreakpoint = (_hasBreakpoint: boolean) => {
    // noop, to be set in the updateCurrentWFTStarted handler
  }

  show(): void {
    this.panel.reveal(vscode.ViewColumn.Beside)
  }

  async updateCurrentWFTStarted(eventId: number): Promise<void> {
    const p = new Promise<boolean>((resolve, reject) => {
      this.updateWorkflowTaskHasBreakpoint = resolve
      setTimeout(() => reject(new Error("Timed out waiting for response from webview")), 5000)
    })
    await this.panel.webview.postMessage({ type: "currentWFTUpdated", eventId })
    const hasBreakpoint = await p
    if (hasBreakpoint) {
      await vscode.commands.executeCommand("workbench.action.debug.pause")
    }
  }

  private constructor(
    protected readonly extensionUri: vscode.Uri,
    private readonly secretStorage: vscode.SecretStorage,
    protected readonly server: Server,
  ) {
    this.panel = vscode.window.createWebviewPanel(HistoryDebuggerPanel.viewType, "Temporal", vscode.ViewColumn.Beside, {
      // Enable javascript in the webview
      enableScripts: true,
      // And restrict the webview to only loading content from our extension's compiled directory.
      localResourceRoots: [vscode.Uri.joinPath(extensionUri, "webview/dist")],
      retainContextWhenHidden: true,
    })

    // Set the webview's initial html content
    this.update()

    let reloadServer: http.Server | undefined = undefined
    // Start a local HTTP server to automatically reload the webview when rollup completes.
    if (process.env.TEMPORAL_DEBUGGER_EXTENSION_DEV_MODE) {
      reloadServer = http.createServer((_req, res) => {
        void vscode.commands.executeCommand("workbench.action.webview.reloadWebviewAction")
        res.writeHead(200, "OK")
        res.end()
      })
      reloadServer.listen(55666, "127.0.0.1")
    }

    // Listen for when the panel is disposed
    // This happens when the user closes the panel or when the panel is closed programatically
    this.panel.onDidDispose(async () => {
      // Close local servers
      server.terminate()
      if (reloadServer) {
        reloadServer.close()
      }

      await this.dispose(), null, this.disposables
    })
  }

  public async dispose(): Promise<void> {
    // Clean up our resources
    this.panel.dispose()

    while (this.disposables.length) {
      const x = this.disposables.pop()
      if (x) {
        await x.dispose()
      }
    }

    delete HistoryDebuggerPanel._instance
  }

  private encodeSettings({ address, tls, clientCert, clientPrivateKey }: Settings): EncodedSettings {
    return {
      address,
      tls,
      base64ClientCert: clientCert ? Buffer.from(clientCert).toString("base64") : undefined,
      base64ClientPrivateKey: clientPrivateKey ? Buffer.from(clientPrivateKey).toString("base64") : undefined,
    }
  }

  private decodeSettings({ address, tls, base64ClientCert, base64ClientPrivateKey }: EncodedSettings): Settings {
    return {
      address,
      tls,
      clientCert: base64ClientCert ? Buffer.from(base64ClientCert, "base64") : undefined,
      clientPrivateKey: base64ClientPrivateKey ? Buffer.from(base64ClientPrivateKey, "base64") : undefined,
    }
  }

  private async getSettings(): Promise<EncodedSettings> {
    const secret = await this.secretStorage.get("settings")
    if (secret === undefined) {
      return {
        address: LOCAL_TARGET,
        tls: false,
      }
    }
    return JSON.parse(secret)
  }

  private async getConnection() {
    const encoded = await this.getSettings()
    const { address, tls, clientCert, clientPrivateKey } = this.decodeSettings(encoded)
    return await Connection.connect({
      address,
      tls:
        clientCert && clientPrivateKey
          ? { clientCertPair: { crt: Buffer.from(clientCert), key: Buffer.from(clientPrivateKey) } }
          : tls
            ? true
            : false,
    })
  }

  private async downloadHistory({ namespace, workflowId, runId }: StartFromId) {
    const connection = await this.getConnection()
    let nextPageToken: Uint8Array | undefined = undefined
    const history: temporal.api.history.v1.IHistory = { events: [] }
    do {
      try {
        const response: temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse =
          await connection.workflowService.getWorkflowExecutionHistory({
            namespace: namespace || "default",
            execution: {
              workflowId,
              runId,
            },
            nextPageToken,
          })
        if (!response.history?.events) {
          throw new Error("Empty history")
        }
        history.events?.push(...response.history.events)
        nextPageToken = response.nextPageToken
      } catch (err) {
        throw new Error(`Unable to find workflow execution history for ${workflowId}`)
      }
    } while (nextPageToken && nextPageToken.length > 0)
    return history
  }

  private update(): void {
    const { webview } = this.panel

    webview.html = this.getHtmlForWebview(webview)

    webview.onDidReceiveMessage(async (e): Promise<void> => {
      try {
        switch (e.type) {
          case "updateWorkflowTaskHasBreakpoint": {
            this.updateWorkflowTaskHasBreakpoint(e.hasBreakpoint)
            break
          }
          case "getSettings": {
            const settings = await this.getSettings()
            await this.panel.webview.postMessage({
              type: "settingsLoaded",
              settings: {
                address: settings.address,
                tls: settings.tls,
                hasClientCert: !!settings.base64ClientCert,
                hasClientPrivateKey: !!settings.base64ClientPrivateKey,
              },
            })
            break
          }
          case "updateSettings": {
            e.settings.address ??= LOCAL_TARGET
            e.settings.tls ??= false
            const encodedSettings = this.encodeSettings(e.settings)
            await this.secretStorage.store("settings", JSON.stringify(encodedSettings))
            await vscode.window.showInformationMessage("Settings updated")
            break
          }
          case "startFromId": {
            const history = await this.downloadHistory(e)
            await this.handleStartProject(history)
            break
          }
          case "startFromHistory": {
            const history = historyFromJSON(e.history)
            await this.handleStartProject(history)
            break
          }
        }
      } catch (err) {
        await vscode.window.showErrorMessage(`${err}`)
      }
    })
  }

  private async getReplayerEndpoint() {
    const config = vscode.workspace.getConfiguration("temporal")
    let replayerEntrypoint = config.get("replayerEntrypoint") as string
    const language = getCurrentLanguage()
    const workspace = vscode.workspace.workspaceFolders?.[0]
    const workspaceFolder = workspace?.uri

    // Debug logging
    console.log("Debug configuration:")
    console.log("- Language:", language)
    console.log("- Configured replayerEntrypoint:", replayerEntrypoint)
    console.log("- Workspace folder:", workspaceFolder?.fsPath)
    console.log("- All temporal config:", config)

    const configuredAbsolutePath = path.isAbsolute(replayerEntrypoint)

    // Provide language-specific defaults if not configured
    if (!replayerEntrypoint) {
      switch (language) {
        case "typescript":
          replayerEntrypoint = "src/debug-replayer.ts"
          break
        case "go":
          replayerEntrypoint = "." // Use current directory for Go package
          break
        case "java":
          replayerEntrypoint = "src/main/java/TemporalReplayer.java"
          break
        case "python":
          replayerEntrypoint = "replayer.py"
          break
        default:
          throw new Error(`No default replayer endpoint for language: ${language}`)
      }
    }

    if (!configuredAbsolutePath) {
      if (workspaceFolder === undefined) {
        throw new Error("temporal.replayerEndpoint not configured, cannot use default without a workspace folder")
      } else {
        replayerEntrypoint = vscode.Uri.joinPath(workspaceFolder, replayerEntrypoint).fsPath
      }
    }

    try {
      const stat: vscode.FileStat = await vscode.workspace.fs.stat(vscode.Uri.file(replayerEntrypoint))
      const { type } = stat
      if (type === vscode.FileType.Directory) {
        throw new Error(
          `Configured temporal.replayerEndpoint (${replayerEntrypoint}) is a folder, please provide a file instead`,
        )
      }
      if (type === vscode.FileType.Unknown) {
        throw new Error(
          `Configured temporal.replayerEndpoint (${replayerEntrypoint}) is of unknown type, please provide a file instead`,
        )
      }
    } catch (err: any) {
      if (err?.code === vscode.FileSystemError.FileNotFound.name) {
        if (!configuredAbsolutePath && (vscode.workspace.workspaceFolders?.length ?? 0) > 1) {
          throw new Error(
            `Configured temporal.replayerEndpoint (${replayerEntrypoint}) not found (multiple workspace folders found, consider using an absolute path to disambiguate)`,
          )
        }
        throw new Error(`Configured temporal.replayerEndpoint (${replayerEntrypoint}) not found`)
      }
      throw err
    }

    return replayerEntrypoint
  }

  private getLanguageRequirements(language: string): string {
    switch (language) {
      case "typescript":
        return "Make sure you have the TypeScript extension installed and ts-node available."
      case "go":
        return "Make sure you have the Go extension installed and delve debugger available."
      case "java":
        return "Make sure you have the Java extension pack installed."
      case "python":
        return "Make sure you have the Python extension installed."
      default:
        return "Unknown language requirements."
    }
  }

  /* eslint-disable @typescript-eslint/naming-convention */
  private async handleStartProject(history: temporal.api.history.v1.IHistory): Promise<void> {
    const bytes = new Uint8Array(temporal.api.history.v1.History.encode(history).finish())
    const buffer = Buffer.from(bytes)
    this.currentHistoryBuffer = buffer
    const workspace = vscode.workspace.workspaceFolders?.[0]
    const replayerEndpoint = await this.getReplayerEndpoint()
    const language = getCurrentLanguage()

    await this.panel.webview.postMessage({ type: "historyProcessed", history: bytes })
    // Make sure the panel is out of focus before starting a debug session, otherwise it will be replaced with an
    // editor
    if (vscode.window.tabGroups.all.length > 1) {
      await vscode.commands.executeCommand("workbench.action.focusFirstEditorGroup")
    } else {
      await vscode.commands.executeCommand("workbench.action.splitEditorLeft")
    }

    const baseConfig = await getBaseConfiguration()

    // Language-specific configuration
    let debugConfig: vscode.DebugConfiguration

    switch (language) {
      case "typescript":
        // TypeScript-specific configuration
        if (process.env.TEMPORAL_DEBUGGER_EXTENSION_DEV_MODE) {
          baseConfig.skipFiles?.push("${workspaceFolder}/packages/worker/src/**")
        }
        // NOTE: Adding NODE_PATH below in case ts-node is not an installed dependency in the workspace.
        const delim = os.platform() === "win32" ? ";" : ":"
        const pathPrefix = process.env.NODE_PATH ? `${process.env.NODE_PATH ?? ""}${delim}` : ""
        debugConfig = {
          ...baseConfig,
          args: [replayerEndpoint],
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
            NODE_PATH: `${pathPrefix}${path.join(__dirname, "../../node_modules")}`,
          },
        }
        break

      case "go":
        // For Go, ensure we have a valid program path
        let goProgram = replayerEndpoint
        if (goProgram.endsWith('.go')) {
          // If it's a .go file, use it directly
          goProgram = replayerEndpoint
        } else {
          // If it's a directory, use the directory path
          goProgram = path.dirname(replayerEndpoint)
        }
        console.log("Go program path:", goProgram)

        debugConfig = {
          ...baseConfig,
          program: goProgram,
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
          },
        }
        break

      case "java":
        debugConfig = {
          ...baseConfig,
          args: [replayerEndpoint],
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
          },
        }
        break

      case "python":
        debugConfig = {
          ...baseConfig,
          args: [replayerEndpoint],
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
          },
        }
        break

      default:
        throw new Error(`Unsupported language: ${language}`)
    }

    try {
      console.log("Final debug configuration:", JSON.stringify(debugConfig, null, 2))
      await vscode.debug.startDebugging(workspace, debugConfig)
      await vscode.window.showInformationMessage(`Starting ${language} debug session`)
    } catch (err) {
      const requirements = this.getLanguageRequirements(language)
      await vscode.window.showErrorMessage(
        `Failed to start ${language} debug session: ${err}\n\n${requirements}`,
      )
      throw err
    }
  }

  private getHtmlForWebview(webview: vscode.Webview): string {
    // And the uri we use to load this script in the webview
    const scriptUri = webview.asWebviewUri(vscode.Uri.joinPath(this.extensionUri, "webview", "dist", "app.js"))
    const styleUri = webview.asWebviewUri(vscode.Uri.joinPath(this.extensionUri, "webview", "dist", "app.css"))

    return `<!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <!--
          TODO: nonce was removed here because protobufjs uses code generation, see if we can bring it back.
          Use a content security policy to only allow scripts that have a specific nonce.
        -->
        <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${webview.cspSource} https:; script-src 'unsafe-inline' 'unsafe-eval' ${webview.cspSource}; style-src 'unsafe-inline' ${webview.cspSource};">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <link href="${styleUri}" rel="stylesheet">
      </head>
      <body>
      <script>
        // Set vscode global object
        const vscode = acquireVsCodeApi();
      </script>
      </body>
      <script src="${scriptUri}"></script>
      </html>`
  }
}
