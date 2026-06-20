// Package result — ghi file kết quả chạy reg/verify vào thư mục Result folder.
//
// Port từ C# FMain.SaveFile + SaveFileUpsertByUid.
//
// Mục tiêu:
//   - Thread-safe khi nhiều goroutine worker ghi song song
//   - Per-file mutex để 2 file khác nhau ghi parallel không block nhau
//   - Lazy mkdir: chỉ tạo thư mục khi thật sự ghi
//   - Upsert theo UID (field đầu tiên, phân tách "|") — port trực tiếp C# logic
//
// API chia 2 tầng:
//   store.go  — low-level: AppendLine, UpsertByUID, Overwrite
//   writer.go — high-level: Writer struct gắn với 1 folder root + constants file name
package result

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// fileLocks — per-file mutex map. Tránh lock toàn cục cho mọi file:
// ví dụ SuccessVerify.txt và DieAfterVerify.txt có thể ghi parallel.
var (
	fileLocksMu sync.Mutex
	fileLocks   = make(map[string]*sync.Mutex)
)

// lockFor trả về mutex riêng cho 1 path (tạo mới nếu chưa có).
func lockFor(path string) *sync.Mutex {
	fileLocksMu.Lock()
	defer fileLocksMu.Unlock()
	m, ok := fileLocks[path]
	if !ok {
		m = &sync.Mutex{}
		fileLocks[path] = m
	}
	return m
}

// ensureDir tạo thư mục cha của path nếu chưa tồn tại.
// Gọi trước mỗi lần ghi để lazy-create folder.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("result: create dir %q: %w", dir, err)
	}
	return nil
}

// AppendLine ghi content + "\n" vào cuối file. Tạo file nếu chưa có.
// Port C# FMain.SaveFile(): append UTF-8, không xóa nội dung cũ.
//
// path: đường dẫn file đầy đủ (ví dụ "result/session_abc/SuccessVerify.txt").
// content: 1 dòng (không bao gồm "\n" cuối).
//
// Lỗi I/O bị swallow — caller không nên panic vì file kết quả không ảnh hưởng reg flow.
// Trả về error để caller có thể log nếu muốn; KHÔNG được dùng để abort reg.
func AppendLine(path, content string) error {
	if path == "" {
		return fmt.Errorf("result: AppendLine path empty")
	}

	m := lockFor(path)
	m.Lock()
	defer m.Unlock()

	if err := ensureDir(path); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("result: open %q: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content + "\n"); err != nil {
		return fmt.Errorf("result: write %q: %w", path, err)
	}
	return nil
}

// UpsertByUID ghi dòng vào file — nếu UID (field đầu, tách "|") đã tồn tại thì
// UPDATE dòng đó, ngược lại APPEND dòng mới.
//
// Port trực tiếp C# FMain.SaveFileUpsertByUid(): tránh 1 UID die được ghi
// nhiều lần → duplicate trong DieAfterVerify.txt.
//
// path: đường dẫn file đầy đủ.
// content: dòng dạng "uid|pass|..." — UID phải ở field đầu.
//
// Thuật toán:
//  1. Đọc toàn bộ file
//  2. Scan tìm dòng có UID khớp (split("|")[0])
//  3. Nếu thấy → thay thế dòng đó
//  4. Nếu không → append cuối
//  5. Ghi đè toàn bộ file (atomic: ghi tempfile + rename để tránh corruption)
func UpsertByUID(path, content string) error {
	if path == "" {
		return fmt.Errorf("result: UpsertByUID path empty")
	}

	// Tách UID từ content (field đầu)
	newUID := firstField(content)
	if newUID == "" {
		// Không có UID → fallback sang append thường (không dedup được)
		return AppendLine(path, content)
	}

	m := lockFor(path)
	m.Lock()
	defer m.Unlock()

	if err := ensureDir(path); err != nil {
		return err
	}

	// Đọc dòng hiện có (nếu file tồn tại)
	var lines []string
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 128*1024), 1024*1024)
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
		_ = f.Close()
	}

	// Upsert logic
	replaced := false
	for i, ln := range lines {
		if firstField(ln) == newUID {
			lines[i] = content
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, content)
	}

	// Ghi atomic: tempfile → rename
	return writeAllLinesAtomic(path, lines)
}

// Overwrite ghi đè toàn bộ file bằng content mới.
// Port C# File.WriteAllText — dùng cho các file counter (CountrySuccess.txt...)
// nơi mỗi lần save là rewrite toàn bộ nội dung.
//
// path: file đích.
// content: text đầy đủ, không thêm "\n" tự động.
func Overwrite(path, content string) error {
	if path == "" {
		return fmt.Errorf("result: Overwrite path empty")
	}

	m := lockFor(path)
	m.Lock()
	defer m.Unlock()

	if err := ensureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}

// firstField trả về field đầu tiên (trước dấu "|") sau khi trim space.
// Dùng để extract UID làm khóa upsert.
func firstField(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '|'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// writeAllLinesAtomic ghi toàn bộ lines ra file path theo cách atomic:
// ghi vào tempfile cùng thư mục → rename đè file gốc. Tránh file bị corrupt
// khi app crash giữa lúc ghi.
func writeAllLinesAtomic(path string, lines []string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".result-*.tmp")
	if err != nil {
		return fmt.Errorf("result: create temp: %w", err)
	}
	tmpPath := tmp.Name()

	// Defer cleanup nếu rename fail
	defer func() {
		if _, err := os.Stat(tmpPath); err == nil {
			_ = os.Remove(tmpPath)
		}
	}()

	w := bufio.NewWriter(tmp)
	for _, ln := range lines {
		if _, err := w.WriteString(ln + "\n"); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("result: write temp: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("result: flush temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("result: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("result: rename %q → %q: %w", tmpPath, path, err)
	}
	return nil
}
