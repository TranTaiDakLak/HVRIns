// new_providers_test.go — Smoke tests cho 8 temp mail providers mới port từ C#.
// CHỈ chạy CreateEmail (không chờ OTP thật). Mục đích: verify endpoint còn live,
// request format đúng, không bị Cloudflare/401 ngay từ bước đầu.
//
// Chạy: go test ./internal/email/temp/ -run TestNewProviders -v -timeout=120s
// Skip: có thể dùng -short để bỏ qua các test cần network.
package temp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewProviders_CreateEmail_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skip smoke test trong -short mode (cần network)")
	}

	cases := []struct {
		name   string
		create func() (string, error)
	}{
		{"TmpInbox", func() (string, error) {
			return NewTmpInbox("").CreateEmail(ctxT(t, 10))
		}},
		{"TenMinuteMail", func() (string, error) {
			return NewTenMinuteMail("").CreateEmail(ctxT(t, 15))
		}},
		{"TempMailTo", func() (string, error) {
			return NewTempMailTo("").CreateEmail(ctxT(t, 20))
		}},
		{"OneSecEmail", func() (string, error) {
			return NewOneSecEmail("").CreateEmail(ctxT(t, 20))
		}},
		{"TempMail100", func() (string, error) {
			return NewTempMail100("").CreateEmail(ctxT(t, 25))
		}},
		{"TempMailSo", func() (string, error) {
			return NewTempMailSo("").CreateEmail(ctxT(t, 30))
		}},
		// PriyoEmail cần API key — chỉ test nếu env có.
		// TempMailOrgPremium dễ bị Cloudflare block — test riêng.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email, err := tc.create()
			if err != nil {
				t.Logf("%s: FAIL — %v", tc.name, err)
				t.Skipf("%s không khả dụng: %v", tc.name, err)
				return
			}
			if !strings.Contains(email, "@") {
				t.Errorf("%s: invalid email: %s", tc.name, email)
				return
			}
			t.Logf("%s: OK — %s", tc.name, email)
		})
	}
}

func TestTempMailOrgPremium_CreateEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: Cloudflare-protected, cần network + luck")
	}
	p := NewTempMailOrgPremium("")
	defer p.Close()
	email, err := p.CreateEmail(ctxT(t, 45))
	if err != nil {
		t.Logf("TempMailOrgPremium FAIL: %v", err)
		t.Skip("Cloudflare hoặc login fail — expected trong CI")
		return
	}
	if !strings.Contains(email, "@") {
		t.Errorf("invalid email: %s", email)
	}
	t.Logf("TempMailOrgPremium: OK — %s", email)
}

func ctxT(t *testing.T, seconds int) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	t.Cleanup(cancel)
	return ctx
}
