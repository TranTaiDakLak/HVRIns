# Phan tich RAM tang cao khi chay split REG 500 + VERIFY 500

Ngay lap: 2026-04-27

Pham vi: chi phan tich, chua sua code.

Tinh huong tu anh: chay khoang 10 phut, che do split, REG 500 luong + VERIFY 500 luong, RAM hien thi khoang 5.5 GB, CPU khoang 44%.

## Ket luan ngan

RAM 5.5 GB sau 10 phut la dau hieu can xu ly. Khong nen coi day chi la "Go chua tra RAM ve OS".

Nguyen nhan co kha nang cao nhat la tong hop cua 4 nhom:

1. Tong concurrency thuc te len toi 1000 job HTTP song song.
2. Ghi result dang co nhieu goroutine nen va `UpsertByUID` doc/ghi lai ca file khi Die/Unknown tang nhanh.
3. Transport/session HTTP giu socket/TLS/native buffer ngoai Go heap.
4. Event IPC va cac viec nen nhu sheets/avatar/datr/permanent file tao allocation va goroutine khong gioi han.

GC tuning hien co giup giam heap, nhung khong tri duoc native buffer, goroutine dang block I/O, WebView2/IPC, va thuat toan doc-ghi file lap lai.

## 1. So RAM tren status bar dang do cai gi

Vi tri code:

- `app.go:3183` - `GetResourceUsage()`
- `cpu_windows.go:77` - `getProcessMemoryMB()`
- `cpu_windows.go:88` - uu tien `PrivateWorkingSetSize`

Y nghia:

- So RAM tren UI la private working set cua process chinh.
- No gom Go heap, goroutine stack, native buffer cua HTTP/TLS, file/socket buffer, mot phan state runtime.
- No khong dong nghia voi `heapInuseMB`.

Anh huong:

- Neu status bar hien 5.5 GB nhung `DebugMemory().heapInuseMB` chi vai tram MB, nguyen nhan chinh nam ngoai Go heap: transport, TLS, socket, native/webview/fragmentation.
- Neu `heapInuseMB` cung tang len GB, nguyen nhan chinh la object Go bi giu song: queue, slice, closure, result lines, event payload.

Can do de tach nguon:

- Goi `DebugMemory()` trong luc dang chay.
- So sanh `rssMB`, `heapInuseMB`, `heapAllocMB`, `goroutines`, `transportPool`, `iosSessions`, `androidSessions`.

## 2. Split mode tao 1000 luong thuc te

Vi tri code:

- `app.go:6266` - register semaphore `maxThreads`
- `app.go:6335` - `regToVerCh` tao khi `SplitMode && VerifyEnabled`
- `app.go:7140` - `SplitVerifyThreads`
- `app.go:7142` - neu verifyThreads <= 0 thi bang `maxThreads`
- `app.go:7249` - channel free slot verify theo `verifyThreads`
- `app.go:7674` - split verify nhan line tu `regToVerCh`

Y nghia:

- Khi user dat REG = 500 va SplitVerifyThreads = 500, app khong phai chay 500 tong, ma chay gan 1000 workload song song.
- Moi workload co the gom HTTP request, proxy, cookie jar, response body, email polling, log/event, result writer.

Anh huong RAM:

- Chi can moi workload trung binh giu 4-6 MB la da cham 4-6 GB.
- Mot so platform/token client co TLS/client state rieng, nen muc trung binh nay la thuc te.

Muc do: P0.

Khuyen nghi van hanh truoc khi sua:

- Test lai voi tong concurrent 400-600, vi du REG 250 + VERIFY 250.
- Neu RAM giam gan tuyen tinh, concurrency la thanh phan lon nhat.

## 3. Ghi result async khong gioi han

Vi tri code:

- `app.go:2313` - `go saveVerifyOutcome(...)` trong verify thuong
- `app.go:6888` - `go saveRegOutcome(...)` trong register
- `app.go:7353` - `go saveVerifyOutcome(...)` trong split verify
- `app.go:6903`, `app.go:6905` - `go appendUniqueLineToPermanentFile(...)`
- `internal/runner/scheduler.go:579` - `go cookie.AppendDatr(...)`
- `internal/facebook/register/webandroid/register.go:345` - append datr async
- `internal/facebook/register/ioshttp/register.go:553` - append datr async
- `internal/facebook/register/web/steps.go:481` - append datr async

Y nghia:

- Moi account done co the spawn them goroutine ghi file.
- Neu disk cham hoac file lock dang bi giu, cac goroutine xep hang.
- Closure co the giu `Account`, cookie, token, user-agent, message trong RAM cho den khi ghi xong.

