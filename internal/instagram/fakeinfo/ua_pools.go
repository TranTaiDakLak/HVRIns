// ua_pools.go — Prebuilt User-Agent pools (file-based mode).
//
// Port từ C# ConfigFileUserAgentBuilder: thay vì ghép UA động từ device+version,
// một số luồng dùng UA prebuilt từ file để đảm bảo UA đã test ok trên production.
//
// Mapping C# → Go:
//
//	config/useragent/Android_UG.txt → ua_android_pool.txt   (1625 UA FB4A)
//	config/useragent/iOS_UG.txt     → ua_ios_pool.txt       (2216 UA FBIOS)
//	config/useragent/PC_UG.txt      → ua_request_pool.txt   (420 UA Chrome Desktop — label "PC")
//	config/useragent/WebChrome_UA.txt → ua_webchrome_pool.txt (2626 UA Chrome Mobile — label "WebMobile")
//
// Pattern "embed + override" giống fbdata:
//   - Mặc định dùng pool embed
//   - User có thể override qua Config/UserAgent/<name>.txt
package fakeinfo

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UAPoolKind phân loại pool để dispatch đúng biến khi Random.
type UAPoolKind string

const (
	UAKindAndroid     UAPoolKind = "android"      // FB4A native Android — label "Android"
	UAKindIOS         UAPoolKind = "ios"          // FBIOS iPhone — label "iOS"
	UAKindRequest     UAPoolKind = "request"      // Chrome Desktop (Win/Mac) — label "PC" — dùng cho api mfb (web desktop)
	UAKindWebChrome   UAPoolKind = "webchrome"    // Chrome Mobile Android — label "WebMobile" — dùng cho api web andr
	UAKindAndroidMess UAPoolKind = "android_mess" // Messenger (Orca) Android — dùng cho appmessv3 reg/ver
	UAKindIOSMess     UAPoolKind = "ios_mess"     // Messenger Lite iOS — dùng cho iosmess reg/ver
)

// UAOverrideDir là thư mục user override UA pools (tương đương C# config/useragent/).
const UAOverrideDir = "Config/UserAgent"

var uaOverrideFiles = map[UAPoolKind]string{
	UAKindAndroid:     "Android_UG.txt",
	UAKindIOS:         "iOS_UG.txt",
	UAKindRequest:     "PC_UG.txt", // label "PC" — Chrome Desktop UAs (đổi từ Request_UG.txt 2026-05)
	UAKindWebChrome:   "WebChrome_UA.txt",
	UAKindAndroidMess: "Android_Mess.txt", // Messenger Orca Android UA pool
	UAKindIOSMess:     "iOS_Mess.txt",     // Messenger Lite iOS UA pool
}

var (
	uaMu        sync.RWMutex
	uaPools     = make(map[UAPoolKind][]string) // kind → list UA active (default | override)
	uaOverrides = make(map[UAPoolKind]bool)     // kind → có dùng override không
)

func init() {
	// Thử load từ Config/UserAgent/ ngay lúc khởi động.
	// Nếu file chưa tồn tại (lần đầu chạy) → pool rỗng tạm thời;
	// app.go sẽ seed file rồi gọi ReloadUAPools() để nạp đúng.
	loadUAPool(UAKindAndroid)
	loadUAPool(UAKindIOS)
	loadUAPool(UAKindRequest)
	loadUAPool(UAKindWebChrome)
	loadUAPool(UAKindAndroidMess)
	loadUAPool(UAKindIOSMess)
}

// loadUAPool đọc pool chỉ từ Config/UserAgent/<file> — không dùng embed làm default.
// Embed data là seed-only (SeedUAFilesIfMissing); nguồn dữ liệu chạy thật là file trên disk.
// Pool rỗng khi file không tồn tại hoặc không có dòng hợp lệ — caller xử lý trường hợp này.
func loadUAPool(kind UAPoolKind) {
	overridePath := filepath.Join(UAOverrideDir, uaOverrideFiles[kind])

	uaMu.Lock()
	defer uaMu.Unlock()

	if userData, err := os.ReadFile(overridePath); err == nil {
		if userList := splitNonEmptyLines(string(userData)); len(userList) > 0 {
			uaPools[kind] = userList
			uaOverrides[kind] = true
			return
		}
	}
	uaPools[kind] = nil
	uaOverrides[kind] = false
}

