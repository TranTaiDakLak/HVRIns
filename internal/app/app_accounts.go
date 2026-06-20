package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	resultpkg "HVRIns/internal/result"
	appsettings "HVRIns/internal/settings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Account struct {
	ID           int    `json:"id"`
	UID          string `json:"uid"`
	FullData     string `json:"fullData"` // Dữ liệu gốc (dòng import nguyên bản)
	Password     string `json:"password"`
	Twofa        string `json:"twofa"`
	Email        string `json:"email"`
	PassMail     string `json:"passMail"`
	MailRecovery string `json:"mailRecovery"`
	Cookie       string `json:"cookie"`
	Token        string `json:"token"`
	Status       string `json:"status"`
	Checkpoint   string `json:"checkpoint"`
	StatusAds    string `json:"statusAds"`
	BM           string `json:"bm"`
	TKQC         string `json:"tkqc"`
	ChatSupport  string `json:"chatSupport"`
	FullName     string `json:"fullName"`
	Location     string `json:"location"`
	Avatar       string `json:"avatar"`
	Cover        string `json:"cover"`
	Phone        string `json:"phone"`
	Proxy        string `json:"proxy"`
	UserAgent    string `json:"userAgent"`
	Note         string `json:"note"`
	NoteRun      string `json:"noteRun"`
	ImportTime   string `json:"importTime"`
	Category     string `json:"category"`
	LastRun      string `json:"lastRun"`
	Activity     string `json:"activity"`
	SourceCode   string `json:"sourceCode"`
	CategoryID   *int   `json:"categoryId"`
	// EmailMeta — TempMail provider creds (JSON-encoded) để verify Restore
	// và đọc OTP từ inbox đã có sẵn (skip CreateEmail + skip AddEmail step).
	// Empty cho mode Phone/Mail (giả) → verify dùng flow CreateEmail mới.
	EmailMeta string `json:"emailMeta,omitempty"`

	// iOS partial reg tokens — được lưu vào file format với prefix SRN:/SCUID:
	// để file-based verify (tab ver thủ công) dùng được thay vì chỉ inline auto-verify.
	Srnonce               string `json:"srnonce,omitempty"`
	SessionlessCryptedUID string `json:"sessionlessCryptedUID,omitempty"`
}

type AccountFilter struct {
	Keyword    string `json:"keyword"`
	Status     string `json:"status"`
	CategoryID *int   `json:"categoryId"`
	SortBy     string `json:"sortBy"`
	SortDir    string `json:"sortDir"`
}

type AccountListResult struct {
	Items []Account `json:"items"`
	Total int       `json:"total"`
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Errors   []string `json:"errors"`
}

type DeleteResult struct {
	Deleted int `json:"deleted"`
}

// App struct

func (a *App) removeAccountLine(filePath, lineToRemove string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	removed := false
	target := strings.TrimSpace(lineToRemove)
	for _, l := range lines {
		if !removed && strings.TrimSpace(l) == target {
			removed = true
			continue
		}
		out = append(out, l)
	}
	if err := os.WriteFile(filePath, []byte(strings.Join(out, "\n")), 0644); err != nil {
		slog.Warn("removeAccountLine: ghi file thất bại", "file", filePath, "err", err)
	}
}

// popAccountFromFolder đọc và xóa nguyên tử dòng tài khoản đầu tiên trong thư mục.
// Giữ removeLineMu toàn bộ thao tác đọc + ghi — thread-safe khi nhiều worker gọi đồng thời.
// Trả về ("", "", nil) khi không còn tài khoản nào.
// isVerifiableAccountFile trả về true nếu filename là file chứa account ĐẦY ĐỦ
// (uid|pass|cookie|token|...) mà split-verify có thể parse + verify được.
//
// Whitelist (dùng cho verify input):
//
//	SuccessReg.txt           — account reg thành công (main source)
//	SuccessReg_*.txt         — các variant (vd SuccessReg_S23.txt nếu split theo platform)
//
// Blacklist (bỏ qua — không phải account hoặc đã fail):
//
//	SuccessNVR_Phone.txt     — chỉ chứa phone number
//	SuccessNVR_Email.txt     — chỉ chứa email
//	Blocked.txt              — reg fail, không có cookie
//	Checkpoint.txt           — reg checkpoint
//	Unknown.txt              — reg/verify unknown
//	Live.txt/Die.txt         — output của verify (tránh loop)
//	SuccessVerify*.txt       — verify output (bao gồm SuccessVerifyUG.txt)
//	FbAppVersisonSuccess.txt — counter tracking
//	errordata.txt, RemainData.txt
//
// name: basename của file (không path), ví dụ "SuccessReg.txt".
func isVerifiableAccountFile(name string) bool {
	if !strings.HasSuffix(name, ".txt") {
		return false
	}
	return name == "SuccessReg.txt" || strings.HasPrefix(name, "SuccessReg_")
}