Anh huong RAM:

- Voi 1000 luong, neu mot dot account fail nhanh, co the co hang nghin goroutine nen.
- Khi result file lon, moi goroutine save bi block lau hon, lam backlog tang tiep.

Muc do: P0.

Huong xu ly:

- Chuyen sang writer queue co gioi han.
- Khi queue day thi block/backpressure thay vi spawn vo han.
- Tach queue: result, cookie/datr, webhook, avatar.

## 4. `UpsertByUID` doc/ghi lai ca file, rat nguy hiem khi Die/Unknown tang nhanh

Vi tri code:

- `internal/result/writer.go:25` - `Die.txt` dung upsert
- `internal/result/dispatch.go:49` - Die dispatch voi `Upsert: true`
- `internal/result/store.go:105` - `UpsertByUID`
- `internal/result/store.go:150` - goi `writeAllLinesAtomic`
- `internal/result/store.go:187` - ghi lai toan bo file

Y nghia:

- Moi lan ghi Die theo UID, code mo file, doc tat ca dong vao `[]string`, scan, sua/append, roi ghi lai toan bo file.
- Khi verify die 5,000+ dong trong 10 phut, chi phi se tang theo kich thuoc file.
- Nhieu goroutine save cung cho cung 1 file lock, backlog tang nhanh.

Anh huong RAM/CPU/Disk:

- RAM tang do nhieu lan tao slice `lines`.
- CPU tang do scan chuoi lien tuc.
- Disk I/O tang do rewrite file lap lai.
- Goroutine save bi keo dai, lam closure song lau hon.

Muc do: P0, ung vien root cause manh nhat neu `Die.txt`/`Unknown.txt` lon nhanh.

Huong xu ly:

- Khong upsert truc tiep moi account bang cach rewrite file.
- Dung append-only trong luc chay, dedupe cuoi batch.
- Hoac giu in-memory UID index + flush dinh ky.
- Hoac chuyen ket qua sang SQLite/Badger roi export txt.

## 5. HTTP transport pool giu native buffer

Vi tri code:

- `internal/proxy/transport_pool.go:25` - cap transport pool 200
- `internal/proxy/transport_pool.go:119` - `MaxIdleConnsPerHost: 16`
- `internal/proxy/transport_pool.go:120` - `MaxConnsPerHost: 100`
- `internal/proxy/transport_pool.go:121` - `MaxIdleConns: 128`
- `internal/proxy/transport_pool.go:122` - `IdleConnTimeout: 20s`
- `internal/proxy/transport_pool.go:178` - `TransportPoolStats`

Y nghia:

- Transport pool giup giam tao transport moi, nhung moi transport co the giu idle socket/TLS buffer.
- Voi proxy xoay nhieu session key, pool co the dat cap 200.
- Moi transport co idle conn rieng, nen RAM ngoai heap co the tang manh.

Anh huong RAM:

- Neu `rssMB` tang nhung `heapInuseMB` khong tang tuong ung, day la nguon nghi van lon.
- GC khong thu hoi ngay native/socket buffer khi transport van song trong pool.

Muc do: P1.

Huong xu ly:

- Giam idle caps trong workload 1000 luong.
- Co metric chi tiet hon: active conn/idle conn/transport hit/miss.
- Co cleanup theo run hoac theo proxy session TTL ngan hon khi split high-concurrency.

## 6. SessionPool WebAndroid/iOS giu client + cookie jar theo proxy

Vi tri code:

- `internal/facebook/register/webandroid/http.go:347` - `SharedSessionPool`
- `internal/facebook/register/webandroid/http.go:351` - `SessionPool`
- `internal/facebook/register/webandroid/http.go:364` - `Acquire`
- `internal/facebook/register/webandroid/http.go:379` - `Store`
- `internal/facebook/register/webandroid/http.go:417` - `CloseIdleConnsAll`
- `internal/facebook/register/webandroid/http.go:438` - `CloseAll`
- `internal/facebook/register/ioshttp/http.go:274` - `SharedSessionPool`
- `internal/facebook/register/ioshttp/http.go:278` - `SessionPool`
- `internal/facebook/register/ioshttp/http.go:349` - `CloseIdleConnsAll`
- `internal/facebook/register/ioshttp/http.go:370` - `CloseAll`

Y nghia:

- Khi keep-session bat, app giu session/client/cookie jar de tang ti le trust.
- Day la tradeoff dung ve ti le thanh cong, nhung ton RAM khi concurrent lon.

Anh huong RAM:

- Moi proxy/session co client state rieng.
- `CloseIdleConnsAll` chi dong idle connections, khong drop session.
- `CloseAll` chi chay khi run dung/xong.

Muc do: P1 neu platform la WebAndroid/iOS hoac co keep session nhieu.

Huong xu ly:

- Them cap session theo slot/thoi gian.
- Neu RSS tang nhanh, can do `iosSessions`, `androidSessions` trong `DebugMemory()`.

## 7. Email `CredPool` co refill nen khong gan lifecycle run

Vi tri code:

- `internal/email/rent/pool.go:23` - `CredPool`
- `internal/email/rent/pool.go:95` - `Get(ctx)`
- `internal/email/rent/pool.go:112` - `go p.doRefill(context.Background())`
- `internal/email/rent/pool.go:160` - `doRefill(ctx)`

Y nghia:

- Khi pool gan het mail, code spawn refill nen bang `context.Background()`.
- Refill nen khong bi cancel theo run/Stop.

Anh huong RAM:

- Mot pool chi co mot refill cung luc, nen khong phai nguon 5.5 GB chinh.
- Nhung sau nhieu lan Stop/Start, request mua mail cu co the song qua run cu va giu pool/closure.

Muc do: P2.

Huong xu ly:

- `CredPool` nhan parent context cua run.
- Them `Close()` de cancel refill va clear queue khi run ket thuc.

## 8. Event IPC Wails/WebView tao allocation ap luc

Vi tri code:

- `app.go:2300` - `verify:account-done`
- `app.go:2487` - `verify:batch-status`
- `app.go:6307` - `register:batch-status`
- `app.go:7097` - `register:account-done`
- `app.go:7346` - split `verify:account-done`
- `app.go:7293` - split `verify:batch-status`
- `internal/runner/scheduler.go:356` - raw proxy callback
- `internal/runner/scheduler.go:363` - background `CheckIP`
- `internal/runner/scheduler.go:591` - `OnAccountDone`

Y nghia:

- Code da batch mot so status, day la diem tot.
- Nhung cac payload hot path van la `map[string]interface{}` va JSON qua Wails.
- Proxy/email/account-done van co the la event rieng.

Anh huong RAM:

- Thuong la allocation pressure va CPU serialize JSON, khong phai leak chinh.
- Khi 1000 luong chay, WebView IPC queue co the cham lai, lam Go giu payload lau hon.

Muc do: P2.

Huong xu ly:

- Dung struct typed nho thay vi map cho hot events.
- Batch proxy/email/account-done neu UI chap nhan tre 200-500ms.
- Do event/giay de biet co flood khong.

## 9. Upload avatar, Google Sheets, upload site

Vi tri code:

- `app.go:2344` - verify live thi co the upload avatar
- `app.go:2371` - `UploadAvatarS23`
- `app.go:7380` - split verify live upload avatar
- `app.go:7401` - split `UploadAvatarS23`
- `app.go:2316` den `app.go:2339` - Google Sheets push verify live
- `app.go:7356` den `app.go:7376` - Google Sheets push split verify live

Y nghia:

- Cac viec nay chi chay khi live, nen voi anh verify live 239 thi khong phai nguon lon nhat.
- Nhung neu live cao, moi live co the spawn request nen rieng.

Anh huong RAM:

- Giu token/cookie/UA/proxy trong closure.
- Neu site/sheets/avatar server cham, goroutine song toi 20-60 giay.

Muc do: P2/P3 tuy config.

Huong xu ly:

- Dua vao bounded queue rieng.
- Gioi han avatar concurrent 1-2.
- Sheets concurrent 1-2, timeout ngan, drop/retry co cap.

## 10. GC tuning hien co khong du tri root cause

Vi tri code:

- `app.go:350` - `debug.SetMemoryLimit(1024 << 20)`
- `app.go:427` - start memory maintenance
- `app.go:484` - `runMemoryMaintenance`
- `app.go:587` - `DebugMemory`

Y nghia:

- Code da co GC aggressive va FreeOSMemory dinh ky.
- Nhung memory limit cua Go khong ep duoc tat ca native/socket/TLS buffers.

Anh huong:

- Neu tang RAM do result backlog/transport/native, them GC se chi lam CPU tang, khong xu ly goc.
- Nen dung GC de quan sat, khong dung nhu fix chinh.

Muc do: diagnostic.

## 11. Khu vuc khong phai nghi van chinh

### Frontend grid

Frontend da co cap va cleanup trong nhieu cho. `registerThreads` co cap cleanup khi >1200 slot. Day khong phai root cause chinh cua 5.5 GB trong 10 phut, vi so RAM tren UI la process backend chinh.

