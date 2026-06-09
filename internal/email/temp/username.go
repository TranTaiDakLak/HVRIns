// username.go — Helper sinh local-part email "trông giống thật" cho các temp mail provider.
//
// Trước đây các provider (wemakemail, byomde, dinlaan, ...) đều ghép `randomString(10) + "@domain"`,
// cho ra dạng "vvpzrzxhkm@workshopbmt.com" — nhìn rõ là bot, dễ bị flag khi verify Facebook.
//
// Helper này dùng `fakeinfo.EmailFromProfile` (đã có sẵn ở fakeinfo/builder.go với 30+ pattern
// như "john.smith22051990ab3c", "smith_john1990xyz4") — sinh tên + ngày sinh giả → ghép thành
// local-part trông giống email thật.
//
// Dùng:
//
//	w.email = realisticEmail(domain)            // "john.smith22051990ab3c@workshopbmt.com"
//	w.user  = realisticLocalPart()              // "john.smith22051990ab3c" (không có domain)
package temp

import (
	"strings"

	"HVRIns/internal/instagram/fakeinfo"
)

// realisticEmail sinh email với local-part dạng tên thật + domain user truyền vào.
// domain có thể có hoặc không có @ ở đầu (helper tự xử lý).
func realisticEmail(domain string) string {
	domain = strings.TrimPrefix(strings.TrimSpace(domain), "@")
	if domain == "" {
		domain = "gmail.com"
	}
	p := fakeinfo.RandomFakeProfile()
	return fakeinfo.EmailFromProfile(p.FirstName, p.LastName, p.Birthday, "@"+domain)
}

// realisticLocalPart trả phần trước @ — dùng cho provider tạo username trước rồi mới ghép domain.
func realisticLocalPart() string {
	full := realisticEmail("placeholder.local")
	if idx := strings.LastIndex(full, "@"); idx > 0 {
		return full[:idx]
	}
	return full
}
