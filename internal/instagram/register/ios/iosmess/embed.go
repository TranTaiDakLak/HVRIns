// Package iosmess — Messenger Lite iOS registration (Bloks CAA, FBAV/563).
//
// Platform "iosmessreg": làm TOÀN BỘ flow reg + email confirm trong 1 Register():
//   pre-steps (aymh/name/birthday/gender/contactpoint_email/password)
//   → create.account → extract crypted_user_id → add-mail (contactpoint_email.async)
//   → screen loads (bottomsheet/change_email) → GetOTP → confirmation.async.
//
// Mirror appmv3reg (Android Messenger) nhưng dùng iOS transport: graph.facebook.com/graphql,
// Safari TLS, x-graphql-client-library:pando, app token 437626316973788, password PLAINTEXT
// (#PWD_ENC:0). Templates byte-faithful từ capture (embed), chỉ thay trường động.
//
// KHÔNG đụng register/android/appmessv3 (Android Mess — platform riêng).
package iosmess

import (
	"embed"
	"strings"
)

//go:embed templates/*.txt
var capFS embed.FS

// loadTemplate đọc body template đã embed (đã là body, không có header).
func loadTemplate(name string) (string, error) {
	b, err := capFS.ReadFile("templates/" + name + ".txt")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
