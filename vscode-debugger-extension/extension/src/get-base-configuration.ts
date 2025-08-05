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
  internalConsoleOptions: "openOnSessionStart",
  pauseForSourceMap: true,
} satisfies vscode.DebugConfiguration

const goConfiguration = {
  name: "Launch Program",
  type: "go",
  request: "launch",
  mode: "debug",
  program: "${workspaceFolder}", // Will be overridden with actual replayer endpoint
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
  type: "python",
  request: "launch",
  program: "${workspaceFolder}",
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
