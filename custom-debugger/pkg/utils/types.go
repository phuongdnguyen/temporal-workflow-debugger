package utils

// JSONRPCRequest is the JSON-RPC message structures
type JSONRPCRequest struct {
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type JSONRPCResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

// DAP message structures

type DAPRequest struct {
	Seq       int         `json:"seq"`
	Type      string      `json:"type"`
	Command   string      `json:"command"`
	Arguments interface{} `json:"arguments"`
}

type DAPResponse struct {
	Seq        int         `json:"seq"`
	Type       string      `json:"type"`
	RequestSeq int         `json:"request_seq"`
	Command    string      `json:"command"`
	Success    bool        `json:"success"`
	Body       interface{} `json:"body,omitempty"`
	Message    string      `json:"message,omitempty"`
	Event      string      `json:"event,omitempty"` // For DAP events
}
