import vscode from "vscode"

function ignoreErrors(t: Thenable<any>) {
  t.then(
    () => {
      // ignore
    },
    () => {
      // ignore
    },
  )
}
// TODO: remove after done with debugging js-debug
export function registerDebugAdapterTrackerFactory(): vscode.Disposable {
  // TODO: register for other types too?
  return vscode.debug.registerDebugAdapterTrackerFactory("pwa-node", {
    createDebugAdapterTracker(_session) {
      return {
        onDidSendMessage(m) {
          console.log(`DAP -> editor: ${JSON.stringify(m)}`)
        },
      }
    },
  })
}
