package models

type PortPair struct {
	LocalPort   int `json:"localPort"`   // local port
	MappingPort int `json:"mappingPort"` // mapping port to cloud
}

// ErrorResponse defines API error response format
type ErrorResponse struct {
	Error string `json:"error"`
}

// TunnelResponse defines tunnel operation success response format
type TunnelResponse struct {
	AppName string `json:"appName"` // application name
	Status  string `json:"status"`  // operation status
	Message string `json:"message"` // response message
}
