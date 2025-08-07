package extractors

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
