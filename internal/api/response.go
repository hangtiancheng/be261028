package api

// Response mirrors com.buy.group.easy.api.response.Response<T>, the uniform REST
// envelope returned by every legacy endpoint.
type Response struct {
	Code string `json:"code"`
	Info string `json:"info"`
	Data any    `json:"data,omitempty"`
}

// OK builds a success envelope.
func OK(data any) *Response {
	return &Response{Code: "0000", Info: "成功", Data: data}
}

// Fail builds an error envelope from a code/info pair.
func Fail(code, info string) *Response {
	return &Response{Code: code, Info: info}
}

// RPCResult is the uniform envelope carried across yukino_rpc. Business outcomes
// (including business errors with response codes) travel in Code/Info rather than
// as transport errors, because the RPC layer flattens handler errors to plain
// strings and would lose the structured code.
type RPCResult struct {
	Code string `json:"code"`
	Info string `json:"info"`
}

// IsSuccess reports whether the RPC result represents a success (code 0000).
func (r RPCResult) IsSuccess() bool { return r.Code == "0000" }
