// writer.go — Writer struct gắn root path + constants tên file chuẩn C#.
package result

import (
	"path/filepath"
	"strings"
)

// File name constants — port chính xác từ C# FMain.cs (giữ hoa/thường theo C#).
// User quen format C# copy-paste thấy file quen thuộc.
const (
	// Register
	FileSuccessReg               = "SuccessReg.txt"       // uid|pass|cookie|token|time|country|NVR
	FileSuccessNVRPhone          = "SuccessNVR_Phone.txt" // phone đã reg
	FileSuccessNVREmail          = "SuccessNVR_Email.txt" // email đã reg
	FileCheckpoint               = "Checkpoint.txt"       // reg bị checkpoint
	FileBlocked                  = "Blocked.txt"          // reg bị FB block
	FileUnknownBlockType         = "UnknownReg.txt"       // reg unknown (block không xác định) — tách riêng khỏi verify unknown
	FileSuccessButErrorCheckLive = "Success_but_error_checklive.txt"
	FileLive                     = "Live.txt" // reg success + check-live xác nhận CÒN SỐNG (sau delay)

	// Verify — gộp Die/Unknown về tên ngắn để folder output clean hơn
	FileSuccessVerify               = "SuccessVerify.txt"                // uid|pass|2fa|cookie|token|email|fullname|time|country
	FileSuccessVerifyNo2FA          = "SuccessVerify_No2FA.txt"          // verify ok nhưng chưa bật 2FA
	FileSuccessVerifyFailedDeactive = "SuccessVerify_FailedDeactive.txt" // verify ok nhưng deactive fail
	FileDieAfterVerify              = "Die.txt"                          // UPSERT by UID (was DieAfterVerify.txt)
	FileUnknownErrorCheckLiveDieApi = "Unknown.txt"                      // verify unknown (was UnknownErrorCheckLiveDieApi.txt)

	// Error details
	FileUnknownErrorApi      = "UnknownErrorApi.txt"
	FileCfemUnknownErrorApi  = "CfemUnknownErrorApi.txt"
	FileChinaMailCantGetCode = "ChinaMail_CantGetCode.txt" // mail service (rent mail TQ) không trả code
	FileBuyMailCantGetCode   = "BuyMail_CantGetCode.txt"
	FileNotTokenIn           = "not_token_in.txt"
	FileErrorData            = "errordata.txt"
	FileRemainData           = "RemainData.txt"

	// Counter tracking (Phase B — auto-save mỗi 5-10s, overwrite)
	FileFbAppVersionSuccess = "FbAppVersisonSuccess.txt" // Note: typo "Verison" giữ giống C#
)

// Writer quản lý việc ghi kết quả vào 1 thư mục root.
// Tạo 1 Writer cho mỗi session chạy (mỗi lần Start), pass vào verify/reg flows.
//
// Thread-safe: per-file mutex trong store.go.
// Zero value không hợp lệ — dùng NewWriter().
type Writer struct {
	root string
}

// NewWriter tạo Writer mới với root folder.
// root: ví dụ "C:/Users/Admin/Documents/NullCoreSummer/KetQua/RegAndroid_20260418_103015/".
// Không mkdir ngay — để lazy tạo khi ghi file đầu.
func NewWriter(root string) *Writer {
	return &Writer{root: strings.TrimRight(root, `/\`)}
}

// Root trả về đường dẫn folder root của Writer.
func (w *Writer) Root() string {
	return w.root
}

// Path trả về đường dẫn đầy đủ của 1 file trong folder.
// filename có thể là constant (FileSuccessVerify) hoặc name động (VerifyFailed_<code>.txt).
func (w *Writer) Path(filename string) string {
	if w == nil || w.root == "" {
		return filename
	}
	return filepath.Join(w.root, filename)
}

// Append ghi dòng vào file filename. Lỗi I/O bị swallow (trả về error để log
// nhưng caller nên bỏ qua — không fail reg vì ghi result fail).
func (w *Writer) Append(filename, line string) error {
	return AppendLine(w.Path(filename), line)
}

// UpsertUID ghi upsert theo UID (field đầu tách "|"). Dùng cho DieAfterVerify.txt.
func (w *Writer) UpsertUID(filename, line string) error {
	return UpsertByUID(w.Path(filename), line)
}

// Overwrite ghi đè toàn bộ file bằng content mới. Dùng cho counter files.
func (w *Writer) Overwrite(filename, content string) error {
	return Overwrite(w.Path(filename), content)
}

// ── Dynamic filename helpers ──────────────────────────────────────────────────
// Các file đặt tên động theo status code: VerifyFailed_<status>.txt, etc.

// VerifyFailedFile tạo tên file theo status code (C#: VerifyFailed_{status}.txt).
func VerifyFailedFile(status string) string {
	return "VerifyFailed_" + sanitizeFilename(status) + ".txt"
}

// LoginFbFailedFile C#: LoginFbFailed_{status}.txt
func LoginFbFailedFile(status string) string {
	return "LoginFbFailed_" + sanitizeFilename(status) + ".txt"
}

// CfemLoginFbFailedFile C#: Cfem_LoginFbFailed_{status}.txt (confirm-email fail).
func CfemLoginFbFailedFile(status string) string {
	return "Cfem_LoginFbFailed_" + sanitizeFilename(status) + ".txt"
}

// LoginGmailFile C#: LoginGmail_{status}.txt
func LoginGmailFile(status string) string {
	return "LoginGmail_" + sanitizeFilename(status) + ".txt"
}

// SuccessVerifyUGFile chứa UA của account verify thành công.
// instanceName đã bị bỏ — tất cả platform dùng chung 1 file (user request: không suffix _s23/_api...).
func SuccessVerifyUGFile(_ string) string {
	return "SuccessVerifyUG.txt"
}

// SuccessRegNVRUGFile chứa UA của account reg NVR thành công.
func SuccessRegNVRUGFile(_ string) string {
	return "SuccessRegNVRUG.txt"
}

// sanitizeFilename bỏ các ký tự không hợp lệ cho tên file Windows/Linux.
// Dùng cho status code (thường alphanumeric) — chỉ defensive.
func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
