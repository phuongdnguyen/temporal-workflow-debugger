<script lang="ts">
  import Icon from "../components/icon/icon.svelte"
  import SubmitButton from "../components/submit-button.svelte"

  let loading = false
  let error = ""
  let history = ""

  /**
   * Event listener for starting a session from workflow ID
   */
  function startFromWorkflowId(e: Event) {
    if (!(e.target instanceof HTMLFormElement)) {
      throw new TypeError("Expected form element")
    }
    const data = Object.fromEntries(new FormData(e.target))

    vscode.postMessage({
      type: "startFromId",
      ...data,
    })
  }

  /**
   * Event listener for loading history from workflow ID without starting debug session
   */
  function loadHistoryFromWorkflowId(e: Event) {
    if (!(e.target instanceof HTMLFormElement)) {
      throw new TypeError("Expected form element")
    }
    const data = Object.fromEntries(new FormData(e.target))

    vscode.postMessage({
      type: "loadHistoryFromId",
      ...data,
    })
  }

  /**
   * Get form data from workflow ID fields
   */
  function getWorkflowIdFormData() {
    const namespaceField = document.getElementById('namespace-field') as any
    const workflowIdField = document.getElementById('workflow-id-field') as any
    const runIdField = document.getElementById('run-id-field') as any
    
    return {
      namespace: namespaceField?.value || '',
      workflowId: workflowIdField?.value || '',
      runId: runIdField?.value || ''
    }
  }

  /**
   * Event listener for starting a session from workflow ID (button click)
   */
  function startFromWorkflowIdClick() {
    const data = getWorkflowIdFormData()
    if (!data.workflowId) {
      error = "Workflow ID is required"
      return
    }
    error = ""
    vscode.postMessage({
      type: "startFromId",
      ...data,
    })
  }

  /**
   * Event listener for loading history from workflow ID without starting debug session (button click)
   */
  function loadHistoryFromWorkflowIdClick() {
    const data = getWorkflowIdFormData()
    if (!data.workflowId) {
      error = "Workflow ID is required"
      return
    }
    error = ""
    vscode.postMessage({
      type: "loadHistoryFromId",
      ...data,
    })
  }

  /**
   * Reads and parses JSON history file
   */
  async function processHistory(file: File) {
    const reader = new FileReader()
    reader.readAsText(file)
    reader.onloadstart = () => {
      loading = true
    }
    reader.onloadend = (event) => {
      const result = event?.target?.result as string
      try {
        history = JSON.parse(result) ?? ""
      } catch {
        error = "Could not parse JSON"
      }
      loading = false
    }
    reader.onerror = () => {
      error = "Failed to read file: " + reader.error
      loading = false
    }
  }

  function handleHistoryFile(e: Event) {
    const target = e.target as HTMLInputElement
    const file = target?.files?.[0]
    error = ""
    if (file) {
      processHistory(file)
    }
  }

  /**
   * Event listener for starting a session from history file
   */
  function startFromHistoryFile() {
    if (history) {
      vscode.postMessage({
        type: "startFromHistory",
        history,
      })
    }
  }

  /**
   * Event listener for loading history from file without starting debug session
   */
  function loadHistoryFromFile() {
    if (history) {
      vscode.postMessage({
        type: "loadHistoryFromFile",
        history,
      })
    }
  }
</script>

<section>
  <p>Debug by ID</p>
  <div class="debug-by-id-container">
    <div class="form-fields">
      <vscode-text-field id="namespace-field" type="text" placeholder="Namespace (default)" />
      <vscode-text-field id="workflow-id-field" type="text" required placeholder="Workflow ID *" />
      <vscode-text-field id="run-id-field" type="text" placeholder="Run ID" />
    </div>
    {#if error}
      <div class="error">
        <Icon name="error" />
        <p>{error}</p>
      </div>
    {/if}
    <div class="button-group">
      <vscode-button type="button" on:click={startFromWorkflowIdClick}>Start</vscode-button>
      <vscode-button type="button" appearance="secondary" on:click={loadHistoryFromWorkflowIdClick}>Load History</vscode-button>
    </div>
  </div>
  <vscode-divider role="presentation" />
  <p>Debug from history file</p>
  <form on:submit|preventDefault={startFromHistoryFile}>
    <div class="debug-history-file">
      {#if loading}
        <vscode-progress-ring />
      {/if}
      <label for="history-file" hidden>History file</label>
      <input id="history-file" name="file" type="file" required on:change={handleHistoryFile} />
    </div>
    {#if error}
      <div class="error">
        <Icon name="error" />
        <p>{error}</p>
      </div>
    {/if}
    <div class="debug-history-btn">
      <div class="button-group">
        <SubmitButton disabled={loading || error}>Start</SubmitButton>
        <vscode-button type="button" appearance="secondary" disabled={loading || error || !history} on:click={loadHistoryFromFile}>Load History</vscode-button>
      </div>
    </div>
  </form>
</section>

<style>
  .debug-by-id-form {
    display: flex;
    margin-bottom: 0.5rem;
  }
  vscode-text-field {
    margin-right: 0.625rem;
  }

  .debug-by-id-container {
    margin-bottom: 0.5rem;
  }

  .form-fields {
    display: flex;
    margin-bottom: 0.5rem;
  }

  .form-fields vscode-text-field {
    margin-right: 0.625rem;
  }

  .button-group {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .debug-history-file {
    display: flex;
    align-items: center;
  }

  .debug-history-file vscode-progress-ring {
    margin-right: 1rem;
  }
  .debug-history-btn {
    margin-top: 0.875rem;
  }

  .error {
    display: flex;
    align-items: center;
    margin-top: 0.5rem;
    color: #f14c4c;
  }

  .error p {
    margin: 0 0 0 0.5rem;
  }
</style>
