package utils

import (
	"fmt"
)

// CheckPortConnectable checks if a port is connectable on localhost
func CheckPortConnectable(port int) bool {
	return checkPortConnectable(port)
}

// CheckPortListenable checks if a port is listenable
func CheckPortListenable(port int) bool {
	return checkPortListenable(port)
}

func isPortAllocated(port int) bool {
	allocated, ok := portAllocs[port]
	if !ok {
		return false
	}
	return allocated
}

func isPortAvailable(port int) bool {
	return !isPortAllocated(port) && checkPortListenable(port)
}

var minPort int = 9000
var maxPort int = 10000
var portAllocs map[int]bool = make(map[int]bool)

func SetAvailablePortRange(min, max int) {
	minPort = min
	maxPort = max
}

func SetPortAllocated(port int) {
	portAllocs[port] = true
}

func AllocPort(preferredPort int) (port int, err error) {
	if preferredPort != 0 && isPortAvailable(preferredPort) {
		portAllocs[preferredPort] = true
		return preferredPort, nil
	}
	for p := minPort; p <= maxPort; p++ {
		if isPortAvailable(p) {
			portAllocs[p] = true
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port found within range %d-%d", minPort, maxPort)
}

func FreePort(port int) {
	portAllocs[port] = false
}