func (a *App) popAccountFromFolder(folderPath string) (line string, filePath string, err error) {
	a.removeLineMu.Lock()
	defer a.removeLineMu.Unlock()

	files, err := filepath.Glob(filepath.Join(folderPath, "*.txt"))
	if err != nil {
		return "", "", fmt.Errorf("đọc thư mục: %w", err)
	}
	for _, fp := range files {
		// Chỉ đọc file account đầy đủ — bỏ qua Phone/Email/UA/Blocked/etc.
		if !isVerifiableAccountFile(filepath.Base(fp)) {
			continue
		}
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			continue
		}
		rawLines := strings.Split(string(content), "\n")
		for i, l := range rawLines {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			// Xóa dòng này ra khỏi file
			out := make([]string, 0, len(rawLines))
			for j, raw := range rawLines {
				if j != i {
					out = append(out, raw)
				}
			}
			if writeErr := os.WriteFile(fp, []byte(strings.Join(out, "\n")), 0644); writeErr != nil {
				slog.Warn("popAccountFromFolder: ghi file thất bại", "file", fp, "err", writeErr)
			}
			return l, fp, nil
		}
	}
	return "", "", nil
}

// popFromFile — đọc và xóa nguyên tử dòng đầu tiên từ một file cụ thể.
// Dùng cho Unknown retry: chỉ đọc từ Unknown.txt, không glob toàn thư mục.
// Trả về "" khi file không tồn tại hoặc hết dòng.
func (a *App) popFromFile(filePath string) string {
	a.removeLineMu.Lock()
	defer a.removeLineMu.Unlock()
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	rawLines := strings.Split(string(content), "\n")
	for i, l := range rawLines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out := make([]string, 0, len(rawLines))
		for j, raw := range rawLines {
			if j != i {
				out = append(out, raw)
			}
		}
		_ = os.WriteFile(filePath, []byte(strings.Join(out, "\n")), 0644)
		return l
	}
	return ""
}

// startFolderWatcher khởi động goroutine poll thư mục nguồn mỗi 3 giây
// Khi tìm thấy account mới → tự động thêm vào memory + emit event lên frontend
func (a *App) startFolderWatcher(folderPath string) {
	// Dừng watcher cũ nếu có — bảo vệ bằng settingsMu
	a.settingsMu.Lock()
	if a.watcherCancel != nil {
		a.watcherCancel()
	}
	if folderPath == "" {
		a.settingsMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.watcherCancel = cancel
	a.settingsMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("folder watcher panic recovered", "panic", r)
			}
		}()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Không load từ folder khi user chọn API mode
				sett := a.LoadSettings()
				if isAPIMode(sett) {
					continue
				}
				result := a.scanAccountFolder(folderPath)
				if result.Imported > 0 {
					runtime.EventsEmit(a.ctx, "accounts:folder-updated", map[string]interface{}{
						"imported": result.Imported,
					})
				}
			}
		}
	}()
}

// === ACCOUNT SOURCE FOLDER ===

// SetAccountSourceFolder — lưu thư mục nguồn (streaming mode: không scan/watcher)
func (a *App) SetAccountSourceFolder(folderPath string) ImportResult {
	a.settingsMu.Lock()
	p := a.appSettings.GetActiveProfile()
	if p != nil {
		p.Account.FolderPath = folderPath
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		slog.Warn("SetAccountSourceFolder: lưu settings thất bại", "err", err)
	}

	// Sync vào general.json để LoadSettings() (dùng trong RunVerify) thấy được giá trị mới.
	const settingsDir = "Config/Settings"
	if b, err := os.ReadFile(filepath.Join(settingsDir, "general.json")); err == nil {
		var existing SettingsData
		if json.Unmarshal(b, &existing) == nil {
			existing.General.AccountSourcePath = folderPath
			if patched, err := json.MarshalIndent(existing, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(settingsDir, "general.json"), patched, 0644)
			}
		}
	}

	// Sync vào interaction.json — VerifySourceFolderPath là alias của AccountSourcePath.
	// Đảm bảo 2 UI (Interaction Setup + Cài đặt chung) luôn hiển thị cùng 1 path.
	if b, err := os.ReadFile(filepath.Join(settingsDir, "interaction.json")); err == nil {
		var ic InteractionConfig
		if json.Unmarshal(b, &ic) == nil {
			ic.VerifySourceFolderPath = folderPath
			if patched, err := json.MarshalIndent(ic, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(settingsDir, "interaction.json"), patched, 0644)
			}
		}
	}

	// Streaming mode: không scan folder vào grid, không start watcher.
	// Accounts sẽ được đọc lần lượt khi user bấm Chạy.
	return ImportResult{}
}

