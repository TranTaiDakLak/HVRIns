# Soak Test — RAM / CPU / Goroutine khi chạy lâu

> Quy trình verify rằng các fix lifecycle/context/HTTP/event listener không bị regress.
> Đo metrics định kỳ trong nhiều giờ, so sánh trước/sau Stop, xác nhận không leak tuyến tính.

---

## 1. Mục tiêu test

Sau loạt fix Task 1-6, app cần thỏa mãn các tiêu chí:

1. **RAM không tăng tuyến tính** — sau giờ đầu plateau, không drift quá 100MB/giờ.
2. **CPU ổn định** — sustained CPU dưới 70% với 200 luồng (không spike vô tận).
3. **Goroutine count bounded** — không grow quá `2 × số luồng` sustained.
4. **Stop responsive** — sau bấm Stop, trong 1-3 phút goroutine giảm về gần baseline.
5. **Không overlap run** — Start/Stop 10 lần liên tiếp không tích lũy worker.
6. **HTTP transport idle clean** — sau Stop, idle TCP/TLS được close periodic.

---

## 2. Cấu hình máy test

| Item | Khuyến nghị |
|---|---|
| OS | Windows 10/11 (môi trường production) |
| RAM | ≥ 8 GB free |
| CPU | ≥ 4 cores |
| Disk | SSD nếu có (tránh disk slow giả lập leak goroutine) |
| Network | Stable internet, không VPN throttle |
| Antivirus | Loại trừ folder app khỏi real-time scan (ảnh hưởng I/O) |

---

## 3. Cấu hình test

### Số luồng theo phase

| Phase | Threads | Mục tiêu |
|---|---|---|
| Smoke | 50 | Verify lifecycle cơ bản, RAM ≤ 500MB |
| Standard | 100 | Workload thực tế, RAM ≤ 800MB |
| Stress | 200 | Tải cao, RAM ≤ 1.5GB |
| Heavy | 500+ | Tải cực hạn (chỉ test ngắn 30 phút) |

### Thời lượng

| Phase | Thời gian | Mục đích |
|---|---|---|
| Quick | 30 phút | Smoke test sau mỗi fix |
| Medium | 2 giờ | Verify plateau memory |
| Long | 6 giờ | Catch slow leak |
| Overnight | 12 giờ | Production simulation |

### Cấu hình máy test cần ghi lại trước khi chạy

Trước mỗi soak run, fill bảng dưới để tracking về sau:

```
Date            : ____-__-__
OS              : Windows __ (build ____)
CPU             : __ cores @ __GHz (e.g. 8C16T Ryzen 5800X)
RAM             : __ GB (free at start: __ GB)
Disk            : SSD/HDD
Network         : direct/VPN, bandwidth __ Mbps
Proxy provider  : minproxy / shoplike / tinsoft / netproxy / proxy_fixed
Threads config  : __
App version     : git commit ______ (HaVu.exe build date __)
DEBUG_PPROF     : on/off
DEBUG_SNAPSHOT  : on/off
```

---

## 3.1. Kịch bản test bắt buộc (Test A–F)

Toàn bộ scenarios sau cần PASS trước khi release production. Mỗi scenario có thời lượng + kỳ vọng cụ thể.

### Test A — Idle 30 phút

| Item | Value |
|---|---|
| Setup | Start app, KHÔNG bấm Run gì cả |
| Duration | 30 phút |
| Monitor | `soak-monitor.ps1 -DurationMinutes 30` |
| PASS criteria | RSS plateau ≤ 250 MB; goroutine ≤ 50 sustained; CPU avg < 1% |
| FAIL nếu | RSS tăng > 50 MB trong 30p; goroutine grow đều; CPU > 5% sustained |

### Test B — Register 50 luồng trong 2 giờ

| Item | Value |
|---|---|
| Setup | Cấu hình proxy + cookie initial. Run register `maxThreads=50` |
| Duration | 2 giờ |
| Monitor | `soak-monitor.ps1 -DurationMinutes 120` |
| PASS criteria | RSS plateau ≤ 800 MB sau giờ đầu; goroutine ≤ 200 sustained; CPU avg ≤ 50%; numGC tăng đều |
| FAIL nếu | RSS drift > 200 MB trong giờ 2; goroutine grow > 300; app panic |

### Test C — Register 100 luồng trong 6 giờ

| Item | Value |
|---|---|
| Setup | Run register `maxThreads=100`. Có thể bật split mode (verify chung) |
| Duration | 6 giờ |
| Monitor | `soak-monitor.ps1 -DurationMinutes 360` |
| PASS criteria | RSS plateau ≤ 1.5 GB; goroutine ≤ 500; **delta RSS start↔end < 300 MB**; CPU avg ≤ 60% |
| FAIL nếu | RSS tăng tuyến tính > 200 MB/giờ; goroutine grow > 1000; HeapAlloc drift |

