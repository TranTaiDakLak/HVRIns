package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// === FILE DIALOG ===

// OpenTextFileDialog mở dialog chọn file text, trả về nội dung file
func (a *App) OpenTextFileDialog() string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn file tài khoản",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return ""
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(data)
}

// OpenFileDialogPath mở dialog chọn file text, trả về PATH (không đọc nội dung)
func (a *App) OpenFileDialogPath() string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn file User Agent",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

// ReadTextFile đọc nội dung file text từ path, trả về nội dung hoặc "" nếu lỗi
func (a *App) ReadTextFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return string(data)
}

// ValidatePath kiểm tra đường dẫn có tồn tại và là thư mục không
// Trả về "" nếu hợp lệ, thông báo lỗi nếu không hợp lệ
func (a *App) ValidatePath(path string) string {
	if path == "" {
		return "Chưa chọn thư mục"
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "Thư mục không tồn tại: " + path
	}
	if err != nil {
		return "Lỗi kiểm tra đường dẫn: " + err.Error()
	}
	if !info.IsDir() {
		return "Đường dẫn không phải thư mục: " + path
	}
	return ""
}

// ValidateFilePath kiểm tra path là FILE tồn tại (ngược với ValidatePath chỉ accept folder).
// Dùng cho AccountSource="file" mode — user pick 1 file .txt cụ thể.
func (a *App) ValidateFilePath(path string) string {
	if path == "" {
		return "Chưa chọn file"
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "File không tồn tại: " + path
	}
	if err != nil {
		return "Lỗi kiểm tra file: " + err.Error()
	}
	if info.IsDir() {
		return "Đường dẫn là thư mục, cần chọn file: " + path
	}
	return ""
}

// OpenFolderDialog mở dialog chọn thư mục, trả về đường dẫn
func (a *App) OpenFolderDialog() string {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn thư mục lưu",
	})
	if err != nil {
		return ""
	}
	return folder
}

// OpenFolderInExplorer mở thư mục trong Windows Explorer.
// Sanitize path: phải là absolute, normalize, và tồn tại — tránh command injection
// qua UNC path (\\attacker\share) hoặc explorer-flag arguments (/root,/e,...).
func (a *App) OpenFolderInExplorer(path string) string {
	if path == "" {
		return "Chưa chọn thư mục"
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "Đường dẫn phải là absolute"
	}
	// Reject UNC path (\\server\share) — không phải thư mục local an toàn để mở
	if strings.HasPrefix(clean, `\\`) {
		return "Đường dẫn UNC không được hỗ trợ"
	}
	// Reject path bắt đầu bằng "/" hoặc "-" trên Windows (có thể bị parse như flag)
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "-") {
		return "Đường dẫn không hợp lệ"
	}
	// Verify path tồn tại + là directory trước khi exec
	info, err := os.Stat(clean)
	if err != nil {
		return "Thư mục không tồn tại: " + clean
	}
	if !info.IsDir() {
		return "Đường dẫn không phải là thư mục: " + clean
	}
	if err := exec.Command("explorer", clean).Start(); err != nil {
		return fmt.Errorf("không mở được thư mục: %w", err).Error()
	}
	return ""
}

// OpenConfigFolder mở thư mục Config trong Windows Explorer.
func (a *App) OpenConfigFolder() string {
	return a.OpenFolderInExplorer(filepath.Join(AppDataDir(), "Config"))
}

// GetDefaultResultPath trả về đường dẫn thư mục kết quả mặc định sẽ được dùng
// nếu user chưa cấu hình ResultFolderPath qua UI.
//
// Logic fallback giống defaultResultFolder() (dùng khi Start Run):
//  1. {exe_dir}/KetQua/ — tạo nếu có quyền ghi
//  2. Fallback ~/Documents/HVR/KetQua/ nếu exe_dir read-only
//
// Gọi từ frontend lúc mount để:
//   - Hiển thị placeholder ở field "Result folder"
//   - Mở đúng folder khi user bấm nút "Mở thư mục" trước lần Start đầu tiên
//     (thay vì bật dialog chọn folder)
//
// Hàm này KHÔNG tạo thư mục — chỉ resolve path. Folder sẽ được tạo lần đầu Run.
func (a *App) GetDefaultResultPath() string {
	return defaultResultFolder()
}
