// register.go — Web platform Registerer + package type aliases.
//
// File này gộp 2 file cũ:
//   - register.go (cũ) → Registerer interface + init()
//   - types.go          → type aliases (RegInput/RegSession/RegResult/StatusCallback)
//
// Package web — Facebook Web platform register (m.facebook.com endpoints).
package web

import (
	"context"

	"HVRIns/internal/instagram"
)

// ─── Type aliases ────────────────────────────────────────────────────────────
// Cho phép các file trong package dùng tên ngắn thay vì prefix facebook. mỗi lần.

type (
	RegInput       = instagram.RegInput
	RegSession     = instagram.RegSession
	RegResult      = instagram.RegResult
	StatusCallback = func(string)
)

// ─── Registerer interface + init ─────────────────────────────────────────────

// Registerer implements instagram.Registerer cho nền tảng web (m.facebook.com).
type Registerer struct{}

// Register thực hiện toàn bộ flow đăng ký qua 8 bước Bloks.
// Implements instagram.Registerer interface.
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return RegisterAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformWeb, func() instagram.Registerer {
		return &Registerer{}
	})
}
