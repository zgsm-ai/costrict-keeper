package models

/**
 * Component object (serialized to JSON format)
 * @property {string} name - Component name
 * @property {string} version - Component version
 * @property {bool} installed - Whether the component is installed
 */
type ComponentInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
}
