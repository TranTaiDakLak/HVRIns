package secapi

import "strings"

// MaskEmail — mapping C#: StringUtils.MaskEmail().
//   abc@gmail.com   → a*c@gmail.com   (community domains giữ nguyên domain)
//   user@company.io → u**r@c******.io (domain khác cũng mask)
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	domain := parts[1]

	// Mask local: giữ ký tự đầu và cuối
	var maskedLocal string
	if len(local) <= 2 {
		maskedLocal = local
	} else {
		maskedLocal = string(local[0]) + strings.Repeat("*", len(local)-2) + string(local[len(local)-1])
	}

	// Community domains (gmail, hotmail, yahoo...) giữ nguyên domain
	communityDomains := []string{"gmail.com", "hotmail.com", "yahoo.com", "outlook.com", "icloud.com", "live.com", "protonmail.com"}
	for _, cd := range communityDomains {
		if strings.EqualFold(domain, cd) {
			return maskedLocal + "@" + domain
		}
	}

	// Domain khác: mask middle
	dotIdx := strings.Index(domain, ".")
	if dotIdx <= 1 {
		return maskedLocal + "@" + domain
	}
	suffix := domain[dotIdx:]
	maskedDomain := string(domain[0]) + strings.Repeat("*", dotIdx-1) + suffix
	return maskedLocal + "@" + maskedDomain
}
