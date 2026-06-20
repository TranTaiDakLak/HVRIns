// errorlog.go — ghi exception / runtime error vào errordata.txt trong run folder.
//
// Port từ C# FMain.SaveFile(SavePath + "\\errordata.txt", data) — log các lỗi
// không expect (exception từ worker, response malformed) để user debug.
//
// Khác với %APPDATA%/logs/ (log toàn app qua slog), errordata.txt scope theo
// từng session run — user mở folder kết quả thấy ngay lỗi của session đó.
package result

import (
	"fmt"
	"runtime/debug"
	"time"
)

// LogError ghi 1 entry error vào errordata.txt kèm timestamp + stack.
//
// context: mô tả ngữ cảnh (VD "verify worker slot=3 uid=100").
// err: error object (có thể nil — khi đó chỉ log context).
//
// Format entry (xuống dòng đôi giữa các entry):
//
//	[2026-04-18 10:30:15] {context}
//	  err: {err.Error()}
//	  stack:
//	  {stack trace}
//
// writer nil → no-op.
func (w *Writer) LogError(context string, err error) error {
	if w == nil || w.root == "" {
		return nil
	}
	var entry string
	ts := time.Now().Format("2006-01-02 15:04:05")
	if err != nil {
		entry = fmt.Sprintf("[%s] %s\n  err: %v\n", ts, context, err)
	} else {
		entry = fmt.Sprintf("[%s] %s\n", ts, context)
	}
	// Append trailing blank line để tách entry
	entry += "\n"
	return w.Append(FileErrorData, entry)
}

// RecordPanic ghi panic stack vào errordata.txt.
// Lưu ý Go spec: recover() phải được gọi TRỰC TIẾP trong deferred function,
// nên caller phải tự recover() rồi pass giá trị vào đây thay vì để RecordPanic
// tự recover.
//
// recovered: giá trị từ recover() ở defer. nil → no-op.
// context: mô tả goroutine/worker để phân biệt trong log.
//
// Usage chuẩn:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        writer.RecordPanic(r, "worker slot=3 uid="+uid)
//	    }
//	}()
//
// writer nil hoặc recovered nil → no-op.
func (w *Writer) RecordPanic(recovered any, context string) {
	if recovered == nil {
		return
	}
	if w == nil || w.root == "" {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] PANIC %s\n  recovered: %v\n  stack:\n%s\n\n",
		ts, context, recovered, debug.Stack())
	_ = w.Append(FileErrorData, entry)
}