### Test D — Stop/Start 10 lần

| Item | Value |
|---|---|
| Setup | Run register 100 thread; chu kỳ Start 5p → Stop → đợi 1p (verify state=idle) → Start lại |
| Duration | ~10 × 6p = 60 phút |
| Monitor | `soak-monitor.ps1 -DurationMinutes 60` + theo dõi log "REG pool create/cleanup" với `runID` |
| PASS criteria | Sau 10 chu kỳ: goroutine baseline ≤ 50; iosSessions/androidSessions = 0 sau Stop; KHÔNG log "REG pool cleanup" của runID cũ kèm theo `closedSessions > 0` đáng kể (nghĩa là pool đã CloseAll OK trước khi Start tiếp); transport pool size không phình; RSS không grow > 200 MB tổng |
| FAIL nếu | Goroutine tích lũy mỗi chu kỳ; pool count sau Stop > 0; log run cũ vẫn xuất hiện sau Start mới |

### Test E — Upload đang chạy rồi Stop

| Item | Value |
|---|---|
| Setup | Cấu hình banclone code+key, push 200-500 acc lên uploadCh (qua reg/verify), bấm Stop khi đang push |
| Duration | Đo từ lúc bấm Stop đến lúc state idle |
| Verify | Xem log `[upload] 🛑 Đã dừng goroutine upload`; log `STOP register completed` |
| PASS criteria | Stop **return trong < 60 giây** (uploadHardStopTimeout sau Task 4); pending acc còn trong `pending.txt` (load lại lần sau OK); goroutine upload exit |
| FAIL nếu | Stop chờ > 60s; goroutine upload còn chạy sau 2 phút; "✅ Tải lên thành công" emit sau khi Stop |

### Test F — Verify đang chạy rồi Stop

| Item | Value |
|---|---|
| Setup | Run verify 100 thread (file mode hoặc CloneHV). Bấm Stop sau ~1 phút |
| Duration | Đo Stop → state idle |
| Verify | Frontend hiển thị "Đang dừng..." → "Hoàn thành" trong < 60s; log `verify:complete` emit 1 lần |
| PASS criteria | Sau Stop **≤ 60 giây**, state=idle; goroutine giảm về < 50; sau Task 2 (ctx propagation) verify worker exit nhanh khỏi `time.Sleep(checkDelay)` |
| FAIL nếu | "Đang dừng" kéo dài > 2 phút; verify worker còn emit batch-status sau 30s; account-done emit sau khi state=idle |

---

## 4. Chỉ số cần ghi (mỗi 60 giây)

`soak-monitor.ps1` ghi tự động (cột CSV): `Timestamp, RssMB, CpuPct, NumGoroutine, ThreadCount, HandleCount`.
Các metric Go-side (HeapAlloc/Sys/iosSessions/...) phải ghi thủ công bằng cách gọi
`DebugMemory()` qua frontend (Settings → Debug Panel) hoặc qua `DEBUG_SNAPSHOT=1` (log
mỗi 1 phút vào structured log file `%APPDATA%/HVR/logs/run-YYYYMMDD.log`).

| Metric | Nguồn | Kỳ vọng |
|---|---|---|
| **rssMB** | `Get-Process HaVu`.WorkingSet64 / 1MB (script tự ghi) | Plateau sau 1h, không tăng linear |
| **heapAllocMB** | `DebugMemory()`.heapAllocMB | Dao động theo GC cycle, không drift |
| **heapInuseMB** | `DebugMemory()`.heapInuseMB | Plateau |
| **heapIdleMB** | `DebugMemory()`.heapIdleMB (Task 7) | Tăng = chưa trả OS, gọi FreeOSMemory |
| **heapReleasedMB** | `DebugMemory()`.heapReleasedMB (Task 7) | Tăng = đã trả OS, RSS sẽ giảm theo |
| **sysMB** | `DebugMemory()`.sysMB | Plateau |
| **numGoroutine** | pprof `/debug/pprof/goroutine?debug=1` line đầu (script tự ghi) | ≤ 2× threads |
| **cpuPct** | `Get-Process` TotalProcessorTime delta / cores (script tự ghi) | Avg ≤ 70% với 200 thread |
| **numGC / lastGCAgoSec** | `DebugMemory()` (Task 7) | Tăng đều; lastGCAgoSec ≤ 60s khi đang chạy |
| **gcCpuFraction** | `DebugMemory()` (Task 7) | < 0.05 (5%); cao = GC quá tích cực |
| **iosSessions / androidSessions** | `DebugMemory()` | ≤ pool size config khi chạy; = 0 sau Stop+Cleanup |
| **transportPool** | `DebugMemory()`.transportPool | ≤ 200 (cap, xem proxy/transport_pool.go) |
| **uploadPending** | `DebugMemory()`.uploadPending | Drain sau mỗi batch |
| **active workers** ¹ | Đếm gián tiếp qua `numGoroutine - baseline` | ≈ maxThreads × 1-3 (mỗi worker 1-3 goroutine phụ) |
| **account processed** | UI status bar (Tổng/Live/Die counter) | Tăng đều khi đang chạy |
| **timeout / cancel errors** ² | grep log file `error\|timeout\|cancel\|panic` | Đếm số dòng/giờ — không grow exponential |
| **registerRunning/Stopping** | `DebugMemory()` | Reflect actual state machine |
| **verifyRunning/Stopping** | `DebugMemory()` | Reflect actual state |
| **uploadStopping** | `DebugMemory()` | True khi đang chờ drain (Task 4) |

