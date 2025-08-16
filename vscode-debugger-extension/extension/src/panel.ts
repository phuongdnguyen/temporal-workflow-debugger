import * as vscode from "vscode"
import * as os from "node:os"
import * as path from "node:path"
import * as http from "node:http"
import { spawn, ChildProcess } from "node:child_process"
import { historyFromJSON } from "@temporalio/common/lib/proto-utils"
import { temporal } from "@temporalio/proto"
import { Connection, LOCAL_TARGET } from "@temporalio/client"
import { Server } from "./server"
import { getBaseConfiguration, getCurrentLanguage, getDependencies } from "./get-base-configuration"
import which from "which"
import net from "node:net"

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

  // stores the binary-encoded representation of a temporal.api.history.v1.History protobuf message
  public currentHistoryBuffer?: Buffer
  private enabledBreakpoints: Set<number> = new Set()
  private debuggerProcess?: ChildProcess // Background process to run alongside debugging
  private debugSessionDisposables: vscode.Disposable[] = [] // Debug session event listeners

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
    // If hasBreakpoint, the event will be highlighted in the history panel
    // No need to pause the VSCode debugger - just let the highlighting show the breakpoint hit
  }

  // Breakpoint management methods (compatible with Java extension)
  public getEnabledBreakpoints(): number[] {
    return Array.from(this.enabledBreakpoints)
  }

  public toggleBreakpoint(eventId: number): boolean {
    if (this.enabledBreakpoints.has(eventId)) {
      this.enabledBreakpoints.delete(eventId)
      return false
    } else {
      this.enabledBreakpoints.add(eventId)
      return true
    }
  }

  public addBreakpoint(eventId: number): void {
    this.enabledBreakpoints.add(eventId)
  }

  public removeBreakpoint(eventId: number): void {
    this.enabledBreakpoints.delete(eventId)
  }

  public clearAllBreakpoints(): void {
    this.enabledBreakpoints.clear()
  }

  public async syncBreakpointsToWebview(): Promise<void> {
    await this.panel.webview.postMessage({
      type: "breakpointsUpdated",
      breakpoints: Array.from(this.enabledBreakpoints),
    })
  }

  /**
   * Checks if a background process is currently running.
   */
  public isDebuggerProcessRunning(): boolean {
    return this.debuggerProcess !== undefined && !this.debuggerProcess.killed
  }

  /**
   * Starts a debugger process before debugging begins.
   * The process will be terminated when the debug session ends.
   * @param command The command to run (e.g., "npm", "go", "python")
   * @param args Array of arguments for the command
   * @param options Optional spawn options (cwd, env, etc.)
   */
  private async startDebugger(command: string, args: string[] = [], options: any = {}): Promise<void> {
    // Terminate any existing background process
    await this.terminateDebugger()

    try {
      console.log(`Starting debugger process: ${command} ${args.join(" ")}`)
      console.log(`Debugger process cwd: ${vscode.workspace.workspaceFolders?.[0]?.uri.fsPath}`)
      vscode.window.showInformationMessage(
        "Starting the debugging process. If this is the first time, you will be prompted to allow installation of dependencies.",
      )

      this.debuggerProcess = spawn(command, args, {
        cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
        env: { ...process.env, ...options.env },
        stdio: ["ignore", "pipe", "pipe"], // Capture stdout and stderr
        ...options,
      })

      // Log output from the background process
      this.debuggerProcess.stdout?.on("data", (data) => {
        console.log(`Debugger process stdout: ${data.toString()}`)
      })

      this.debuggerProcess.stderr?.on("data", (data) => {
        console.log(`Debugger process stderr: ${data.toString()}`)
      })

      this.debuggerProcess.on("error", (error) => {
        console.error(`Debugger process error: ${error.message}`)
        vscode.window.showErrorMessage(`Debugger process failed: ${error.message}`)
      })

      this.debuggerProcess.on("exit", (code, signal) => {
        console.log(`Debugger process exited with code ${code}, signal ${signal}`)
        this.debuggerProcess = undefined
      })

      // Give the process a moment to start
      let attemp = 0
      let startingUpNotificationShown = false
      while (attemp < 10) {
        if (attemp > 0) {
          switch (getCurrentLanguage()) {
            case "go":
              if (startingUpNotificationShown) {
                break
              }
              vscode.window.showInformationMessage("Waiting for debugger to start")
              startingUpNotificationShown = true
              break
            case "python":
              if (startingUpNotificationShown) {
                break
              }
              vscode.window.showInformationMessage("Waiting for debugger to start")
              startingUpNotificationShown = true
              break
            case "typescript":
              if (startingUpNotificationShown) {
                break
              }
              vscode.window.showInformationMessage("Waiting for debugger to start")
              startingUpNotificationShown = true
              break
          }
        }
        await new Promise((resolve) => setTimeout(resolve, 2000))
        if (await this.isPortListening(60000)) {
          vscode.window.showInformationMessage("Debugger started")
          break
        }
        attemp++
      }

      if (this.debuggerProcess?.killed) {
        throw new Error("Debugger process failed to start")
      }

      console.log(`Debugger process started successfully (PID: ${this.debuggerProcess?.pid})`)
    } catch (error) {
      console.error(`Failed to start debugger process: ${error}`)
      throw error
    }
  }

  private async runChildProcess(command: string, args: string[] = [], options: any = {}): Promise<boolean> {
    try {
      console.log(`Starting child process: ${command} ${args.join(" ")}`)
      console.log(`child process cwd: ${vscode.workspace.workspaceFolders?.[0]?.uri.fsPath}`)

      let childProcess = spawn(command, args, {
        cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
        env: { ...process.env, ...options.env },
        stdio: ["ignore", "pipe", "pipe"], // Capture stdout and stderr
        ...options,
      })

      // Log output from the background process
      childProcess.stdout?.on("data", (data) => {
        console.log(`child process stdout: ${data.toString()}`)
      })

      childProcess.stderr?.on("data", (data) => {
        console.log(`child process stderr: ${data.toString()}`)
      })

      childProcess.on("error", (error) => {
        console.error(`child process error: ${error.message}`)
        vscode.window.showErrorMessage(`child process failed: ${error.message}`)
      })
      const timeoutMs = 10000
      const { timedOut, code } = await this.waitForProcessExit(childProcess, timeoutMs)

      if (timedOut) {
        console.log(`debuggerDepCheck timed out after ${timeoutMs}ms; killing process`)
        try {
          childProcess.kill("SIGKILL")
        } catch {}
        return false
      }

      return code === 0
    } catch (error) {
      console.error(`Failed to start child process: ${error}`)
      throw error
    }
  }

  private waitForProcessExit(
    child: ChildProcess,
    timeoutMs: number,
  ): Promise<{ timedOut: boolean; code: number | null; signal: NodeJS.Signals | null }> {
    return new Promise((resolve) => {
      const onExit = (code: number | null, signal: NodeJS.Signals | null) => {
        cleanup()
        resolve({ timedOut: false, code, signal })
      }
      const onError = () => {
        cleanup()
        resolve({ timedOut: false, code: 1, signal: null })
      }
      const cleanup = () => {
        clearTimeout(timer)
        child.off("exit", onExit)
        child.off("error", onError)
      }
      const timer = setTimeout(() => {
        cleanup()
        resolve({ timedOut: true, code: null, signal: null })
      }, timeoutMs)

      child.once("exit", onExit)
      child.once("error", onError)
    })
  }

  private async isPortListening(port: number, host = "127.0.0.1", timeoutMs = 1000): Promise<boolean> {
    return new Promise((resolve) => {
      const socket = net.createConnection({ port, host })
      const finish = (ok: boolean) => {
        socket.destroy()
        resolve(ok)
      }
      socket.setTimeout(timeoutMs)
      socket.once("connect", () => finish(true))
      socket.once("timeout", () => finish(false))
      socket.once("error", () => finish(false))
    })
  }

  /**
   * Terminates the debugger process if it's running
   */
  private async terminateDebugger(): Promise<void> {
    if (this.debuggerProcess && !this.debuggerProcess.killed) {
      console.log(`Terminating debugger process (PID: ${this.debuggerProcess.pid})`)

      try {
        // Try graceful termination first
        this.debuggerProcess.kill("SIGTERM")

        // Wait up to 5 seconds for graceful termination
        await new Promise<void>((resolve) => {
          const timeout = setTimeout(() => {
            if (this.debuggerProcess && !this.debuggerProcess.killed) {
              console.log("Forcefully killing background process")
              this.debuggerProcess.kill("SIGKILL")
            }
            resolve()
          }, 5000)

          this.debuggerProcess?.on("exit", () => {
            clearTimeout(timeout)
            resolve()
          })
        })
      } catch (error) {
        console.error(`Error terminating background process: ${error}`)
      }

      this.debuggerProcess = undefined
    }
  }

  /**
   * Sets up debug session event listeners to manage background process lifecycle
   */
  private setupDebugSessionListeners(): void {
    // Listen for debug session termination
    const onDidTerminateDebugSession = vscode.debug.onDidTerminateDebugSession(async (session) => {
      // Check if this is our debug session by looking at the configuration
      if (session.configuration && session.configuration.env?.TEMPORAL_DEBUGGER_PLUGIN_URL === this.server.url) {
        console.log("Debug session terminated, cleaning up background process")
        await this.terminateDebugger()
      }
    })

    // Listen for debug session start (for logging purposes)
    const onDidStartDebugSession = vscode.debug.onDidStartDebugSession(async (session) => {
      if (session.configuration && session.configuration.env?.TEMPORAL_DEBUGGER_PLUGIN_URL === this.server.url) {
        console.log("Debug session started successfully")
      }
    })

    // Store the disposables so we can clean them up later
    this.debugSessionDisposables.push(onDidTerminateDebugSession, onDidStartDebugSession)
  }

  /**
   * Gets the tdlv process configuration from VS Code settings
   */
  private async getTdlvConfigs(): Promise<{ command?: string; args?: string[]; options?: any }[]> {
    const language = getCurrentLanguage()
    const tdlv = this.resolveOnPath("tdlv")
    if (tdlv === "") {
      throw new Error("Please install tdlv first")
    }
    const baseArgs = ["--install"]
    switch (language) {
      case "python":
        const entryPoint = await this.getReplayerEntrypoint()
        return [
          {
            command: tdlv,
            args: ["--lang=python", `--entrypoint=${entryPoint}`],
          },
          {
            command: tdlv,
            args: ["--lang=python", `--entrypoint=${entryPoint}`].concat(baseArgs),
          },
          {
            command: tdlv,
            args: ["--lang=python", `--entrypoint=${entryPoint}`, "--start=true"].concat(baseArgs),
          },
        ]
      case "typescript":
        return [
          {
            command: tdlv,
            args: ["--lang=js"],
          },
          {
            command: tdlv,
            args: ["--lang=js"].concat(baseArgs),
          },
          {
            command: tdlv,
            args: ["--lang=js", "--start=true"].concat(baseArgs),
          },
        ]
      default:
        return [
          {
            command: tdlv,
            args: ["--lang=go"],
          },
          {
            command: tdlv,
            args: ["--lang=go"].concat(baseArgs),
          },
          {
            command: tdlv,
            args: ["--lang=go", "--start=true"].concat(baseArgs),
          },
        ]
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

    // Set up debug session event listeners
    this.setupDebugSessionListeners()

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
    // Terminate any running background process
    await this.terminateDebugger()

    // Clean up debug session listeners
    while (this.debugSessionDisposables.length) {
      const disposable = this.debugSessionDisposables.pop()
      if (disposable) {
        disposable.dispose()
      }
    }

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
        console.log(`webview receive a message of type: ${e.type}`)
        switch (e.type) {
          case "updateWorkflowTaskHasBreakpoint": {
            this.updateWorkflowTaskHasBreakpoint(e.hasBreakpoint)
            // Also update our breakpoint storage if eventId is provided
            if (e.eventId !== undefined) {
              if (e.hasBreakpoint) {
                this.addBreakpoint(e.eventId)
              } else {
                this.removeBreakpoint(e.eventId)
              }
            }
            break
          }
          case "toggleBreakpoint": {
            if (e.eventId !== undefined) {
              const isEnabled = this.toggleBreakpoint(e.eventId)
              console.log(`Breakpoint ${isEnabled ? "enabled" : "disabled"} for event ${e.eventId}`)
              // Sync updated breakpoints back to webview
              await this.syncBreakpointsToWebview()
            }
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
          case "loadHistoryFromId": {
            const history = await this.downloadHistory(e)
            await this.handleLoadHistoryOnly(history)
            break
          }
          case "loadHistoryFromFile": {
            const history = historyFromJSON(e.history)
            await this.handleLoadHistoryOnly(history)
            break
          }
          case "startDebugSession": {
            if (!this.currentHistoryBuffer) {
              throw new Error("No history loaded. Please load history first.")
            }
            const history = temporal.api.history.v1.History.decode(this.currentHistoryBuffer)
            console.log(`temporal.api.history.v1.History.decode(this.currentHistoryBuffer): ${history}`)
            // Convert the stored JSON back to protobuf format for handleStartProject
            await this.handleStartProject(history)
            break
          }
        }
      } catch (err) {
        await vscode.window.showErrorMessage(`${err}`)
      }
    })
  }

  private async getReplayerEntrypoint() {
    const config = vscode.workspace.getConfiguration("temporal")
    let replayerEntryPoint = config.get("replayerEntryPoint") as string
    const language = getCurrentLanguage()
    const workspace = vscode.workspace.workspaceFolders?.[0]
    const workspaceFolder = workspace?.uri

    // Debug logging
    console.log("Debug configuration:")
    console.log("- Language:", language)
    console.log("- Workspace folder:", workspaceFolder?.fsPath)
    console.log("- All temporal config:", config)

    const configuredAbsolutePath = path.isAbsolute(replayerEntryPoint)

    // Provide language-specific defaults if not configured
    if (!replayerEntryPoint) {
      switch (language) {
        case "typescript":
          replayerEntryPoint = "src/debug-replayer.ts"
          break
        default:
          throw new Error(`No default replayer endpoint for language: ${language}`)
      }
    }

    if (!configuredAbsolutePath) {
      if (workspaceFolder === undefined) {
        throw new Error("temporal.replayerEndpoint not configured, cannot use default without a workspace folder")
      } else {
        replayerEntryPoint = vscode.Uri.joinPath(workspaceFolder, replayerEntryPoint).fsPath
      }
    }

    try {
      const stat: vscode.FileStat = await vscode.workspace.fs.stat(vscode.Uri.file(replayerEntryPoint))
      const { type } = stat
      if (type === vscode.FileType.Directory) {
        throw new Error(
          `Configured temporal.replayerEndpoint (${replayerEntryPoint}) is a folder, please provide a file instead`,
        )
      }
      if (type === vscode.FileType.Unknown) {
        throw new Error(
          `Configured temporal.replayerEndpoint (${replayerEntryPoint}) is of unknown type, please provide a file instead`,
        )
      }
    } catch (err: any) {
      if (err?.code === vscode.FileSystemError.FileNotFound.name) {
        if (!configuredAbsolutePath && (vscode.workspace.workspaceFolders?.length ?? 0) > 1) {
          throw new Error(
            `Configured temporal.replayerEndpoint (${replayerEntryPoint}) not found (multiple workspace folders found, consider using an absolute path to disambiguate)`,
          )
        }
        throw new Error(`Configured temporal.replayerEndpoint (${replayerEntryPoint}) not found`)
      }
      throw err
    }

    return replayerEntryPoint
  }

  private getLanguageRequirements(language: string): string {
    switch (language) {
      case "typescript":
        return "Make sure you have the TypeScript extension installed and ts-node available."
      case "go":
        return "Make sure you have the Go extension installed and delve debugger available."
      case "python":
        return "Make sure you have the Python extension installed."
      default:
        return "Unknown language requirements."
    }
  }

  /* eslint-disable @typescript-eslint/naming-convention */
  private async handleStartProject(history: temporal.api.history.v1.IHistory): Promise<void> {
    // // Still send protobuf bytes to webview for UI processing
    // const bytes = new Uint8Array(temporal.api.history.v1.History.encode(history).finish())
    const workspace = vscode.workspace.workspaceFolders?.[0]
    const language = getCurrentLanguage()

    // await this.panel.webview.postMessage({ type: "historyProcessed", history: bytes })
    // // Sync current breakpoints to webview
    // await this.syncBreakpointsToWebview()
    // Make sure the panel is out of focus before starting a debug session, otherwise it will be replaced with an
    // editor
    if (vscode.window.tabGroups.all.length > 1) {
      await vscode.commands.executeCommand("workbench.action.focusFirstEditorGroup")
    } else {
      await vscode.commands.executeCommand("workbench.action.splitEditorLeft")
    }

    const baseConfig = await getBaseConfiguration()

    // Start debugger process
    const tdlvConfigs = await this.getTdlvConfigs()
    console.log(`debuggerConfig: ${JSON.stringify(tdlvConfigs)}`)
    const dependenciesCheckConfig = tdlvConfigs[0]
    const dependenciesInstallationConfig = tdlvConfigs[1]
    const startDebuggerConfig = tdlvConfigs[2]

    // Check for dependencies installed
    if (dependenciesCheckConfig.command) {
      try {
        const dependenciesCheckSuccess = await this.runChildProcess(
          dependenciesCheckConfig.command,
          dependenciesCheckConfig.args,
          dependenciesCheckConfig.options,
        )
        if (!dependenciesCheckSuccess) {
          const shouldInstall = await this.showConfirmModal()
          if (shouldInstall) {
            if (dependenciesInstallationConfig.command) {
              vscode.window.showInformationMessage("Installing required dependencies")
              const dependenciesInstalled = await this.runChildProcess(
                dependenciesInstallationConfig.command,
                dependenciesInstallationConfig.args,
                dependenciesInstallationConfig.options,
              )
              if (!dependenciesInstalled) {
                throw new Error(`Installing ${getDependencies(language)} failed, try to install it manually instead`)
              } else {
                vscode.window.showInformationMessage("Dependencies installed")
              }
            }
          } else {
            console.log("User refused to install dependencies, exit now.")
            return
          }
        }
      } catch (error) {
        console.error("Failed to check dependencies for debugger:", error)
        // Show error and stop the debug session
        throw error
      }
    }
    // Start the debugger itself
    if (startDebuggerConfig.command) {
      try {
        await this.startDebugger(startDebuggerConfig.command, startDebuggerConfig.args, startDebuggerConfig.options)
        console.log("Debugger process started successfully")
      } catch (error) {
        console.error("Failed to start debugger process:", error)
        // Show error and stop the debug session
        await vscode.window.showErrorMessage(`Failed to start background process: ${error}. Debugging will exit.`)
        throw error
      }
    }

    // Language-specific configuration
    let debugConfig: vscode.DebugConfiguration
    const replayerEntrypoint = await this.getReplayerEntrypoint()

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
          args: [replayerEntrypoint],
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
            NODE_PATH: `${pathPrefix}${path.join(__dirname, "../../node_modules")}`,
          },
        }
        break

      case "go":
        debugConfig = {
          ...baseConfig,
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
          },
          program: replayerEntrypoint,
        }
        break

      case "python":
        debugConfig = {
          ...baseConfig,
          env: {
            TEMPORAL_DEBUGGER_PLUGIN_URL: this.server.url,
          },
          program: replayerEntrypoint,
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
      await vscode.window.showErrorMessage(`Failed to start ${language} debug session: ${err}\n\n${requirements}`)
      throw err
    }
  }

  private async showConfirmModal(): Promise<boolean> {
    const yes: vscode.MessageItem = { title: "Yes" }
    const no: vscode.MessageItem = { title: "No", isCloseAffordance: true } // Esc = No

    const picked = await vscode.window.showInformationMessage("Install missing dependencies?", { modal: true }, yes, no)

    return picked == yes
  }

  private async handleLoadHistoryOnly(history: temporal.api.history.v1.IHistory): Promise<void> {
    const bytes = new Uint8Array(temporal.api.history.v1.History.encode(history).finish())
    const buffer = Buffer.from(bytes)
    this.currentHistoryBuffer = buffer

    // Send protobuf bytes to webview for UI processing
    await this.panel.webview.postMessage({ type: "historyProcessed", history: bytes })
    // Sync current breakpoints to webview
    await this.syncBreakpointsToWebview()
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

  private resolveOnPath(command: string, env?: NodeJS.ProcessEnv): string {
    const resolved = which.sync(command, { nothrow: true, path: env?.PATH ?? process.env.PATH })
    if (!resolved) {
      return ""
    }
    return resolved
  }
}