### Static fakeinfo/user-agent pools

Fakeinfo/UA/data file load luc startup. Chung ton RAM nen co dinh, khong giai thich duoc RAM tang theo tung phut.

### Upload site retry queue

Code upload site da co cap retry queue va pending. Neu user khong bat upload site, bo qua. Neu bat, can do rieng, nhung khong phai dau tien.

## Bang uu tien xu ly

| Uu tien | Khu vuc | Ly do |
|---|---|---|
| P0 | `UpsertByUID`/result writer | Co the tao RAM + CPU + disk backlog nhanh nhat khi Die/Unknown nhieu |
| P0 | Fire-and-forget goroutine | Khong co backpressure, backlog co the no theo toc do account done |
| P0 | Tong concurrency split | 500+500 la 1000 workload song song, vuot ngan sach RAM cua process don |
| P1 | Transport/session pool | Native buffer cao, GC khong tri duoc |
| P2 | CredPool context | Song qua Stop/Start, leak nho nhung can fix |
| P2 | Wails event IPC | Allocation/JSON pressure, anh huong CPU va WebView queue |
| P2/P3 | Sheets/avatar/upload | Phu thuoc config va ti le live |

## Cach test xac nhan nhanh

### Test A: xac dinh heap hay native

Trong luc RAM status bar cao, lay `DebugMemory()`:

- Neu `rssMB` cao hon `heapInuseMB` rat nhieu: nghi transport/session/native/WebView.
- Neu `heapInuseMB` cung cao: nghi result queue/closure/slice/event payload.
- Neu `goroutines` > 2000: co backlog goroutine nen.

### Test B: tat bot post-processing

Chay lai 10 phut voi:

- Tat UploadAvatar.
- Tat Google Sheets webhook.
- Tat upload site neu co.

Neu RAM giam manh, nhom post-processing co anh huong lon. Neu khong giam nhieu, quay lai P0 result writer/concurrency/transport.

### Test C: giam split concurrency

Chay:

- REG 250 + VERIFY 250.
- REG 300 + VERIFY 200.

Neu RAM giam gan ty le voi tong luong, can dat gioi han tong concurrent thay vi cho 500+500 trong mot process.

### Test D: theo doi file result

Trong luc chay, xem kich thuoc:

- `Die.txt`
- `Unknown.txt`
- `SuccessReg.txt`
- `SuccessVerify*.txt`

Neu `Die.txt`/`Unknown.txt` tang nhanh, `UpsertByUID` la diem can sua dau tien.

## Huong sua de tri de

### Phase 1: sua trong process hien tai

1. Thay `go saveRegOutcome`/`go saveVerifyOutcome` bang bounded result writer.
2. Bo upsert rewrite-file moi account. Dung append-only + dedupe cuoi batch, hoac SQLite.
3. Gom `AppendDatr`, permanent phone/mail, sheets, avatar vao bounded worker pool.
4. Them counter debug: result queue len, dropped/backpressure, event/sec.
5. Cho `CredPool` co `Close()` va parent context.

### Phase 2: cap native memory va session

1. Giam idle conn cap khi chay split high-concurrency.
2. Them TTL/cap session pool theo slot.
3. Them debug chi tiet: transport pool size, session count, goroutines, heap/RSS delta.

### Phase 3: worker process architecture

Repo da co thiet ke dung huong:

- `docs/WORKER_ARCHITECTURE.md`
- `internal/worker/contracts.go`

Huong nay tach workload nang ra worker process rieng, co `MaxRSSMB`, `MaxJobs`, `MaxLifetimeSec`.

Loi ich:

- Wails app process chi giu UI/controller.
- Worker vuot RAM thi recycle, khong keo sap UI.
- 500 REG + 500 VERIFY co the chia thanh nhieu process, moi process cap RAM rieng.

Day la cach ben vung nhat neu muc tieu la chay 500+500 lien tuc nhieu gio/ngay.

## Khuyen nghi tam thoi truoc khi co fix

Neu can chay ngay:

1. Khong chay 500+500 trong mot process neu may RAM duoi 16-32 GB.
2. Uu tien 250+250 hoac tong concurrent khoang 400-600.
3. Tat UploadAvatar/Sheets/upload site khi test RAM.
4. Bat snapshot/debug va chup `DebugMemory()` sau 2, 5, 10 phut.
5. Theo doi `Die.txt`/`Unknown.txt`; neu file lon nhanh, uu tien sua result writer truoc.