// ReloadUAPools force reload tất cả pool từ Config/UserAgent/ disk.
// Gọi sau khi user save file qua UI, edit thủ công, hoặc sau khi SeedUAFilesIfMissing.
func ReloadUAPools() {
	loadUAPool(UAKindAndroid)
	loadUAPool(UAKindIOS)
	loadUAPool(UAKindRequest)
	loadUAPool(UAKindWebChrome)
	loadUAPool(UAKindAndroidMess)
	loadUAPool(UAKindIOSMess)
}

// AppendUAToPool ghi 1 UA mới vào file override + add vào memory pool.
// Idempotent: nếu UA đã tồn tại → bỏ qua.
// Dùng cho learning loop: verify thành công với UA nào thì append lại pool đó.
// Trả true nếu UA mới được thêm, false nếu trùng/skip/lỗi.
func AppendUAToPool(kind UAPoolKind, ua string) bool {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return false
	}
	overrideFile, ok := uaOverrideFiles[kind]
	if !ok {
		return false
	}

	uaMu.Lock()
	// Check dup trong memory pool
	for _, existing := range uaPools[kind] {
		if existing == ua {
			uaMu.Unlock()
			return false
		}
	}
	// Append vào memory
	uaPools[kind] = append(uaPools[kind], ua)
	uaOverrides[kind] = true
	uaMu.Unlock()

	// Append vào file (best-effort, không lock — file I/O concurrent OS-level)
	if err := os.MkdirAll(UAOverrideDir, 0755); err != nil {
		return true // vẫn có trong memory
	}
	path := filepath.Join(UAOverrideDir, overrideFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return true
	}
	defer f.Close()
	_, _ = f.WriteString(ua + "\n")
	return true
}

// RandomUAFromPool lấy UA ngẫu nhiên từ pool tương ứng kind.
// Trả về "" nếu pool rỗng (caller có thể fallback về BuildAndroidUA động).
func RandomUAFromPool(kind UAPoolKind) string {
	uaMu.RLock()
	pool := uaPools[kind]
	uaMu.RUnlock()
	if len(pool) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	return pool[r.Intn(len(pool))]
}

// UAPoolSize trả về số UA đang active cho kind.
func UAPoolSize(kind UAPoolKind) int {
	uaMu.RLock()
	defer uaMu.RUnlock()
	return len(uaPools[kind])
}

// UAPoolAll trả về copy danh sách UA đang active (default hoặc override nếu có).
// Frontend dùng để fill textarea với data gốc khi user chưa override.
func UAPoolAll(kind UAPoolKind) []string {
	uaMu.RLock()
	defer uaMu.RUnlock()
	src := uaPools[kind]
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// UAPoolOverrideActive trả về true nếu kind đang dùng file override (thay vì embed).
func UAPoolOverrideActive(kind UAPoolKind) bool {
	uaMu.RLock()
	defer uaMu.RUnlock()
	return uaOverrides[kind]
}

// EnsureUAOverrideDir tạo thư mục Config/UserAgent/ nếu chưa tồn tại.
func EnsureUAOverrideDir() error {
	return os.MkdirAll(UAOverrideDir, 0755)
}

// SeedUAFilesIfMissing seed tất cả UA files từ embedded data vào Config/UserAgent/
// khi chưa tồn tại. Gọi từ app.go lúc khởi động để user thấy file sẵn, có thể edit.
// SeedUAFilesIfMissing tạo thư mục Config/UserAgent/ nếu chưa tồn tại.
// Không còn seed content từ embed — user tự quản lý các file UA pool.
func SeedUAFilesIfMissing(dir string) error {
	if dir == "" {
		dir = UAOverrideDir
	}
	return os.MkdirAll(dir, 0755)
}

// UAOverridePath trả về đường dẫn override file tương ứng kind.
// Frontend dùng để hiển thị path user có thể edit.
func UAOverridePath(kind UAPoolKind) string {
	return filepath.Join(UAOverrideDir, uaOverrideFiles[kind])
}

// splitNonEmptyLines chia text thành các dòng, bỏ dòng rỗng/whitespace.
func splitNonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
