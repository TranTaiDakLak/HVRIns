// extractcode_test.go — Test ExtractCode với:
//   - Subject ở nhiều ngôn ngữ (EN/VN/ES/PT/ID/FR)
//   - Body HTML Facebook (universal — letter-spacing:5px structure)
//   - Trường hợp subject empty → phải extract từ body
//   - False-positive cases (số trong subject không phải OTP)
package temp

import "testing"

func TestExtractCode_Empty(t *testing.T) {
	if got := ExtractCode(""); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestExtractCode_Subject(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		want    string
	}{
		{"EN_facebook", "57603 is your confirmation code", "57603"},
		{"VN_facebook", "53227 là mã xác nhận của bạn", "53227"},
		{"ES_facebook", "12345 es tu código de confirmación", "12345"},
		{"PT_facebook", "98765 é o seu código de confirmação", "98765"},
		{"ID_facebook", "44556 adalah kode konfirmasi Anda", "44556"},
		{"FR_facebook", "11223 est votre code de confirmation", "11223"},
		{"start_only_digits_then_space", "98765 Facebook security alert", "98765"},
		{"facebook_brand_anywhere", "Your Facebook code is 67890", "67890"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractCode(tc.subject); got != tc.want {
				t.Errorf("subject %q: got %q, want %q", tc.subject, got, tc.want)
			}
		})
	}
}

func TestExtractCode_NotMatch(t *testing.T) {
	// Các subject KHÔNG được match (false positive prevention)
	cases := []string{
		"Order 12345 has been confirmed",          // số ở giữa — không phải OTP
		"Welcome to our newsletter",               // không có số
		"Your invoice #98765 is ready",            // số sau "#"
		"Promotion: 50% off today!",               // số % không phải OTP
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			if got := ExtractCode(s); got != "" {
				t.Errorf("subject %q: should NOT match, got %q", s, got)
			}
		})
	}
}

// TestExtractCode_FacebookHTMLBody — kiểm tra extract từ HTML body Facebook.
// Pattern 3 (letter-spacing:5px) hoạt động bất kể ngôn ngữ vì FB dùng cùng template HTML.
func TestExtractCode_FacebookHTMLBody(t *testing.T) {
	// Mẫu HTML thật từ Facebook (rút gọn từ test trước)
	html := `<table><tr><td style="font-family:Roboto;font-size:17px; font-weight: 700;
            letter-spacing: 5px; margin-left: 0px; margin-right: 0px;">57603</span></td></tr></table>`
	if got := ExtractCode(html); got != "57603" {
		t.Errorf("HTML body extraction failed: got %q, want %q", got, "57603")
	}

	// Verify hoạt động bất kể ngôn ngữ — html chỉ khác text mô tả
	htmlVN := `<div>Mã xác nhận:</div><span style="letter-spacing: 5px;">88991</span>`
	if got := ExtractCode(htmlVN); got != "88991" {
		t.Errorf("VN HTML body: got %q, want %q", got, "88991")
	}

	htmlES := `<div>Tu código:</div><span style="letter-spacing:5px;">42424</span>`
	if got := ExtractCode(htmlES); got != "42424" {
		t.Errorf("ES HTML body: got %q, want %q", got, "42424")
	}
}

// TestExtractCode_PlainTextBody — đảm bảo extract từ plain text body khi không có HTML.
func TestExtractCode_PlainTextBody(t *testing.T) {
	// textBody từ API thường có dạng:
	textBody := `========================================
57603Don't share this code with anyone.
57603 is your confirmation code

Thanks, Facebook`
	if got := ExtractCode(textBody); got != "57603" {
		t.Errorf("plain text body: got %q, want %q", got, "57603")
	}
}

// TestExtractCode_BodyWhenSubjectEmpty — simulate khi subject trống/non-OTP, body MUST cover.
func TestExtractCode_BodyWhenSubjectEmpty(t *testing.T) {
	// Subject không chứa code
	subject := "Notification from Facebook"
	if got := ExtractCode(subject); got != "" {
		t.Logf("subject extracted (unexpected but ok): %q", got)
	}
	// Body phải vẫn extract được
	htmlBody := `<span style="letter-spacing: 5px;">12345</span>`
	if got := ExtractCode(htmlBody); got != "12345" {
		t.Errorf("body extraction must work when subject empty: got %q", got)
	}
}