// GetAccountSourceFolder — trả về thư mục nguồn hiện tại
func (a *App) GetAccountSourceFolder() string {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		return p.Account.FolderPath
	}
	return ""
}

// RefreshAccountSource — scan lại thư mục nguồn, chỉ thêm account mới (dedup UID)
func (a *App) RefreshAccountSource() ImportResult {
	folderPath := a.GetAccountSourceFolder()
	if folderPath == "" {
		return ImportResult{Errors: []string{"Chưa cấu hình thư mục nguồn"}}
	}
	return a.scanAccountFolder(folderPath)
}

// scanAccountFolder đọc tất cả .txt trong thư mục, chỉ add account chưa có (dedup UID)
func (a *App) scanAccountFolder(folderPath string) ImportResult {
	files, err := filepath.Glob(filepath.Join(folderPath, "*.txt"))
	if err != nil {
		return ImportResult{Errors: []string{"Lỗi đọc thư mục: " + err.Error()}}
	}
	if len(files) == 0 {
		return ImportResult{Errors: []string{"Không tìm thấy file .txt nào trong thư mục"}}
	}

	// Build UID set để dedup — lock khi đọc
	a.accountsMu.Lock()
	existingUIDs := make(map[string]bool, len(a.accounts))
	for _, acc := range a.accounts {
		if acc.UID != "" {
			existingUIDs[acc.UID] = true
		}
	}
	a.accountsMu.Unlock()

	imported := 0
	errors := make([]string, 0)
	var newAccounts []Account

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Lỗi đọc %s: %v", filepath.Base(filePath), err))
			continue
		}
		lines := splitLines(string(content))
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			acc := autoDetectAccount(line)
			if acc.UID == "" {
				errors = append(errors, fmt.Sprintf("%s dòng %d: không nhận diện UID", filepath.Base(filePath), i+1))
				continue
			}
			if existingUIDs[acc.UID] {
				continue // skip duplicate
			}
			existingUIDs[acc.UID] = true
			acc.FullData = line
			acc.Status = "new"
			acc.ImportTime = time.Now().Format("2006/01/02 15:04")
			acc.SourceCode = filepath.Base(filePath)
			newAccounts = append(newAccounts, acc)
			imported++
		}
	}

	// Append dưới lock — re-check UID để tránh TOCTOU race với folder watcher
	if len(newAccounts) > 0 {
		a.accountsMu.Lock()
		// Re-build live UID set để catch duplicate được thêm sau khi unlock lần đầu
		liveUIDs := make(map[string]bool, len(a.accounts))
		for _, acc := range a.accounts {
			if acc.UID != "" {
				liveUIDs[acc.UID] = true
			}
		}
		base := len(a.accounts)
		added := 0
		for i := range newAccounts {
			if liveUIDs[newAccounts[i].UID] {
				continue // race: UID đã được thêm bởi goroutine khác trong lúc đọc file
			}
			newAccounts[i].ID = base + added + 1
			a.accounts = append(a.accounts, newAccounts[i])
			added++
		}
		a.accountsMu.Unlock()
		imported = added
	}

	return ImportResult{Imported: imported, Errors: errors}
}

// === ACCOUNT CRUD ===

