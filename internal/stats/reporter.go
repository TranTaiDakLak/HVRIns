// Package stats — Collect and report operational statistics
// Mapping từ C#: IStaticsReport + StaticsReport implementation
//
// Tracks counters for register/verify outcomes (success, fail, checkpoint,
// live, die, etc.) and provides summary reporting for the UI status bar.
//
// TODO: Port from C# StaticsReport — thread-safe counters, rate tracking,
// ETA estimation, per-platform breakdown.
package stats

import (
	"fmt"
	"sync/atomic"
)

// Reporter collects operational statistics for register/verify runs.
// Mapping từ C#: IStaticsReport
type Reporter struct {
	// Register counters
	RegTotal      atomic.Int64
	RegSuccess    atomic.Int64
	RegFail       atomic.Int64
	RegNeedVerify atomic.Int64

	// Verify counters
	VerifyTotal   atomic.Int64
	VerifyLive    atomic.Int64
	VerifyDie     atomic.Int64
	VerifyError   atomic.Int64

	// Shared
	ProxyFail atomic.Int64
}

// New creates a zeroed Reporter.
func New() *Reporter {
	return &Reporter{}
}

// AddRegSuccess increments the register success counter.
func (r *Reporter) AddRegSuccess() { r.RegSuccess.Add(1); r.RegTotal.Add(1) }

// AddRegFail increments the register failure counter.
func (r *Reporter) AddRegFail() { r.RegFail.Add(1); r.RegTotal.Add(1) }

// AddRegNeedVerify increments the "needs phone verification" counter.
func (r *Reporter) AddRegNeedVerify() { r.RegNeedVerify.Add(1); r.RegTotal.Add(1) }

// AddVerifyLive increments the verify live counter.
func (r *Reporter) AddVerifyLive() { r.VerifyLive.Add(1); r.VerifyTotal.Add(1) }

// AddVerifyDie increments the verify die counter.
func (r *Reporter) AddVerifyDie() { r.VerifyDie.Add(1); r.VerifyTotal.Add(1) }

// AddVerifyError increments the verify error counter.
func (r *Reporter) AddVerifyError() { r.VerifyError.Add(1); r.VerifyTotal.Add(1) }

// AddProxyFail increments the proxy failure counter.
func (r *Reporter) AddProxyFail() { r.ProxyFail.Add(1) }

// RegSummary returns a human-readable register stats string.
func (r *Reporter) RegSummary() string {
	return fmt.Sprintf("Reg: %d OK / %d fail / %d verify / %d total",
		r.RegSuccess.Load(), r.RegFail.Load(), r.RegNeedVerify.Load(), r.RegTotal.Load())
}

// VerifySummary returns a human-readable verify stats string.
func (r *Reporter) VerifySummary() string {
	return fmt.Sprintf("Verify: %d live / %d die / %d err / %d total",
		r.VerifyLive.Load(), r.VerifyDie.Load(), r.VerifyError.Load(), r.VerifyTotal.Load())
}

// Reset zeros all counters.
func (r *Reporter) Reset() {
	r.RegTotal.Store(0)
	r.RegSuccess.Store(0)
	r.RegFail.Store(0)
	r.RegNeedVerify.Store(0)
	r.VerifyTotal.Store(0)
	r.VerifyLive.Store(0)
	r.VerifyDie.Store(0)
	r.VerifyError.Store(0)
	r.ProxyFail.Store(0)
}