¹ Backend chưa expose riêng "active workers" counter (xem Task 7 R1). Tạm tính qua delta goroutine. Có thể thêm `activeWorkers` field vào DebugMemory ở task tương lai nếu cần.

² Soak xong, parse log:
```powershell
Select-String -Path "$env:APPDATA\HVR\logs\run-*.log" -Pattern "panic|error|timeout|cancel" | Measure-Object
```

---

## 5. Cách bật DEBUG_PPROF + lấy profile

### Bật pprof endpoint

PowerShell:
```powershell
$env:DEBUG_PPROF = "1"
$env:DEBUG_SNAPSHOT = "1"  # thêm log snapshot mỗi 1 phút vào file log
.\build\bin\HaVu.exe
```

Bash (Git Bash):
```bash
DEBUG_PPROF=1 DEBUG_SNAPSHOT=1 ./build/bin/HaVu.exe
```

→ pprof listen tại `http://127.0.0.1:6060/debug/pprof/` (chỉ localhost).

### Lấy heap profile

```bash
# Tải heap snapshot
go tool pprof -png -output heap.png http://127.0.0.1:6060/debug/pprof/heap

# Hoặc interactive:
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
# (pprof) top
# (pprof) list <FunctionName>
```

### Lấy goroutine dump

```bash
# Text dump (xem từng goroutine + stack):
curl http://127.0.0.1:6060/debug/pprof/goroutine?debug=2 -o goroutines.txt

# Pprof binary (analyze count):
go tool pprof http://127.0.0.1:6060/debug/pprof/goroutine
```

### Lấy CPU profile (30s sample)

```bash
go tool pprof -png -output cpu.png "http://127.0.0.1:6060/debug/pprof/profile?seconds=30"
```

---

## 6. Workflow test

### Smoke test 30 phút (sau mỗi fix)

1. Start app với `DEBUG_PPROF=1 DEBUG_SNAPSHOT=1`
2. Chạy script monitoring:
   ```powershell
   .\scripts\soak-monitor.ps1 -DurationMinutes 30 -OutputCsv .\soak-30m.csv
   ```
3. Trong app: Start register/verify với 50 thread
4. Chờ 30 phút → kết thúc → Stop
5. Đợi 3 phút → check goroutine count giảm về < 50
6. Phân tích CSV: vẽ chart `rssMB`, `goroutines` theo thời gian

### Long soak 6 giờ

```powershell
.\scripts\soak-monitor.ps1 -DurationMinutes 360 -OutputCsv .\soak-6h.csv
```

→ Sau 6h, check trong CSV:
- `rssMB` start vs end → diff < 200MB
- `goroutines` không tăng đều
- `numGC` tăng đều (không stall)

### Stress Stop/Start 10 lần

```
Loop 10 lần:
  1. Start với 100 thread
  2. Chờ 5 phút
  3. Stop
  4. Đợi 1 phút
  5. Verify goroutine baseline (≤ 50)
```

→ Check sau loop: tổng goroutine không grow theo số lần Stop/Start.

---

## 7. Tiêu chí pass / fail

### ✅ PASS criteria