// ListAccounts trả về danh sách accounts đã lọc + tổng số kết quả.
// filter.Keyword: lọc theo UID/email/tên/note (case-insensitive)
// filter.Status: lọc theo trạng thái ("live", "die", "checkpoint", "new", "")
// filter.CategoryID: lọc theo category (nil = tất cả)
// filter.SortBy / filter.SortDir: chưa implement (dành cho tương lai)
// Merge realtime activity từ activityCache (sync.Map) mà không cần lock a.accounts.
func (a *App) ListAccounts(filter AccountFilter) AccountListResult {
	filtered := make([]Account, 0)
	kw := strings.ToLower(filter.Keyword)

	a.accountsMu.RLock()
	snapshot := make([]Account, len(a.accounts))
	copy(snapshot, a.accounts)
	a.accountsMu.RUnlock()

	for _, acc := range snapshot {
		// Merge realtime activity từ lock-free cache (cập nhật bởi onStatus hot path)
		if v, ok := a.activityCache.Load(acc.ID); ok {
			if s, ok := v.(string); ok {
				acc.Activity = s
			}
		}
		if kw != "" {
			match := strings.Contains(strings.ToLower(acc.UID), kw) ||
				strings.Contains(strings.ToLower(acc.Email), kw) ||
				strings.Contains(strings.ToLower(acc.FullName), kw) ||
				strings.Contains(strings.ToLower(acc.Note), kw)
			if !match {
				continue
			}
		}
		if filter.Status != "" && acc.Status != filter.Status {
			continue
		}
		if filter.CategoryID != nil && (acc.CategoryID == nil || *acc.CategoryID != *filter.CategoryID) {
			continue
		}
		filtered = append(filtered, acc)
	}

	return AccountListResult{Items: filtered, Total: len(filtered)}
}

