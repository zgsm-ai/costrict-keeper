package models

import (
	"encoding/json"
	"time"
)

// Tunnel represents a local service tunnel mapping
type Tunnel struct {
	Name        string    `json:"name"`        // service name
	LocalPort   int       `json:"localPort"`   // local port
	MappingPort int       `json:"mappingPort"` // mapping port to cloud
	Protocol    string    `json:"protocol"`    // protocol type(http/https)
	Status      RunStatus `json:"status"`      // tunnel status(running/stopped/error/exited)
	CreatedTime time.Time `json:"createdTime"` // creation time
	Pid         int       `json:"pid"`         // process ID of the tunnel
}

// NewTunnel creates new Tunnel instance
func NewTunnel(name string, localPort, mappingPort int, protocol string) *Tunnel {
	return &Tunnel{
		Name:        name,
		LocalPort:   localPort,
		MappingPort: mappingPort,
		Protocol:    protocol,
		Status:      StatusStopped,
	}
}

// ToJSON converts Tunnel to JSON string
func (t *Tunnel) ToJSON() (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses Tunnel from JSON string
func FromJSON(data string) (*Tunnel, error) {
	var t Tunnel
	err := json.Unmarshal([]byte(data), &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
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

// CreateTunnelRequest defines create tunnel API request parameters
type CreateTunnelRequest struct {
	AppName string `json:"app"`  // application name(path parameter)
	Port    int    `json:"port"` // port number(query parameter)
}

// DeleteTunnelRequest defines delete tunnel API request parameters
type DeleteTunnelRequest struct {
	AppName string `json:"app"`  // application name(path parameter)
	Port    int    `json:"port"` // port number(query parameter)
}
