import * as vscode from "vscode"
import { supportsESM } from "./is-esm"

const getLanguageFromConfig = (): SupportedLanguage => {
  const config = vscode.workspace.getConfiguration("temporal")
  const language = config.get("debugLanguage") as SupportedLanguage
  return language || "typescript" // Default to TypeScript for backward compatibility
}

const getExecutableFromConfig = (): string => {
  const lang = getLanguageFromConfig()
  console.log(`getExecutableFromConfig, lang: ${lang}`)
  console.log(
    `vscode.workspace.getConfiguration("temporal").get(lang) as string: ${vscode.workspace.getConfiguration("temporal").get(lang) as string}`,
  )
  return vscode.workspace.getConfiguration("temporal").get(lang) as string
}

type SupportedLanguage = "typescript" | "go" | "python"

export function getDependencies(lang: SupportedLanguage): string {
  switch (lang) {
    case "go":
      return "delve"
    case "python":
      return "debugpy"
    case "typescript":
      return "vscode-js-debug"
    default:
      return ""
  }
}

const nodeConfiguration = {
  name: "Launch Program",
  type: "node",
  request: "launch",
  skipFiles: [
    "<node_internals>/**",
    "**/node_modules/@temporalio/worker/src/**",
    "**/node_modules/@temporalio/worker/lib/**",
    "**/node_modules/@temporalio/common/src/**",
    "**/node_modules/@temporalio/common/lib/**",
    "**/node_modules/**/source-map/**",
  ],
  internalConsoleOptions: "openOnSessionStart",
  pauseForSourceMap: true,
} satisfies vscode.DebugConfiguration

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

const pythonConfiguration = {
  name: "Launch Program",
  type: "debugpy",
  request: "launch",
  // python: getExecutableFromConfig() || "python",
  connect: {
    host: "localhost",
    port: 60000,
  },
  console: "integratedTerminal",
  internalConsoleOptions: "openOnSessionStart",
} satisfies vscode.DebugConfiguration

export const getBaseConfiguration = async (): Promise<vscode.DebugConfiguration> => {
  const language = getLanguageFromConfig()

  switch (language) {
    case "typescript":
      // Set the environment variable to force VS Code's JavaScript debugger to use local DAP port
      process.env.JS_DEBUG_USE_LOCAL_DAP_PORT = "60000"
      const runtimeArgs = (await supportsESM())
        ? ["--loader=ts-node/esm"]
        : ["--nolazy", "-r", "ts-node/register/transpile-only"]
      // reload executable when user restart the debug "project"
      return {
        ...nodeConfiguration,
        runtimeArgs,
        runtimeExecutable: getExecutableFromConfig() || "node",
      }

    case "go":
      return { ...goConfiguration }

    case "python":
      // reload executable when user restart the debug "project"
      return { ...pythonConfiguration, python: getExecutableFromConfig() || "python" }

    default:
      throw new Error(`Unsupported language: ${language}`)
  }
}

export const getCurrentLanguage = (): SupportedLanguage => {
  return getLanguageFromConfig()
}
