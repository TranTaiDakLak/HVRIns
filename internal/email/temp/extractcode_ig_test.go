// extractcode_ig_test.go — kiểm chứng ExtractCode bắt được mã OTP Instagram.
//
// Instagram gửi OTP từ mail.instagram.com với nhiều định dạng subject/body khác nhau.
// Test này đảm bảo hàm ExtractCode dùng chung (moakt.go) không bỏ sót mã IG.
package temp

import "testing"

func TestExtractCodeInstagram(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "IG subject EN",
			content: "123456 is your Instagram code",
			want:    "123456",
		},
		{
			name:    "IG subject EN với đuôi",
			content: "654321 is your Instagram code. Don't share it.",
			want:    "654321",
		},
		{
			name:    "IG body EN brand-first",
			content: `<html><body>Hi, your Instagram code is <b>998877</b> to confirm your account.</body></html>`,
			want:    "998877",
		},
		{
			name:    "IG subject VN",
			content: "112233 là mã Instagram của bạn",
			want:    "112233",
		},
		{
			name:    "IG body có brand Instagram + code",
			content: `<div>Your Instagram code is</div><div>445566</div>`,
			want:    "445566",
		},
		{
			name:    "subject chỉ có số đầu (universal)",
			content: "778899 is your code",
			want:    "778899",
		},
		// Đảm bảo không phá vỡ Facebook cũ
		{
			name:    "FB EN vẫn hoạt động",
			content: "246810 is your confirmation code",
			want:    "246810",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExtractCode(c.content)
			if got != c.want {
				t.Errorf("ExtractCode(%q) = %q, muốn %q", c.content, got, c.want)
			}
		})
	}
}
