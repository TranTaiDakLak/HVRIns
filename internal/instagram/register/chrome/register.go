// Package chrome — Facebook Chrome extension platform Registerer (skeleton)
// Mapping từ C#: FacebookRegisterAutomation (Chrome extension variant)
// TODO: Implement using Chrome extension flow + chrome-specific headers
package chrome

import (
	"context"

	"HVRIns/internal/instagram"
)

// Registerer implements instagram.Registerer for the Chrome extension platform.
type Registerer struct{}

// Register performs account creation via Chrome extension flow.
// Currently returns not-implemented result; full implementation pending.
func (r *Registerer) Register(_ context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	if onStatus != nil {
		onStatus("[chrome] Register: not yet implemented")
	}
	return &instagram.RegResult{Success: false, Message: "chrome register: not implemented"}
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformChrome, func() instagram.Registerer {
		return &Registerer{}
	})
}