| Test | Pass nếu |
|---|---|
| **Test A** Idle 30m | RSS plateau ≤ 250 MB; goroutine ≤ 50; **CPU idle < 1%** |
| **Test B** 50 thread / 2h | RSS plateau ≤ 800 MB; goroutine ≤ 200; CPU avg ≤ 50%; HeapAlloc dao động không drift |
| **Test C** 100 thread / 6h | RSS plateau ≤ 1.5 GB; goroutine ≤ 500; **không drift > 200 MB/giờ**; delta start↔end < 300 MB |
| **Test D** Stop/Start × 10 | **Không tích lũy goroutine** sau loop (delta < 50); iosSessions/androidSessions = 0 sau Stop; transport pool size không phình; **không có log từ run cũ sau khi run mới start** |
| **Test E** Upload Stop | **Stop return ≤ 60 giây** (uploadHardStopTimeout sau Task 4 — KHÔNG còn 60 phút); pending acc giữ trong pending.txt |
| **Test F** Verify Stop | **Stop return ≤ 60 giây** (sau Task 1 state machine + Task 2 ctx propagation); state=idle; goroutine giảm về baseline; verify worker thoát khỏi `time.Sleep(checkDelay)` ngay (không chờ hết) |
| Stop responsive (chung) | Sau Stop bất kỳ run nào, **goroutine giảm về baseline trong ≤ 3 phút** |
| Upload site die simulation | Sau 5 retry, batch drop + log "⛔ Bỏ qua N acc" — không retry vô hạn (Task 4) |
| **Không có panic/race obvious** | Toàn bộ soak run KHÔNG có dòng log `panic:` hoặc Go race-detector report |
| **CPU idle thấp** | Khi không Run gì, CPU process ≤ 1% (chỉ tick GC + watchdog) |

### ❌ FAIL criteria

- RSS tăng tuyến tính > 200 MB/giờ liên tục
- Goroutine count tăng đều theo thời gian (không bounded)
- Sau Stop 3 phút, goroutine vẫn > 100 (worker zombie)
- Stop/Start 10 lần làm RSS double
- App crash / panic trong soak (kiểm tra log)
- HTTP push goroutine sống > 10 phút sau Stop
- **Upload Stop chờ > 60 giây** (regression Task 4)
- **Verify Stop chờ > 60 giây** với checkDelay nhỏ (regression Task 2 R01)
- **Log run cũ xuất hiện ở run mới** (regression Task 1 / Task 3 lifecycle ownership)
- **CPU idle vẫn cao** (>5%) kéo dài nhiều phút khi không có gì chạy

---

## 8. Script monitoring

`scripts/soak-monitor.ps1` ghi metrics mỗi 60s vào CSV.

Cách chạy:
```powershell
.\scripts\soak-monitor.ps1 `
  -ProcessName HaVu `
  -PProfPort 6060 `
  -OutputCsv .\soak.csv `
  -IntervalSec 60 `
  -DurationMinutes 360
```

Tham số:
- `-ProcessName` — tên process (default `HaVu`)
- `-PProfPort` — port pprof (default 6060, chỉ dùng nếu `DEBUG_PPROF=1`)
- `-OutputCsv` — file output
- `-IntervalSec` — chu kỳ ghi (default 60)
- `-DurationMinutes` — chạy trong N phút (default 60)

CSV columns: `Timestamp, RssMB, CpuPct, NumGoroutine, NumThreads, HandleCount`

---

## 9. Phân tích kết quả

### Excel/Google Sheets

1. Open CSV
2. Insert chart line cho `RssMB` và `NumGoroutine` theo `Timestamp`
3. Mong muốn: cả 2 line plateau (flat) hoặc dao động trong band cố định

### Pattern xấu cần phát hiện

| Pattern | Diagnose |
|---|---|
| RSS line dốc lên đều | Memory leak dài hạn — check `DebugMemory().heapAllocMB` xem có grow không, nếu Go heap stable nhưng RSS grow → native (WebView2/TLS buffer) leak |
| RSS spike cao rồi không giảm | GC không kịp trả OS — tăng `GOGC` aggressive, hoặc force `FreeOSMemory` |
| Goroutine tăng đều | Goroutine leak — pprof `/debug/pprof/goroutine?debug=2` dump để identify |
| CPU 100% sustained | Tight loop — pprof `/debug/pprof/profile?seconds=30` |
| Sau Stop goroutine không giảm | Lifecycle Stop không hoạt động — check Task 1+2 fixes (state machine, ctx cancel) |

---

## 10. Báo cáo kết quả

Sau khi chạy soak, gửi:

1. **CSV file** `soak-Xh.csv` (raw data)
2. **Chart** RssMB + Goroutine theo thời gian
3. **Heap profile** PNG khi end
4. **Goroutine dump** text dump khi end
5. **Số liệu summary**:
   - Start RSS / End RSS / Peak RSS
   - Avg / Peak CPU%
   - Start goroutine / End goroutine / Peak goroutine
   - Số lần GC trong soak
   - Số account processed
   - Số timeout/cancel error trong log
6. **Test verdict** PASS/FAIL theo tiêu chí mục 7
