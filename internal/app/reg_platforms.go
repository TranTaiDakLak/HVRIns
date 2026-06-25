package app

import (
	"strings"

	"HVRIns/internal/instagram"
)

// reg_platforms.go — danh sách platform reg (IG-only sau khi gỡ toàn bộ FB sXXX/iOS).
//
// Thay cho regPlatformList cũ trong app_reg_sxxx.go (đã xoá cùng 181 platform FB).
// Giữ nguyên ngữ nghĩa multi-version: user có thể chọn nhiều version qua
// InteractionConfig.ApiRegPlatforms; danh sách được trim + bỏ rỗng + dedup, giữ thứ tự.
// Khác bản cũ ở chỗ default fallback là IG (ig_ios_bloks) thay vì PlatformWeb (FB).

// regPlatformList trả về danh sách platform reg user đã chọn (hỗ trợ multi-version).
//   - ApiRegPlatforms (len>0) → dùng list này, trim + bỏ rỗng + dedup, giữ thứ tự.
//   - Rỗng → fallback [ApiRegPlatform].
//   - Vẫn rỗng → [PlatformIGIOSBloks] (default IG sau rebrand).
func regPlatformList(c InteractionConfig) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(c.ApiRegPlatforms)+1)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, p := range c.ApiRegPlatforms {
		add(p)
	}
	if len(out) == 0 {
		add(c.ApiRegPlatform)
	}
	if len(out) == 0 {
		out = append(out, instagram.PlatformIGIOSBloks)
	}
	return out
}
