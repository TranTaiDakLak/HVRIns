// cmd/verifytest/main.go — Test verify 1 account thực tế với Store1s mail
// Chạy: go run ./cmd/verifytest/
package main

import (
	"context"
	"fmt"
	"time"

	"HVRIns/internal/email/rent"
	"HVRIns/internal/instagram"
	_ "HVRIns/internal/instagram/verify/web"
)

const (
	testUID      = "61575419885228"
	testPassword = "cibhyslj0bpv"
	testCookie   = "c_user=61575419885228;xs=6:0-7FgH0ol12bbw:2:1775024105:-1:-1;locale=it_IT;fr=0fnm0FvptEJJ4d0Yj.AWdsczDY_dCTOCAn75S79eCUb-ODzKV3ZTEv-iodQOnGUld6vSw.BpzLfl..AAA.0.0.BpzLfl.AWcotXFDYqTSiyurtxVN6iQunuM;datr=CZXMaUbue1YCGP9wRegmXI2l;"
	testToken    = "EAAAAUaZA8jlABRGZC8xEJ8li2YvZCjzuZA6iI8LflzZBxH2YhfGsGjTwUfQoZAoCjvQyZAw9sKScMlP8gz7yUp6cZBwjlvnlTLEk7bjUHMlQjLLCKj1htg8cClC48bUX0FZAUCANz9IAq2MtfldTkNQrAjUjTLhZB2nW5ZAiyypAxJwPbPBRW3bh63L5CYg0sO3ZAN9u2hAJarqwUAZDZD"
	testFullName = "Test User"

	store1sApiKey    = "cd5ba78d21d2d38112f68970da4285b4QgRLoXU6M9GbkEPTArYNFuqtnKa0Vyix"
	store1sProductID = "50510"
)

func main() {
	fmt.Println("=== Verify Test — Store1s Mail ===")
	fmt.Printf("UID: %s\n\n", testUID)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	session := &instagram.Session{
		UID:          testUID,
		Password:     testPassword,
		Cookie:       testCookie,
		Token:        testToken,
		FullName:     testFullName,
		Proxy:        "",
		InputAccount: testUID + "|" + testPassword,
	}
	session.UserAgent = instagram.DefaultUserAgent()

	fmt.Println("[Login] Đang đăng nhập lấy session tokens...")
	loginResult, err := instagram.LoginWithCookieMobile(ctx, session)
	if err != nil {
		fmt.Printf("❌ Login lỗi: %v\n", err)
		return
	}
	if !loginResult.Success {
		fmt.Printf("❌ Login thất bại: %s\n", loginResult.Message)
		return
	}
	fmt.Printf("✅ Login OK — fb_dtsg=%s... jazoest=%s\n\n", truncate(session.FbDtsg, 15), session.Jazoest)

	notify := func(msg string) { fmt.Println("[EMAIL]", msg) }
	s1s := rent.NewStore1s(store1sApiKey, store1sProductID, "")
	s1s.SetOnStatus(notify)
	_ = s1s

	cfg := &instagram.VerifyConfig{
		VerifyEnabled:     true,
		MailProvider:      "store1s",
		Store1sApiKey:     store1sApiKey,
		Store1sProductID:  store1sProductID,
		CheckLiveDie:      true,
		TimeDelayCheck:    5,
		TimeDelaySendCode: 8,
		SendAgainCode:     true,
		OutputPath:        "",
	}

	fmt.Println("[Verify] Bắt đầu verify...")
	verifier, err := instagram.NewVerifier("web")
	if err != nil {
		fmt.Printf("❌ NewVerifier: %v\n", err)
		return
	}
	result := verifier.Verify(ctx, session, cfg, "", func(uid, msg string) {
		fmt.Printf("[%s] %s\n", uid, msg)
	})

	fmt.Println()
	if result.Success {
		fmt.Printf("✅ THÀNH CÔNG — Status: %s\n", result.Status)
		fmt.Printf("   Email dùng: %s\n", result.Email)
		fmt.Printf("   Message: %s\n", result.Message)
	} else {
		fmt.Printf("❌ THẤT BẠI — %s\n", result.Message)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
