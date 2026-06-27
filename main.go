package main

import (
	"context"
	"crypto/md5"
	"embed"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" // pprof endpoint để chẩn đoán CPU/goroutine (localhost:6060)
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	igapp "HVRIns/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

// AppVersion được inject lúc build: -ldflags "-X main.AppVersion=vMM.DD.HH.mm"
var AppVersion = "dev"

// -minimized CLI flag: khởi động window ở trạng thái Minimised → save GPU/CPU cho
// webview khi user không xem UI, vẫn giữ UI functional. Dùng cho chạy 24/7 (Phase 3).
// Cách dùng: HaVu.exe -minimized
var flagMinimized = flag.Bool("minimized", false, "Khởi động app ở trạng thái minimized (tiết kiệm RAM khi chạy 24/7)")

// -pprof CLI flag: bật pprof server ở localhost:6060 để chẩn đoán CPU/goroutine.
// Cách dùng: HaVu.exe -pprof  → rồi `go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=20`
var flagPprof = flag.Bool("pprof", false, "Bật pprof server localhost:6060 (chẩn đoán CPU/goroutine)")

// instanceUniqueID trả về ID duy nhất dựa trên đường dẫn thư mục chứa exe.
// Mỗi thư mục khác nhau → ID khác nhau → nhiều bản copy chạy độc lập song song.
func instanceUniqueID() string {
	exe, err := os.Executable()
	if err != nil {
		return "havu-wemake-2024-unique"
	}
	dir := filepath.Dir(filepath.Clean(exe))
	hash := md5.Sum([]byte(dir))
	return fmt.Sprintf("havu-%x", hash[:6])
}

func main() {
	flag.Parse()

	// GC tuning: reg chạy nhiều luồng cấp phát JSON liên tục → GC mặc định (GOGC=100)
	// chạy rất thường xuyên, tốn CPU. Tăng lên 300 → GC chạy ~1/3 tần suất → giảm CPU
	// GC đáng kể, đổi bằng RAM (chạy reg dư RAM). Env GOGC vẫn override nếu user set.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(300)
	}

	// pprof: chỉ bật khi -pprof → không ảnh hưởng run thường. Lỗi bind (port bận / chạy
	// nhiều bản) bỏ qua. Dùng để chẩn đoán CHÍNH XÁC hàm nào ăn CPU khi reg nhiều luồng.
	if *flagPprof {
		go func() { _ = http.ListenAndServe("127.0.0.1:6060", nil) }()
	}

	igapp.SetBuildVersion(AppVersion)

	// Chuyển CWD sang data dir ngay đầu — tất cả relative path (Config/, logs/) tính từ đây.
	// Dev: bin/dev/  |  Production: thư mục chứa exe. Xem internal/app/datadir.go.
	if err := os.Chdir(igapp.AppDataDir()); err != nil {
		println("Warning: cannot chdir to data dir:", err.Error())
	}

	// Mở rộng Windows ephemeral port range từ 16k (mặc định) lên ~48k.
	// Giảm WSAEADDRINUSE khi chạy nhiều thread concurrent qua proxy.
	// netsh int ipv4 set dynamicport tcp start=16384 num=49152
	igapp.ExpandEphemeralPortRange()

	// Tạo instance ứng dụng
	app := igapp.NewApp()
	app.SetVersion(AppVersion)

	windowStartState := options.Normal
	if *flagMinimized {
		windowStartState = options.Minimised
	}

	// Chạy ứng dụng với cấu hình desktop
	err := wails.Run(&options.App{
		// SingleInstanceLock: chỉ cho phép 1 instance chạy PER THƯ MỤC.
		// UniqueId = hash đường dẫn thư mục chứa exe → 2 bản copy ở 2 thư mục khác nhau
		// sẽ có UniqueId khác nhau → chạy độc lập song song được.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: instanceUniqueID(),
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				app.OnSecondInstance()
			},
		},
		Title:     "Hạ Vũ",
		Width:     1440,
		Height:    900,
		MinWidth:  979,
		MinHeight: 379,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Dark background mặc định (#0f1117)
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 23, A: 1},
		OnStartup:        app.Startup,
		// OnBeforeClose: chặn tắt nhầm khi user nhấn X / Alt+F4 / taskbar close.
		// Trả true → block close + emit event để FE show dialog xác nhận.
		// User confirm trong dialog → FE gọi app.RequestQuit() (set flag + Quit)
		// → OnBeforeClose chạy lại với flag=true → trả false → app close bình thường.
		OnBeforeClose: func(_ context.Context) bool {
			if app.IsConfirmedQuit() {
				return false // user đã confirm → cho phép close
			}
			app.EmitQuitConfirm() // emit event để FE show ConfirmDialog
			return true            // block close
		},
		WindowStartState: windowStartState,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    true,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Lỗi khởi động:", err.Error())
	}
}
