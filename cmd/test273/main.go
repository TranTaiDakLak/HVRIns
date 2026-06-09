// cmd/test273/main.go — test full s273 verify flow: add email → OTP → confirm.
// Dùng: go run ./cmd/test273 <token> <email>
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	addEmailURL = "https://b-api.facebook.com/method/user.editregistrationcontactpoint"
	confirmURL  = "https://b-api.facebook.com/method/user.confirmcontactpoint"
	resendURL   = "https://b-api.facebook.com/method/user.sendconfirmationcode"

	addEmailFriendly = "editRegistrationContactpoint"
	confirmFriendly  = "confirmContactpoint"
	resendFriendly   = "sendConfirmationCode"

	addEmailCaller = "com.facebook.confirmation.fragment.ConfContactpointFragment"
	confirmCaller  = "com.facebook.confirmation.fragment.ConfCodeInputFragment"
	resendCaller   = "com.facebook.confirmation.fragment.ConfCodeInputFragment"

	uaStr = "Dalvik/2.1.0 (Linux; U; Android 9; V2242A Build/PQ3A.190705.05150936) [FBAN/FB4A;FBAV/273.0.0.39.123;FBPN/com.facebook.katana;FBLC/vi_VN;FBBV/218047977;FBCR/MobiFone;FBMF/vivo;FBBD/vivo;FBDV/V2242A;FBSV/9;FBCA/x86:armeabi-v7a;FBDM/{density=1.5,width=900,height=1600};FB_FW/1;FBRV/0;]"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./cmd/test273 <token> <email>")
		os.Exit(1)
	}
	token := os.Args[1]
	emailAddr := os.Args[2]

	client := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	// ─── Step 1: Add email ────────────────────────────────────────────────────
	fmt.Printf("\n[STEP 1] Add email: %s\n", emailAddr)
	addResp := doPost(ctx, client, token, addEmailURL, addEmailFriendly, addEmailCaller, url.Values{
		"add_contactpoint":      {emailAddr},
		"add_contactpoint_type": {"EMAIL"},
		"format":                {"json"},
		"locale":                {"vi_VN"},
		"client_country_code":   {"VN"},
		"method":                {"user.editregistrationcontactpoint"},
		"fb_api_req_friendly_name": {addEmailFriendly},
		"fb_api_caller_class":   {addEmailCaller},
	})

	fmt.Printf("Response: %s\n", addResp)
	trimmed := strings.TrimSpace(addResp)
	if trimmed != "true" && !strings.Contains(addResp, `"result":true`) {
		fmt.Println("❌ Add email FAILED — dừng.")
		os.Exit(1)
	}
	fmt.Println("✅ Add email OK!")

	// ─── Step 2: Nhập OTP ────────────────────────────────────────────────────
	fmt.Printf("\n[STEP 2] Kiểm tra hộp thư %s và nhập OTP (hoặc 'r' để resend, 'q' để thoát): ", emailAddr)
	reader := bufio.NewReader(os.Stdin)
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	if strings.ToLower(code) == "q" {
		fmt.Println("Thoát.")
		os.Exit(0)
	}

	if strings.ToLower(code) == "r" {
		fmt.Println("[RESEND] Gửi lại code...")
		resendResp := doPost(ctx, client, token, resendURL, resendFriendly, resendCaller, url.Values{
			"normalized_contactpoint": {emailAddr},
			"contactpoint_type":       {"EMAIL"},
			"format":                  {"json"},
			"locale":                  {"vi_VN"},
			"client_country_code":     {"VN"},
			"method":                  {"user.sendconfirmationcode"},
			"fb_api_req_friendly_name": {resendFriendly},
			"fb_api_caller_class":     {resendCaller},
		})
		fmt.Printf("Resend response: %s\n", resendResp)
		fmt.Print("Nhập OTP sau khi resend: ")
		code, _ = reader.ReadString('\n')
		code = strings.TrimSpace(code)
	}

	if code == "" {
		fmt.Println("Không có OTP, thoát.")
		os.Exit(1)
	}

	// ─── Step 3: Confirm ─────────────────────────────────────────────────────
	fmt.Printf("\n[STEP 3] Confirm OTP: %s\n", code)
	confResp := doPost(ctx, client, token, confirmURL, confirmFriendly, confirmCaller, url.Values{
		"normalized_contactpoint": {emailAddr},
		"contactpoint_type":       {"EMAIL"},
		"code":                    {code},
		"source":                  {"ANDROID_DIALOG_API"},
		"surface":                 {"hard_cliff"},
		"format":                  {"json"},
		"locale":                  {"vi_VN"},
		"client_country_code":     {"VN"},
		"method":                  {"user.confirmcontactpoint"},
		"fb_api_req_friendly_name": {confirmFriendly},
		"fb_api_caller_class":     {confirmCaller},
	})

	fmt.Printf("Response: %s\n", confResp)
	respLow := strings.ToLower(confResp)
	switch {
	case strings.Contains(respLow, `"error_code":459`) || strings.Contains(respLow, "checkpointed"):
		fmt.Println("⚠️  CHECKPOINT / Die")
	case strings.Contains(respLow, `"error_code":`) || strings.Contains(respLow, `"error_msg":`):
		fmt.Println("❌ Confirm FAILED — lỗi từ FB")
	default:
		fmt.Println("✅ Confirm OK! — Email đã được xác nhận")
	}
}

func doPost(ctx context.Context, client *http.Client, token, targetURL, friendly, caller string, params url.Values) string {
	// Gzip-compress request body (server expects content-encoding: gzip).
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte(params.Encode()))
	_ = gz.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, &buf)
	if err != nil {
		fmt.Println("ERROR:", err)
		return ""
	}
	req.Header.Set("host", "b-api.facebook.com")
	req.Header.Set("x-fb-connection-quality", "EXCELLENT")
	req.Header.Set("x-fb-sim-hni", "45201")
	req.Header.Set("x-fb-connection-type", "unknown")
	req.Header.Set("user-agent", uaStr)
	req.Header.Set("x-fb-connection-bandwidth", "7903429")
	req.Header.Set("authorization", "OAuth "+token)
	req.Header.Set("x-fb-friendly-name", friendly)
	req.Header.Set("x-fb-net-hni", "45201")
	req.Header.Set("x-zero-state", "unknown")
	req.Header.Set("content-encoding", "gzip")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("x-tigon-is-retry", "False")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("x-fb-http-engine", "Liger")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP ERROR:", err)
		return ""
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gr, gerr := gzip.NewReader(resp.Body)
		if gerr == nil {
			defer gr.Close()
			reader = gr
		}
	}
	raw, _ := io.ReadAll(reader)
	return string(raw)
}