// GetAccount trả về một account theo ID.
// id: ID nội bộ của account (auto-increment khi import).
// Thread-safe: dùng RLock vì không thay đổi slice.
// Trả về nil + error nếu ID không tồn tại.
func (a *App) GetAccount(id int) (*Account, error) {
	a.accountsMu.RLock()
	defer a.accountsMu.RUnlock()
	for _, acc := range a.accounts {
		if acc.ID == id {
			cp := acc
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("account ID %d không tồn tại", id)
}

// ImportAccounts — auto-detect fields từ dữ liệu giống WeBM frmAddAccount.btnSaveV2_Click
// Tự động nhận diện: UID, Password, Cookie (c_user=), Token (EAA), 2FA (32 chars hex),
// Email (@), PassMail, MailRecovery, Phone (digits 8-15), Client_ID (GUID), Refresh_token (M.xxx)
// LoadAccountsFromFile đọc 1 file .txt → parse accounts → thay thế store hiện tại.
// Khác với ImportAccounts (append): hàm này CLEAR store trước khi nạp.
// Dùng cho AccountSource="file" — user chọn file, load toàn bộ accounts vào grid, tick chọn.
//
// Emit event "accounts:folder-updated" sau khi load → AccountsPage auto refresh grid
// hiển thị accounts mới mà không cần user reload thủ công.
func (a *App) LoadAccountsFromFile(filePath string) ImportResult {
	// Chặn load khi verify/register đang chạy — clobber a.accounts giữa chừng sẽ làm
	// slot rows biến mất khỏi grid và worker đọc được dữ liệu sai.
	a.verifyMu.Lock()
	verifyRunning := a.isRunning
	a.verifyMu.Unlock()
	// Cấm load file khi register ở bất kỳ state nào ngoài idle (running OR stopping):
	// stopping vẫn còn workers đang quẫy → đụng vào a.accounts gây inconsistency.
	regBusy := runState(a.registerState.Load()) != runStateIdle
	if verifyRunning || regBusy {
		return ImportResult{Errors: []string{"Đang chạy verify/register — vui lòng dừng trước khi load file mới"}}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return ImportResult{Errors: []string{fmt.Sprintf("Không đọc được file: %v", err)}}
	}
	// Clear store trước khi load — file mode là replace, không append.
	a.accountsMu.Lock()
	a.accounts = nil
	a.accountsMu.Unlock()
	// Lưu path để OnAccountDone xóa dòng khỏi file gốc khi verify = live.
	a.sourceFilePathMu.Lock()
	a.sourceFilePath = filePath
	a.sourceFilePathMu.Unlock()
	// Persist accountSourcePath cho verify flow sau nhận path từ settings.
	existing := a.LoadSettings()
	existing.General.AccountSource = "file"
	existing.General.AccountSourcePath = filePath
	_ = a.SaveSettings(existing)
	result := a.ImportAccounts(string(data))
	// Emit event để AccountsPage refresh grid realtime.
	runtime.EventsEmit(a.ctx, "accounts:folder-updated", map[string]interface{}{
		"imported": result.Imported,
		"source":   "file",
		"path":     filePath,
	})
	return result
}

func (a *App) ImportAccounts(data string) ImportResult {
	lines := splitLines(data)
	imported := 0
	errors := make([]string, 0)

	// Parse trước, không cần lock
	var newAccounts []Account
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		acc := autoDetectAccount(line)
		if acc.UID == "" {
			errors = append(errors, fmt.Sprintf("Dòng %d: không nhận diện được UID", i+1))
			continue
		}
		acc.FullData = line
		acc.Status = "new"
		acc.ImportTime = time.Now().Format("2006/01/02 15:04")
		newAccounts = append(newAccounts, acc)
	}

	// Lock chỉ khi ghi vào a.accounts
	a.accountsMu.Lock()
	for i := range newAccounts {
		newAccounts[i].ID = len(a.accounts) + 1
		newAccounts[i].SourceCode = fmt.Sprintf("Import #%d", newAccounts[i].ID)
		a.accounts = append(a.accounts, newAccounts[i])
		imported++
	}
	a.accountsMu.Unlock()

	return ImportResult{Imported: imported, Errors: errors}
}

// autoDetectAccount — tự động nhận diện fields từ chuỗi phân cách |
// Mapping từ WeBM frmAddAccount.cs btnSaveV2_Click lines 256-383
func autoDetectAccount(line string) Account {
	fields := strings.Split(line, "|")
	acc := Account{}

	hasUID := false
	hasPassword := false
	hasEmail := false
	hasRecoveryEmail := false

	for i, raw := range fields {
		f := strings.TrimSpace(raw)
		if f == "" {
			continue
		}

		// 0a. iOS partial reg tokens — SRN:<srnonce> và SCUID:<sessionlessCryptedUID>.
		// Được append bởi formatRegResultLine cho iOS accounts để file-based verify hoạt động.
		if strings.HasPrefix(f, "SRN:") {
			acc.Srnonce = f[4:]
			continue
		}
		if strings.HasPrefix(f, "SCUID:") {
			acc.SessionlessCryptedUID = f[6:]
			continue
		}

		// 0b. EmailMeta — TempMail provider creds, format "MM:<base64-json>".
		// Append làm cột cuối khi save (xem internal/result/format.go FormatReg).
		// Loader cũ skip vì không match pattern UID/email/cookie nào.
		if strings.HasPrefix(f, "MM:") {
			if meta := resultpkg.ParseEmailMetaFromLine(f); meta != "" {
				acc.EmailMeta = meta
			}
			continue
		}

		// 1. Client_ID — GUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
		if isGUID(f) {
			acc.Note = f // Tạm lưu vào note, hoặc field riêng nếu cần
			continue
		}

		// 2. Refresh_token — bắt đầu bằng "M." và dài > 50
		if strings.HasPrefix(f, "M.") && len(f) > 50 {
			continue // Refresh token — không dùng cho verify
		}

		// 3. UID — field đầu tiên, ngắn < 50 ký tự
		if i == 0 && len(f) < 50 && !hasUID {
			acc.UID = f
			hasUID = true
			continue
		}

		// 4. 2FA — 32 ký tự hex (chứa cả chữ và số, không có dấu đặc biệt)
		cleaned2fa := strings.ReplaceAll(f, " ", "")
		if len(cleaned2fa) == 32 && isAlphaNumeric(cleaned2fa) && hasLetterAndDigit(cleaned2fa) {
			acc.Twofa = cleaned2fa
			continue
		}

		// 5. Token — bắt đầu bằng "EAA"
		if strings.HasPrefix(f, "EAA") {
			acc.Token = f
			continue
		}

		// 6. Cookie — chứa "c_user=" hoặc "ds_user_id="
		if strings.Contains(f, "c_user=") || strings.Contains(f, "ds_user_id=") {
			acc.Cookie = f
			// Extract UID từ cookie nếu chưa có
			if acc.UID == "" {
				acc.UID = extractCUserFromCookie(f)
				hasUID = true
			}
			continue
		}

		// 7. Password — ngay sau UID, chưa nhận diện password
		if hasUID && !hasPassword && !strings.Contains(f, "@") && !strings.Contains(f, "c_user=") {
			acc.Password = f
			hasPassword = true
			continue
		}

		// 8. Email — chứa @ và có dạng email
		if strings.Contains(f, "@") && strings.Contains(f, ".") {
			if !hasEmail {
				acc.Email = f
				hasEmail = true
				continue
			} else if !hasRecoveryEmail && f != acc.Email {
				acc.MailRecovery = f
				hasRecoveryEmail = true
				continue
			}
		}

		// 9. Pass Email — ngay sau email, chưa nhận diện
		if hasEmail && acc.PassMail == "" && !strings.Contains(f, "@") && !strings.Contains(f, "c_user=") && !strings.HasPrefix(f, "EAA") {
			acc.PassMail = f
			continue
		}

		// 10. Phone — toàn số, 8-15 ký tự
		if isAllDigits(f) && len(f) >= 8 && len(f) <= 15 {
			acc.Phone = f
			continue
		}
	}

	return acc
}

// isGUID kiểm tra string có đúng định dạng GUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx không.
// s: chuỗi cần kiểm tra (ví dụ "550e8400-e29b-41d4-a716-446655440000").
// Dùng để phân biệt Client_ID (GUID) với các field khác khi auto-detect account.
func isGUID(s string) bool {
	// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isAlphaNumeric kiểm tra string chỉ chứa chữ cái (a-z, A-Z) và chữ số (0-9).
// s: chuỗi cần kiểm tra.
// Dùng bước đầu để lọc ứng viên 2FA key (32 ký tự alphanumeric).
func isAlphaNumeric(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// hasLetterAndDigit kiểm tra string có chứa ít nhất 1 chữ cái VÀ 1 chữ số không.
// s: chuỗi đã qua isAlphaNumeric (chỉ còn [a-zA-Z0-9]).
// Lý do: 2FA key phải là mix của cả chữ và số (loại bỏ các field toàn số như phone, toàn chữ như password đơn giản).
func hasLetterAndDigit(s string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// isAllDigits kiểm tra string chỉ chứa chữ số và không rỗng.
// s: chuỗi cần kiểm tra.
// Dùng để nhận diện số điện thoại: toàn số, độ dài 8-15 ký tự.
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// DeleteAccounts xóa các accounts khỏi memory theo danh sách ID.
// ids: slice các ID cần xóa (từ frontend khi user chọn hàng rồi bấm Delete).
// Thread-safe: dùng Lock vì thay đổi slice a.accounts.
// Trả về số lượng thực tế đã xóa (có thể < len(ids) nếu một số ID không tồn tại).
func (a *App) DeleteAccounts(ids []int) DeleteResult {
	idSet := make(map[int]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	a.accountsMu.Lock()
	remaining := make([]Account, 0, len(a.accounts))
	deleted := 0
	for _, acc := range a.accounts {
		if idSet[acc.ID] {
			deleted++
		} else {
			remaining = append(remaining, acc)
		}
	}
	a.accounts = remaining
	a.accountsMu.Unlock()
	return DeleteResult{Deleted: deleted}
}

// === VERIFY ===

// GetPermanentFileCounts trả về số dòng trong phone.txt và mail.txt.
// Frontend dùng để hiển thị số lượng data đã tích lũy.
func (a *App) GetPermanentFileCounts() map[string]int {
	permDir := defaultPermanentDir()
	result := map[string]int{"phone": 0, "mail": 0}
	for _, key := range []string{"phone", "mail"} {
		data, err := os.ReadFile(filepath.Join(permDir, key+".txt"))
		if err != nil {
			continue
		}
		count := 0
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		result[key] = count
	}
	return result
}

// setupLogging — cấu hình slog ghi vào thư mục logs/ cạnh file exe.
func setupLogging() {
	logDir := filepath.Join(AppDataDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "run-"+time.Now().Format("20060102")+".log")

	// Log rotation cho 24/7: nếu file > 10 MB, rotate sang .N (keep 3 backups).
	// Cap tổng log history ~40 MB thay vì grow unbounded.
	rotateIfLarge(logFile, 10<<20, 3)

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return // fallback: slog default (stderr)
	}
	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
	slog.Info("app started", "version", "0.1.0")
}

// rotateIfLarge check file size → nếu vượt maxBytes thì rotate (file → file.1, file.1 → file.2...).
// keep: số file backup tối đa. File cũ nhất bị xoá.
func rotateIfLarge(path string, maxBytes int64, keep int) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxBytes {
		return
	}
	lastBackup := fmt.Sprintf("%s.%d", path, keep)
	_ = os.Remove(lastBackup)
	for i := keep - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(path, path+".1")
}

// extractCUserFromCookie lấy UID Facebook từ chuỗi cookie bằng cách tìm c_user=VALUE.
// cookie: chuỗi cookie dạng "name1=value1; c_user=123456789; name3=value3".
// Trả về UID (ví dụ "123456789") hoặc "" nếu không có c_user.
// Dùng trong autoDetectAccount khi account chỉ có cookie (không có UID riêng).
func extractCUserFromCookie(cookie string) string {
	for _, pair := range strings.Split(cookie, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "c_user=") {
			return strings.TrimPrefix(pair, "c_user=")
		}
	}
	return ""
}
