package webandroid

import (
	"net/url"
	"strings"
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"
)

// TestGetCookiesFBOrder kiểm tra thứ tự cookie match C# L228 + double-decode xs.
func TestGetCookiesFBOrder(t *testing.T) {
	sess, err := newSession("")
	if err != nil {
		t.Fatalf("newSession: %v", err)
	}

	// Set cookie theo thứ tự "lộn xộn" để verify Go sort đúng theo C#.
	u, _ := url.Parse("https://m.facebook.com")
	// xs = double-encoded: original "v1.0" → "v1.0" nhưng Facebook thường trả
	// xs URL-encoded 2 lần kiểu "%2526x%253Dy%2526" → sau 2 lần decode là "&x=y&"
	doubleEncoded := "%2526x%253Dy%2526" // "%26x%3Dy%26" sau lần 1, "&x=y&" sau lần 2
	sess.client.SetCookies(u, []*fhttp.Cookie{
		{Name: "fr", Value: "0xABC", Path: "/", Domain: ".facebook.com"},
		{Name: "c_user", Value: "100012345", Path: "/", Domain: ".facebook.com"},
		{Name: "datr", Value: "datr_value", Path: "/", Domain: ".facebook.com"},
		{Name: "xs", Value: doubleEncoded, Path: "/", Domain: ".facebook.com"},
		{Name: "sb", Value: "sb_value", Path: "/", Domain: ".facebook.com"},
		{Name: "pas", Value: "pas_value", Path: "/", Domain: ".facebook.com"},
		// Cookie khác không thuộc 6 FB cookies → bị skip
		{Name: "_fbp", Value: "some_tracking", Path: "/", Domain: ".facebook.com"},
	})

	got := sess.getCookiesFBOrder()
	// Thứ tự phải là: datr;sb;c_user;xs;fr;pas
	want := "datr=datr_value;sb=sb_value;c_user=100012345;xs=&x=y&;fr=0xABC;pas=pas_value"
	if got != want {
		t.Errorf("cookie order/format sai:\ngot:  %q\nwant: %q", got, want)
	}

	// _fbp không được xuất hiện
	if strings.Contains(got, "_fbp") {
		t.Errorf("_fbp không thuộc 6 cookies FB, phải bị skip")
	}
}

// TestGetCookiesFBOrder_MissingSomeCookies kiểm tra khi thiếu cookie không lỗi.
func TestGetCookiesFBOrder_MissingSomeCookies(t *testing.T) {
	sess, err := newSession("")
	if err != nil {
		t.Fatalf("newSession: %v", err)
	}
	u, _ := url.Parse("https://m.facebook.com")
	sess.client.SetCookies(u, []*fhttp.Cookie{
		{Name: "datr", Value: "d1", Path: "/", Domain: ".facebook.com"},
		{Name: "c_user", Value: "u1", Path: "/", Domain: ".facebook.com"},
		// Không có xs, fr, sb, pas
	})

	got := sess.getCookiesFBOrder()
	want := "datr=d1;c_user=u1"
	if got != want {
		t.Errorf("missing cookies should be skipped:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestGetCookiesFBOrder_EmptyCookie verify return rỗng khi không có cookie.
func TestGetCookiesFBOrder_EmptyCookie(t *testing.T) {
	sess, err := newSession("")
	if err != nil {
		t.Fatalf("newSession: %v", err)
	}
	got := sess.getCookiesFBOrder()
	if got != "" {
		t.Errorf("empty jar should return empty string, got %q", got)
	}
}
