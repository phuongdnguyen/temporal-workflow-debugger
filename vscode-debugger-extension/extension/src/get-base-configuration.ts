import * as vscode from "vscode"
import { supportsESM } from "./is-esm"

type SupportedLanguage = "typescript" | "go" | "java" | "python"

const nodeConfiguration = {
  name: "Launch Program",
  type: "node",
  request: "launch",
  runtimeExecutable: "node",
  skipFiles: [
    "<node_internals>/**",
    "**/node_modules/@temporalio/worker/src/**",
    "**/node_modules/@temporalio/worker/lib/**",
    "**/node_modules/@temporalio/common/src/**",
    "**/node_modules/@temporalio/common/lib/**",
    "**/node_modules/**/source-map/**",
  ],
  env: {
    JS_DEBUG_USE_LOCAL_DAP_PORT: 60000,
  },
  internalConsoleOptions: "openOnSessionStart",
  pauseForSourceMap: true,
} satisfies vscode.DebugConfiguration

// const goConfiguration = {
//   name: "Launch Program",
//   type: "go",
//   request: "attach",
//   mode: "remote",
//   port: 60000,
//   host: "127.0.0.1",
//   apiVersion: 2,
//   showLog: true,
//   internalConsoleOptions: "openOnSessionStart",
// } satisfies vscode.DebugConfiguration

const goConfiguration = {
  name: "Launch Program",
  type: "go",
  request: "attach",
  mode: "remote",
  port: 60000,
  host: "127.0.0.1",
  apiVersion: 2,
  showLog: true,
  internalConsoleOptions: "openOnSessionStart",
} satisfies vscode.DebugConfiguration

const javaConfiguration = {
  name: "Launch Program",
  type: "java",
  request: "launch",
  mainClass: "${workspaceFolder}",
  internalConsoleOptions: "openOnSessionStart",
} satisfies vscode.DebugConfiguration

// const pythonConfiguration = {
//   name: "Launch Program",
//   type: "debugpy",
//   request: "attach",
//   connect: {
//     host: "localhost",
//     port: 60000,
//   },
//   console: "integratedTerminal",
//   internalConsoleOptions: "openOnSessionStart",
// } satisfies vscode.DebugConfiguration

const pythonConfiguration = {
  name: "Launch Program",
  type: "debugpy",
  request: "launch",
  connect: {
    host: "localhost",
    port: 60000,
  },
  console: "integratedTerminal",
  internalConsoleOptions: "openOnSessionStart",
} satisfies vscode.DebugConfiguration

const getLanguageFromConfig = (): SupportedLanguage => {
  const config = vscode.workspace.getConfiguration("temporal")
  const language = config.get("debugLanguage") as SupportedLanguage
  return language || "typescript" // Default to TypeScript for backward compatibility
}

export const getBaseConfiguration = async (): Promise<vscode.DebugConfiguration> => {
  const language = getLanguageFromConfig()

  switch (language) {
    case "typescript":
      // Set the environment variable to force VS Code's JavaScript debugger to use local DAP port
      process.env.JS_DEBUG_USE_LOCAL_DAP_PORT = "60000"
      const runtimeArgs = (await supportsESM())
        ? ["--loader=ts-node/esm"]
        : ["--nolazy", "-r", "ts-node/register/transpile-only"]
      return { ...nodeConfiguration, runtimeArgs }

    case "go":
      return { ...goConfiguration }

    case "java":
      return { ...javaConfiguration }

    case "python":
      return { ...pythonConfiguration }

    default:
      throw new Error(`Unsupported language: ${language}`)
  }
}

export const getCurrentLanguage = (): SupportedLanguage => {
  return getLanguageFromConfig()
}
