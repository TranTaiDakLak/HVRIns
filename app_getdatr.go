package main

// app_getdatr.go — lấy datr mới từ account đã Live qua GraphQL profile-switcher.
//
// Flow (port C#, chạy inline sau verify Live khi GetNewDatrOnLive bật):
//   1. Dùng token + cookie + UA của account vừa verify Live.
//   2. POST graph.facebook.com/graphql với profile-switcher body.
//   3. Parse response → extract datr mới.
//   4. Ghi vào Pool file + in-memory pool.

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/proxy"
)

const (
	graphQLEndpoint  = "https://graph.facebook.com/graphql"
	getNewDatrTimeoutSec = 20
	bloksVersion     = "f7f7731412f4c87953c95acd93686df979f7c47efd9b38d1778f2ffa2ae19220"
	stylesID         = "964d559c1e2aa0142b5069bc8cb1adea"
)

const defaultAndroidUAForDatr = "Mozilla/5.0 (Linux; Android 11; Redmi Note 8 Build/RKQ1.201004.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/107.0.5304.141 Mobile Safari/537.36"

// fetchNewDatrFromAccountUA — POST GraphQL profile-switcher với UA của account.
// Dùng sau verify Live để lấy datr mới → thêm vào pool.
func fetchNewDatrFromAccountUA(ctx context.Context, uid, token, cookieStr, userAgent, proxyStr string) string {
	if userAgent == "" {
		userAgent = defaultAndroidUAForDatr
	}
	body := buildProfileSwitcherBody(uid)
	client := proxy.CreateClient(proxyStr, getNewDatrTimeoutSec*time.Second)

	req, err := http.NewRequestWithContext(ctx, "POST", graphQLEndpoint, strings.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("X-Fb-Connection-Type", "WIFI")
	req.Header.Set("X-Fb-Http-Engine", "Tigon/Liger")
	req.Header.Set("X-Fb-Client-Ip", "True")
	req.Header.Set("X-Fb-Server-Cluster", "True")
	req.Header.Set("X-Tigon-Is-Retry", "False")
	req.Header.Set("x-fb-device-group", "5427")
	req.Header.Set("Authorization", "OAuth "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("x-fb-qpl-active-flows-json", `{"schema_version":"v2","inprogress_qpls":[],"snapshot_attributes":{}}`)
	req.Header.Set("x-graphql-client-library", "graphservice")
	req.Header.Set("x-zero-state", "unknown")
	req.Header.Set("priority", "u=3, i")
	req.Header.Set("x-fb-background-state", "1")
	req.Header.Set("Accept", "application/json, text/json, text/x-json, text/javascript, application/xml, text/xml")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Host", "graph.facebook.com")
	req.Header.Set("Accept-Encoding", "gzip")
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("[GetDatrOnLive] HTTP error", "uid", uid, "err", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("[GetDatrOnLive] HTTP status không phải 200", "uid", uid, "status", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, gerr := gzip.NewReader(resp.Body)
		if gerr != nil {
			slog.Warn("[GetDatrOnLive] gzip decode lỗi", "uid", uid, "err", gerr)
			return ""
		}
		defer gr.Close()
		reader = gr
	}
	raw, err := io.ReadAll(io.LimitReader(reader, 512*1024))
	if err != nil {
		slog.Warn("[GetDatrOnLive] đọc body lỗi", "uid", uid, "err", err)
		return ""
	}

	data := unescapeUnicode(raw)
	datr := getValueAfterKey(data, `datr","value":"`)
	if datr == "" {
		datr = getValueAfterKey(data, `"datr","value":"`)
	}
	if datr == "" {
		slog.Warn("[GetDatrOnLive] parse datr thất bại", "uid", uid, "body_prefix", safePrefix(string(raw), 200))
	}
	return strings.TrimSpace(datr)
}

// buildProfileSwitcherBody — body POST GraphQL profile-switcher (port C#).
func buildProfileSwitcherBody(uid string) string {
	sessionID := uuid.NewString()
	traceID := uuid.NewString()
	ntCtx := fmt.Sprintf(
		`{"using_white_navbar":true,"styles_id":"%s","pixel_ratio":3,"is_push_on":true,"debug_tooling_metadata_token":null,"is_flipper_enabled":false,"theme_params":[],"bloks_version":"%s"}`,
		stylesID, bloksVersion,
	)
	payloadURL := fmt.Sprintf(
		`%%2Fprofile%%2Fswitcher%%2Fdirectswitch%%3Fprofile_id%%3D%s%%26caller_subsurface%%3Dbookmarks_nt_switcher%%26logging_event_session_id%%3D%s`,
		uid, sessionID,
	)
	variables := fmt.Sprintf(
		`{"scale":"3","params":{"payload":"%s","nt_context":%s},"nt_context":%s,"profile_image_size":282,"include_image_ranges":true}`,
		payloadURL, ntCtx, ntCtx,
	)
	return fmt.Sprintf(
		`method=post&pretty=false&format=json&server_timestamps=true&locale=en_US`+
			`&fb_api_req_friendly_name=NativeTemplateAsyncQuery`+
			`&fb_api_caller_class=graphservice`+
			`&client_doc_id=307495399210002463256326833731`+
			`&fb_api_client_context={"is_background":false}`+
			`&variables=%s`+
			`&fb_api_analytics_tags=["GraphServices"]`+
			`&client_trace_id=%s`,
		variables, traceID,
	)
}

// getValueAfterKey — C# StringHelper.GetValue: tìm key, trả chuỗi đến '"' tiếp theo.
func getValueAfterKey(s []byte, key string) string {
	idx := bytes.Index(s, []byte(key))
	if idx < 0 {
		return ""
	}
	s = s[idx+len(key):]
	end := bytes.IndexByte(s, '"')
	if end < 0 {
		return string(s)
	}
	return string(s[:end])
}

// unescapeUnicode — giải mã \uXXXX trong JSON response (giống C# StringHelper.Unescape).
func unescapeUnicode(b []byte) []byte {
	if !bytes.Contains(b, []byte(`\u`)) {
		return b
	}
	s := strings.ReplaceAll(string(b), `<`, "<")
	s = strings.ReplaceAll(s, `>`, ">")
	s = strings.ReplaceAll(s, `&`, "&")
	s = strings.ReplaceAll(s, `"`, `"`)
	s = strings.ReplaceAll(s, `/`, "/")
	return []byte(s)
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
