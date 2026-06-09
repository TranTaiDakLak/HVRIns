// seed_config.go — Tạo cấu trúc thư mục Config/ khi chạy lần đầu.
// Không còn seed file content từ embedded data — tất cả data do user quản lý trong Config/.
package fakeinfo

import (
	"os"
	"path/filepath"
)

// SeedConfigDataIfMissing tạo cấu trúc thư mục Config/ và các file placeholder rỗng.
// Được app.go gọi 1 lần khi khởi động.
//
// baseDir: root Config folder (thường là "Config" cạnh exe).
// Không ghi đè file đã tồn tại.
func SeedConfigDataIfMissing(baseDir string) error {
	if baseDir == "" {
		baseDir = "Config"
	}

	// Tạo các thư mục cần thiết
	dirs := []string{
		"Namereg/US",
		"Namereg/VN",
		"SimNetwork",
		"Locales",
		"UserAgent",
		"DeviceInfo",     // Android device pool: devices.txt, carriers.txt, locales.txt, etc.
		"DeviceInfoIOS",  // iOS device pool (chuẩn bị cho UA builder iOS)
		"DeviceInfoPC",   // PC device pool (chuẩn bị cho UA builder PC/desktop)
		"Fbapp",          // versions_and_builds.txt — pool FBAV/FBBV cho Android FB4A
		"PhoneDatabase",
		"phone_database",
		"Permanent",
		"Cookie",
	}
	for _, d := range dirs {
		_ = os.MkdirAll(filepath.Join(baseDir, d), 0755)
	}

	// Proxy folder + placeholder files (user tự paste proxy vào)
	proxyDir := filepath.Join(baseDir, "Proxy")
	_ = os.MkdirAll(proxyDir, 0755)
	for _, f := range []string{"proxy_tempmail.txt", "proxy_rentmail.txt"} {
		p := filepath.Join(proxyDir, f)
		if _, err := os.Stat(p); err != nil {
			_ = os.WriteFile(p, []byte("# Mỗi dòng 1 proxy (host:port:user:pass hoặc http://user:pass@host:port)\n"), 0644)
		}
	}

	// Permanent phone/mail placeholder
	permDir := filepath.Join(baseDir, "Permanent")
	for _, f := range []string{"phone.txt", "mail.txt"} {
		p := filepath.Join(permDir, f)
		if _, err := os.Stat(p); err != nil {
			kind := f[:len(f)-4]
			_ = os.WriteFile(p, []byte("# Mỗi dòng 1 "+kind+" đã có sẵn (dùng khi Phone × Mail = Random File)\n"), 0644)
		}
	}

	// TempMail domains placeholder
	tempmailDir := filepath.Join(baseDir, "TempMail")
	_ = os.MkdirAll(tempmailDir, 0755)
	domainsPath := filepath.Join(tempmailDir, "domains.txt")
	if _, err := os.Stat(domainsPath); err != nil {
		_ = os.WriteFile(domainsPath, []byte("# Mỗi dòng 1 domain tempmail (ưu tiên dùng trước default provider)\n# Vd: tmpbox.net\n"), 0644)
	}

	return nil
}
