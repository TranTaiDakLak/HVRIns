// wemakemail_test.go — Smoke tests cho WeMakeMail provider.
//
// Cần API key thật để chạy. Set biến môi trường trước:
//
//	$env:WEMAKEMAIL_API_KEY = "wm_live_..."
//
// Chạy toàn bộ:
//
//	go test ./internal/email/temp/ -run TestWeMakeMail -v -timeout=120s
//
// Chỉ test FetchDomains:
//
//	go test ./internal/email/temp/ -run TestWeMakeMail_FetchDomains -v
package temp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func wmApiKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("WEMAKEMAIL_API_KEY")
	if key == "" {
		t.Skip("WEMAKEMAIL_API_KEY chưa set — bỏ qua test")
	}
	return key
}

// TestWeMakeMail_FetchDomains kiểm tra endpoint /api/account/domains.
// Verify: response parse được, ít nhất 1 domain, phân loại free/premium đúng.
func TestWeMakeMail_FetchDomains(t *testing.T) {
	apiKey := wmApiKey(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := FetchWeMakeMailDomains(ctx, apiKey)
	if err != nil {
		t.Fatalf("FetchWeMakeMailDomains FAIL: %v", err)
	}

	t.Logf("Gói: %s | Tổng: %d domain | Free: %d | Trả phí: %d",
		result.Plan, len(result.All), len(result.Free), len(result.Paid))
	for _, d := range result.Free {
		t.Logf("  [free] %s", d)
	}
	for _, d := range result.Paid {
		t.Logf("  [paid] %s", d)
	}

	if len(result.All) == 0 {
		t.Fatal("Không có domain nào — API có thể trả lỗi hoặc account trống")
	}
	if len(result.Free)+len(result.Paid) != len(result.All) {
		t.Errorf("free(%d)+paid(%d) != all(%d) — tier mapping bị lệch",
			len(result.Free), len(result.Paid), len(result.All))
	}
}

// TestWeMakeMail_CreateEmail kiểm tra việc tạo địa chỉ email (chọn domain ngẫu nhiên từ API).
func TestWeMakeMail_CreateEmail(t *testing.T) {
	apiKey := wmApiKey(t)
	svc := NewWeMakeMail(apiKey, "", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	email, err := svc.CreateEmail(ctx)
	if err != nil {
		t.Fatalf("CreateEmail FAIL: %v", err)
	}
	if !strings.Contains(email, "@") {
		t.Fatalf("Email không hợp lệ: %q", email)
	}
	t.Logf("CreateEmail OK: %s", email)
}

// TestWeMakeMail_CreateEmail_CustomDomain kiểm tra khi user chỉ định domain cụ thể.
func TestWeMakeMail_CreateEmail_CustomDomain(t *testing.T) {
	apiKey := wmApiKey(t)

	// Lấy domain đầu tiên từ API để dùng làm custom domain
	ctx := context.Background()
	result, err := FetchWeMakeMailDomains(ctx, apiKey)
	if err != nil || len(result.All) == 0 {
		t.Skip("Không fetch được domain — bỏ qua test custom domain")
	}
	domain := result.All[0]
	t.Logf("Dùng custom domain: %s", domain)

	svc := NewWeMakeMail(apiKey, "", []string{domain})
	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	email, err := svc.CreateEmail(ctx2)
	if err != nil {
		t.Fatalf("CreateEmail với custom domain FAIL: %v", err)
	}
	if !strings.HasSuffix(email, "@"+domain) {
		t.Errorf("Email không dùng domain đã chỉ định: got %q, want suffix @%s", email, domain)
	}
	t.Logf("CreateEmail custom domain OK: %s", email)
}

// TestWeMakeMail_PollInbox kiểm tra poll inbox (không có mail thật → expect empty, không crash).
func TestWeMakeMail_PollInbox(t *testing.T) {
	apiKey := wmApiKey(t)
	svc := NewWeMakeMail(apiKey, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	email, err := svc.CreateEmail(ctx)
	if err != nil {
		t.Fatalf("CreateEmail FAIL: %v", err)
	}
	t.Logf("Inbox: %s", email)

	// Poll 1 lần — không có mail thật, expect empty (không lỗi)
	code, err := svc.pollOnce(ctx)
	if err != nil {
		t.Fatalf("pollOnce FAIL: %v", err)
	}
	if code == "" {
		t.Logf("pollOnce OK: inbox trống (expected với địa chỉ mới tạo)")
	} else {
		t.Logf("pollOnce trả code ngay lập tức: %q (inbox có mail sẵn?)", code)
	}
}

// TestWeMakeMail_PollExistingInbox kiểm tra poll inbox đã có mail OTP sẵn.
// Set 2 env: WEMAKEMAIL_API_KEY và WEMAKEMAIL_TEST_INBOX (vd "vvpzrzxhkm@workshopbmt.com").
// Inbox phải có sẵn ít nhất 1 mail Facebook OTP để verify extraction work.
func TestWeMakeMail_PollExistingInbox(t *testing.T) {
	apiKey := wmApiKey(t)
	inbox := os.Getenv("WEMAKEMAIL_TEST_INBOX")
	if inbox == "" {
		t.Skip("WEMAKEMAIL_TEST_INBOX chưa set — bỏ qua test")
	}

	svc := NewWeMakeMail(apiKey, "", nil)
	svc.email = inbox // override mà không cần CreateEmail

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	code, err := svc.pollOnce(ctx)
	if err != nil {
		t.Fatalf("pollOnce FAIL: %v", err)
	}
	if code == "" {
		t.Fatalf("Không extract được OTP từ inbox %s — inbox có thể trống hoặc subject không match regex", inbox)
	}
	t.Logf("OK — extract được OTP %q từ %s", code, inbox)
}
