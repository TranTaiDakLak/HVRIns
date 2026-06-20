# REG S561V2 vs S561V3 comparison

Muc dich: dung tai lieu nay de so sanh offline giua `s561v2` hien co va mot spec/capture `s561v3` neu co sau nay. Tai lieu nay khong wire them platform REG moi va khong them luong tao tai khoan runnable.

## Trang thai hien tai

| Hang muc | Gia tri |
|---|---|
| Platform hien co | `s561`, `s561v2`, `s562`, `s562v3` |
| Platform chua co | `s561v3` |
| Thu muc `s561v2` | `internal/facebook/register/s561v2/` |
| Wiring hien co | `facebook.PlatformS561V2`, `app_reg_sxxx.go`, `uabuilder/android.go` |
| Canh bao git | `internal/facebook/register/s561v2/` dang untracked trong working tree |

## Bang so sanh

| Nhom | `s561v2` hien co | `s561v3` spec/capture | Ghi chu audit |
|---|---|---|---|
| Package | `s561v2` | Chua co | Khong tao package runnable khi chua co spec |
| FBAV | `561.0.0.42.67` | Chua co | Lay tu `S560AppVersions` |
| FBBV | `968460367` | Chua co | Lay tu `S560AppVersions` |
| Original UA | FB4A `561.0.0.42.67`, `968460367`, `SM-S911B` | Chua co | Neu co v3, UA phai khop FBAV/FBBV/body |
| `client_doc_id` | `119940804217607550791392237684` | Chua co | Khac `s561` |
| `bloks_versioning_id` | `9d448b7c3b47250635f0acbdd801409700933b0eba77ec236358984692f4d562` | Chua co | Giong `s561` |
| `styles_id` | `8a1cdd5e66badb0c4d074cbdf05d3791` | Chua co | Giong `s561` |
| `theme_params` | FDS-only, value empty | Chua co | Khac `s561` co XMDS + FDS |
| `is_push_on` | `true` | Chua co | Nam trong `nt_context` |
| Top-level form keys | `method`, `pretty`, `format`, `server_timestamps`, `locale`, `purpose`, `fb_api_req_friendly_name`, `fb_api_caller_class`, `client_doc_id`, `fb_api_client_context`, `variables`, `fb_api_analytics_tags`, `client_trace_id` | Chua co | Khong thay key rieng so voi `s561` |
| `client_input_params` | Co `ck_error`, `aac`, `device_id`, `waterfall_id`, `zero_balance_state`, `failed_birthday_year_count`, `headers_last_infra_flow_id`, `machine_id`, `lois_settings`, `encrypted_msisdn`, ... | Chua co | Can diff theo key va type |
| `server_params` | Co `event_request_id`, `device_id`, `waterfall_id`, `flow_info`, `reg_info`, `family_device_id`, `offline_experiment_group`, `x_app_device_signals`, ... | Chua co | Can diff theo key va type |
| HTTP transport/header | Giong `s561` ve code `http.go` | Chua co | Chi package name khac khi so voi `s561` |
| Profile/UA builder | Giong `s561` ve code `profile.go`, dung version/build v2 | Chua co | Version/build den tu `PlatformAppVersions` |
| Warm seed/session | Ham `warmSession` ton tai trong `extras.go`, nhung `register.go` cua `s561v2` khong goi | Chua co | Khac `s561` va `s562v3`, ca hai co goi warm |
| Register log tag | `[S561V2]` | Chua co | Dung de grep runtime log |

## Diem khac chinh giua `s561` va `s561v2`

| Hang muc | `s561` | `s561v2` |
|---|---|---|
| FBAV/FBBV | `561.0.0.3.67` / `964730465` | `561.0.0.42.67` / `968460367` |
| `client_doc_id` | `119940804210934765769791410287` | `119940804217607550791392237684` |
| `bloks_versioning_id` | Giong `s561v2` | Giong `s561` |
| `styles_id` | Giong `s561v2` | Giong `s561` |
| `theme_params` | XMDS `three_neutral_gray` + FDS empty | FDS empty only |
| Warm seed/session | Co copy seed cookie va goi `warmSession` | Khong goi trong `Register` |

## Can capture/spec gi de dien `s561v3`

Neu co spec/capture `s561v3`, chi dien bang offline cac truong sau:

| Truong can lay | Ly do |
|---|---|
| FBAV/FBBV/FBRV | Doi chieu UA va body version |
| `client_doc_id` | De biet co dung query revision moi khong |
| `bloks_versioning_id` | De biet co doi Bloks bundle khong |
| `styles_id` | De biet `nt_context` co doi UI bundle khong |
| `theme_params` | De so sanh FDS/XMDS |
| `is_push_on` | De so sanh `nt_context` |
| Top-level form keys | Phat hien key moi/bo key |
| `client_input_params` keys/types | Phat hien schema moi |
| `server_params` keys/types | Phat hien schema moi |
| Header names only | So sanh cau truc request, khong can ghi secret/cookie |
| Body size va response class | Phuc vu debug/log, khong dung de bypass |

## Ghi chu ve `s561v3`

Khong nen tao `s561v3` runnable bang cach clone `s561v2` va doan version/doc_id neu khong co spec. Lam vay chi tao mot platform thu nghiem mo ho, kho audit va de lam sai thong ke loi. Cach dung tai lieu nay la dien spec vao cot `s561v3`, sau do review khac biet offline truoc khi quyet dinh co can thay doi code noi bo nao khong.
