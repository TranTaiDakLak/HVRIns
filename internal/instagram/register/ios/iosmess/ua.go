package iosmess

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

// iosDev — pool device iPhone (FBDV) + iOS version (FBSV) cho UA Messenger Lite.
type iosDev struct{ fbdv, fbsv, fbss string }

var iosDevs = []iosDev{
	{"iPhone9,1", "15.8.4", "2"},   // iPhone 7
	{"iPhone9,3", "15.8.3", "2"},   // iPhone 7
	{"iPhone10,1", "16.7.10", "2"}, // iPhone 8
	{"iPhone10,4", "16.7.10", "2"}, // iPhone 8
	{"iPhone10,5", "16.7.10", "3"}, // iPhone 8 Plus
	{"iPhone11,2", "16.7.10", "3"}, // iPhone XS
	{"iPhone11,8", "16.7.10", "2"}, // iPhone XR
	{"iPhone12,1", "16.7.10", "2"}, // iPhone 11
	{"iPhone12,8", "16.7.10", "2"}, // iPhone SE2
	{"iPhone13,2", "16.7.10", "3"}, // iPhone 12
}

var uaLocales = []string{"vi_VN", "vi_VN", "vi_VN", "en_US"}

// messAppBuild — cặp (FBAV, FBBV) MessengerLite iOS lấy từ pool config.
type messAppBuild struct{ fbav, fbbv string }

// messBuildFallback — giá trị 563 đang chạy (gắn với doc_id trong body templates).
// Dùng khi file pool thiếu/rỗng để KHÔNG bao giờ tạo UA vô nghĩa.
var messBuildFallback = []messAppBuild{{"563.0.0.27.106", "980221516"}}

var (
	messBuildMu    sync.Mutex
	messBuildCache = map[string][]messAppBuild{}
)

// loadMessIOSBuilds đọc pool (FBAV,FBBV) từ Config/DeviceInfoIOS/mess_ios_app_builds_<kind>.txt
// (fallback mess_ios_app_builds.txt → hardcode 563). Format mỗi dòng: FBAV|FBBV|FBRV.
// Cache theo kind ("reg"/"ver").
func loadMessIOSBuilds(kind string) []messAppBuild {
	messBuildMu.Lock()
	defer messBuildMu.Unlock()
	if p, ok := messBuildCache[kind]; ok {
		return p
	}
	base := uabuilder.GetConfigBaseDir()
	path := filepath.Join(base, "DeviceInfoIOS", "mess_ios_app_builds_"+kind+".txt")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(base, "DeviceInfoIOS", "mess_ios_app_builds.txt")
	}
	var out []messAppBuild
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) < 2 {
				continue
			}
			fbav, fbbv := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if fbav == "" || fbbv == "" {
				continue
			}
			out = append(out, messAppBuild{fbav: fbav, fbbv: fbbv})
		}
		f.Close()
	}
	if len(out) == 0 {
		out = messBuildFallback
	}
	messBuildCache[kind] = out
	return out
}

// buildUA — UA Messenger Lite iOS random device/locale + (FBAV,FBBV) từ pool theo kind.
// kind="reg" cho register, "ver" cho verify. doc_id trong body templates đang khoá ở 563:
// nếu pool có version khác, UA sẽ lệch nhẹ so với doc_id (FB thường chấp nhận — như FB4A).
func buildUA(r *rand.Rand, kind string) string {
	d := iosDevs[r.Intn(len(iosDevs))]
	loc := uaLocales[r.Intn(len(uaLocales))]
	pool := loadMessIOSBuilds(kind)
	ab := pool[r.Intn(len(pool))]
	return fmt.Sprintf("LightSpeed [FBAN/MessengerLiteForiOS;FBAV/%s;FBBV/%s;FBDV/%s;FBMD/iPhone;FBSN/iOS;FBSV/%s;FBSS/%s;FBCR/;FBID/phone;FBLC/%s;FBOP/0]",
		ab.fbav, ab.fbbv, d.fbdv, d.fbsv, d.fbss, loc)
}

// RandomUA — entry cho RegisterPlatformVerifyUA (hiển thị "UA gốc"): deterministic theo country
// để hiển thị nhất quán. Pool dùng "ver".
func RandomUA(country string) string {
	return buildUA(rand.New(rand.NewSource(int64(len(country))+9876543210)), "ver")
}

// RandomMessUA — UA per-account cho verify (rotate qua pool "ver" mỗi lần gọi).
func RandomMessUA() string {
	return buildUA(rand.New(rand.NewSource(time.Now().UnixNano()+rand.Int63())), "ver")
}
