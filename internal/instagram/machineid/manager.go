// Package machineid — Manage Facebook machine_id and device identifiers
// Mapping từ C#: IFacebookMachineIdManager / FacebookMachineIdManager
//
// machine_id is a persistent device identifier that Facebook uses to track
// sessions across logins. It must be stable per device/account and consistent
// across all API requests from the same session.
//
// TODO: Port from C# FacebookMachineIdManager — persist per account, generate
// on first use, include in all form bodies via body.go constants.
package machineid

import (
	"crypto/rand"
	"encoding/hex"
)

// Manager handles generation and storage of Facebook machine_id values.
// Mapping từ C#: IFacebookMachineIdManager
type Manager struct {
	// machineID is the stable identifier for this device/session.
	machineID string
}

// New creates a Manager with a freshly generated machine_id.
func New() *Manager {
	return &Manager{machineID: generate()}
}

// NewWithID creates a Manager restoring a previously saved machine_id.
func NewWithID(id string) *Manager {
	if id == "" {
		id = generate()
	}
	return &Manager{machineID: id}
}

// Get returns the current machine_id string.
func (m *Manager) Get() string {
	return m.machineID
}

// generate creates a new random 16-byte hex machine_id (32 hex chars).
// Format matches Facebook's expected format for machine_id field.
func generate() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
