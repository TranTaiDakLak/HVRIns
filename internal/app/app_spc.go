// app_spc.go — Secondary Profile Creation (SPC) Phase 1.
//
// Sau khi 1 account REG live + check-live xác nhận "live", dùng chính account đó
// làm PARENT để tạo N account CON qua luồng SPC (internal/instagram/register/igspc).
//
// Phase 1: tạo con (account_created=true) → lưu SPC_Children.txt + live-check bearer.
// Con SPC ở trạng thái partially_created → live-check thường "unknown" tới khi verify
// email (Phase 2). Con vẫn có session (cookie/bearer) + có thể tự làm parent.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram/register/igspc"
	resultpkg "HVRIns/internal/result"

	runtime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// spcCooldown — nghỉ giữa 2 lần tạo từ CÙNG parent (doc 2.2: ≥30s tránh rate-limit).
	spcCooldown = 35 * time.Second
	// spcDefaultChildren — số con/parent mặc định khi config = 0.
	spcDefaultChildren = 2
	// spcMaxChildren — chặn trên an toàn (tránh spam parent → checkpoint, mất tài sản gốc).
	spcMaxChildren = 10
)

// runSPCForParent tạo tối đa N con từ 1 parent live. Chạy TRONG goroutine check-live
// đã detached khỏi reg slot → sleep cooldown không block reg. KHÔNG trả lỗi: mọi
// nhánh fail đều log/emit + continue.
func (a *App) runSPCForParent(ctx context.Context, acc Account, proxyStr string, cfg InteractionConfig, writer *resultpkg.Writer) {
	n := cfg.SPCChildrenPerParent
	if n <= 0 {
		n = spcDefaultChildren
	}
	if n > spcMaxChildren {
		n = spcMaxChildren
	}

	uid := acc.UID
	if uid == "" {
		uid = cookieValueSPC(acc.Cookie, "ds_user_id")
	}
	bearer := resultpkg.BuildIGBearerToken(acc.Cookie)
	if bearer == "" || uid == "" {
		a.emitSPC(acc.Username, 0, n, "bỏ qua: parent thiếu cookie/uid")
		return
	}
	parent := igspc.Parent{Username: acc.Username, UID: uid, Bearer: bearer}
	mailCfg := buildVerifyConfigFromInteraction(cfg)
	noop := func(string) {}

	slog.Info("[SPC] START", "parent", acc.Username, "parentUID", uid, "children", n, "provider", mailCfg.MailProvider)
	a.emitSPC(acc.Username, 0, n, fmt.Sprintf("bắt đầu SPC: parent %s → %d con", acc.Username, n))

	created := 0
	for i := 0; i < n; i++ {
		if ctx.Err() != nil {
			return
		}
		// Cooldown giữa các lần (trừ lần đầu).
		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(spcCooldown):
			}
		}

		// 1. Email tạm cho con (dùng provider đang cấu hình).
		h, mErr := acquireTempMailForReg(ctx, mailCfg, proxyStr, noop)
		if mErr != nil {
			a.emitSPC(acc.Username, created, n, "mail lỗi: "+mErr.Error())
			continue
		}
		childEmail := h.Email

		// 2. Tạo con — retry 1 lần khi lỗi MẠNG (proxy EOF/timeout), không retry lỗi logic.
		var child igspc.Child
		var cErr error
		for attempt := 0; attempt < 2; attempt++ {
			child, cErr = igspc.CreateChild(ctx, parent, igspc.Options{Proxy: proxyStr, Email: childEmail})
			if cErr == nil {
				break
			}
		}
		h.Close()

		if cErr != nil {
			a.emitSPC(acc.Username, created, n, "con lỗi mạng: "+cErr.Error())
			continue
		}
		if !child.Success {
			a.emitSPC(acc.Username, created, n, "con fail: "+child.Message)
			continue
		}
		created++

		// 3. Live-check con bằng bearer (Phase 1: partial → thường unknown; vẫn ghi nhận).
		lc, lcancel := context.WithTimeout(ctx, 20*time.Second)
		liveRes := igcore.CheckLiveByBearer(lc, child.Bearer, proxyStr)
		lcancel()

		// 4. Lưu con: lead = uid (con chưa có @handle thật), password rỗng (passwordless).
		line := resultpkg.FormatReg(resultpkg.RegData{
			UID:    child.UID,
			Cookie: child.Cookie,
			Token:  child.Bearer,
		}, nil)
		if writer != nil && line != "" {
			_ = writer.Append(resultpkg.FileSPCChildren, line)
			if liveRes == "live" {
				_ = writer.Append(resultpkg.FileLive, line)
			}
		}

		runtime.EventsEmit(a.ctx, "register:spc-child", map[string]interface{}{
			"parentUsername": acc.Username,
			"childUID":       child.UID,
			"live":           liveRes,
			"created":        created,
			"total":          n,
		})
		a.emitSPC(acc.Username, created, n, fmt.Sprintf("✅ con %s (live=%s)", child.UID, liveRes))
	}

	a.emitSPC(acc.Username, created, n, fmt.Sprintf("xong SPC: %d/%d con từ %s", created, n, acc.Username))
}

// emitSPC bắn event tiến trình SPC lên UI + ghi run-*.log (để chẩn đoán: SPC vô hình
// trên UI nếu FE chưa nghe event, nên log file là nguồn theo dõi chính).
func (a *App) emitSPC(parentUsername string, created, total int, msg string) {
	slog.Info("[SPC] "+msg, "parent", parentUsername, "created", created, "total", total)
	runtime.EventsEmit(a.ctx, "register:spc-result", map[string]interface{}{
		"parent":  parentUsername,
		"created": created,
		"total":   total,
		"msg":     msg,
	})
}

// cookieValueSPC lấy value của 1 key trong cookie string ("k1=v1; k2=v2").
func cookieValueSPC(cookie, key string) string {
	for _, seg := range strings.Split(cookie, ";") {
		seg = strings.TrimSpace(seg)
		if strings.HasPrefix(seg, key+"=") {
			return strings.TrimPrefix(seg, key+"=")
		}
	}
	return ""
}
