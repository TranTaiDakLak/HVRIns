// upload_avatar.go — S23 rupload avatar upload flow
// Flow: GET rupload (check offset) → POST binary → POST graphql Bloks NUX set avatar
// Tham chiếu: E:\WEMAKE\DocWeMake\API\UpAVT\ ([317], [326], [333])
package android

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	ruploadBase    = "https://rupload.facebook.com/service_menu_uploads"
	graphqlURL     = "https://graph.facebook.com/graphql"
	bloksVerID     = "385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665"
	nuxAvatarAppID = "com.bloks.www.bloks.nux.profilepicture.async.upload"
	nuxClientDocID = "11994080422955588194694478490"
	nuxFriendly    = "FbBloksActionRootQuery-com.bloks.www.bloks.nux.profilepicture.async.upload"
)

// UploadAvatarS23 chọn ảnh ngẫu nhiên từ avatarDir và upload làm avatar
// thông qua S23 rupload flow. Cần token OAuth hợp lệ.
func UploadAvatarS23(ctx context.Context, proxyStr, token, ua, avatarDir string) error {
	imgPath, imgBytes, err := pickRandomImage(avatarDir)
	if err != nil {
		return fmt.Errorf("pick avatar: %w", err)
	}
	_ = imgPath

	hash := md5.Sum(imgBytes)
	md5hex := fmt.Sprintf("%x", hash)
	tsMs := time.Now().UnixMilli()
	uploadID := fmt.Sprintf("%s-0-%d-%d-%d", md5hex, len(imgBytes), tsMs, tsMs+int64(mrand.Intn(400)+100))

	client := buildAvatarClient(proxyStr)
	// Task 5: per-call client + custom Transport → close idle khi func return.
	// Avatar upload chạy in background goroutine sau verify live; mỗi acc 1 client.
	defer client.CloseIdleConnections()

	ruploadURL := fmt.Sprintf("%s/%s", ruploadBase, uploadID)

	if err := ruploadGet(ctx, client, ruploadURL, token, ua); err != nil {
		return fmt.Errorf("rupload GET: %w", err)
	}

	handle, err := ruploadPost(ctx, client, ruploadURL, uploadID, imgBytes, token, ua)
	if err != nil {
		return fmt.Errorf("rupload POST: %w", err)
	}

	return setAvatarNUX(ctx, client, token, ua, handle)
}

func buildAvatarClient(proxyStr string) *http.Client {
	transport := &http.Transport{
		DisableCompression: true,
	}
	if pURL := proxy.FormatProxyURL(proxyStr); pURL != "" {
		if u, err := url.Parse(pURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Transport: transport, Timeout: 60 * time.Second}
}

func ruploadGet(ctx context.Context, client *http.Client, ruploadURL, token, ua string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", ruploadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "OAuth "+token)
	req.Header.Set("user-agent", ua)
	req.Header.Set("x-fb-friendly-name", "Resumable-Upload-Get")
	req.Header.Set("x-meta-zca", "empty_token")
	req.Header.Set("accept-encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func ruploadPost(ctx context.Context, client *http.Client, ruploadURL, uploadID string, imgBytes []byte, token, ua string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", ruploadURL, bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}
	size := fmt.Sprintf("%d", len(imgBytes))
	req.Header.Set("x-entity-length", size)
	req.Header.Set("x-entity-name", uploadID)
	req.Header.Set("x-entity-type", "image/jpeg")
	req.Header.Set("offset", "0")
	req.Header.Set("x-fb-friendly-name", "Resumable-Upload-Post")
	req.Header.Set("authorization", "OAuth "+token)
	req.Header.Set("user-agent", ua)
	req.Header.Set("content-type", "application/octet-stream")
	req.Header.Set("x-meta-zca", "empty_token")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.ContentLength = int64(len(imgBytes))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		H string `json:"h"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse handle: %w body=%s", err, string(body))
	}
	if result.H == "" {
		return "", fmt.Errorf("empty handle body=%s", string(body))
	}
	return result.H, nil
}

func setAvatarNUX(ctx context.Context, client *http.Client, token, ua, handle string) error {
	formBody, err := buildSetAvatarBody(handle)
	if err != nil {
		return err
	}

	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	if _, err := gz.Write([]byte(formBody)); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, &gzBuf)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "OAuth "+token)
	req.Header.Set("user-agent", ua)
	req.Header.Set("x-fb-friendly-name", nuxFriendly)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("content-encoding", "gzip")
	req.Header.Set("x-meta-zca", "empty_token")
	req.Header.Set("x-graphql-client-library", "graphservice")
	req.Header.Set("x-graphql-request-purpose", "fetch")
	req.Header.Set("accept-encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("graphql status %d", resp.StatusCode)
	}
	return nil
}

func buildSetAvatarBody(handle string) (string, error) {
	// Inner-most params (level 4)
	inner := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"profile_picture_handle": handle,
		},
		"server_params": map[string]interface{}{
			"is_caa":                            0,
			"INTERNAL__latency_qpl_marker_id":   36707139,
			"INTERNAL__latency_qpl_instance_id": mrand.Int63n(1e12),
			"profile_picture_source":            "nux",
			"is_renux":                          0,
		},
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", err
	}

	// Level 3: {"params": "<innerJSON>"}
	level3JSON, err := json.Marshal(map[string]interface{}{"params": string(innerJSON)})
	if err != nil {
		return "", err
	}

	// nt_context block
	ntCtx := map[string]interface{}{
		"using_white_navbar":           true,
		"styles_id":                    "6100e7e89411ccf67ace027cedecd84f",
		"pixel_ratio":                  3,
		"is_push_on":                   false,
		"debug_tooling_metadata_token": nil,
		"is_flipper_enabled":           false,
		"theme_params": []interface{}{
			map[string]interface{}{"value": []string{"three_neutral_gray"}, "design_system_name": "XMDS"},
			map[string]interface{}{"value": []string{}, "design_system_name": "FDS"},
		},
		"bloks_version": bloksVerID,
	}

	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params":              string(level3JSON),
			"bloks_versioning_id": bloksVerID,
			"app_id":              nuxAvatarAppID,
		},
		"scale":                                    "3",
		"use_native_entrypoint_for_stars_on_reels": true,
		"nt_context":                               ntCtx,
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return "", err
	}

	analyticsJSON, _ := json.Marshal([]string{"GraphServices"})

	vals := url.Values{}
	vals.Set("method", "post")
	vals.Set("pretty", "false")
	vals.Set("format", "json")
	vals.Set("server_timestamps", "true")
	vals.Set("locale", "en_US")
	vals.Set("purpose", "fetch")
	vals.Set("fb_api_req_friendly_name", nuxFriendly)
	vals.Set("fb_api_caller_class", "graphservice")
	vals.Set("client_doc_id", nuxClientDocID)
	vals.Set("fb_api_client_context", `{"is_background":false}`)
	vals.Set("variables", string(variablesJSON))
	vals.Set("fb_api_analytics_tags", string(analyticsJSON))
	vals.Set("client_trace_id", uuid.New().String())
	return vals.Encode(), nil
}

func pickRandomImage(dir string) (string, []byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("read dir %q: %w", dir, err)
	}
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".png") {
			images = append(images, filepath.Join(dir, e.Name()))
		}
	}
	if len(images) == 0 {
		return "", nil, fmt.Errorf("no images in %q", dir)
	}
	chosen := images[mrand.Intn(len(images))]
	data, err := os.ReadFile(chosen)
	if err != nil {
		return "", nil, err
	}
	return chosen, data, nil
}
