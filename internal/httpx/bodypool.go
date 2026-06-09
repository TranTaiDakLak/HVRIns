// Package httpx — shared utilities cho HTTP response body reading với buffer pool.
// Giảm GC pressure khi app chạy throughput cao: mỗi HTTP call bình thường allocate
// []byte mới cho response body → với 100+ req/s thì ~2-3 MB/s allocation.
// Buffer pool reuse []byte buffer → giảm ~60-70% allocation rate cho body reads.
package httpx

import (
	"bytes"
	"io"
	"sync"
)

// bodyBufPool tái sử dụng *bytes.Buffer cho việc đọc HTTP response body.
// Size khởi tạo 8 KB — đủ cho hầu hết JSON response, grow tự động nếu cần.
var bodyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 8*1024)
		return bytes.NewBuffer(b)
	},
}

// maxReusableBodyBufferCap — cap capacity tối đa khi return buffer về pool.
// Nếu buffer đã grow vượt cap (do response lớn), discard thay vì giữ lại,
// tránh trường hợp buffer 5MB+ kẹt trong pool → giữ RAM mãi không trả lại OS.
// 256 KB đủ cho 99% response FB/banclone bình thường.
const maxReusableBodyBufferCap = 256 * 1024

// ReadBody đọc response body vào buffer từ pool, trả về []byte copy.
// Caller không cần close buffer (tự return vào pool sau khi copy xong).
// limit: max bytes đọc để chặn response quá lớn — 0 = không limit.
//
// Pattern dùng:
//
//	resp, err := client.Do(req)
//	if err != nil { ... }
//	defer resp.Body.Close()
//	data, err := httpx.ReadBody(resp.Body, 512*1024)  // limit 512 KB
func ReadBody(r io.Reader, limit int64) ([]byte, error) {
	buf := bodyBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	// Defer return: chỉ put lại pool nếu cap < threshold, tránh giữ buffer lớn
	defer func() {
		if buf.Cap() <= maxReusableBodyBufferCap {
			bodyBufPool.Put(buf)
		}
		// else: drop buffer, để GC dọn — pool sẽ tạo buffer mới lần sau
	}()

	var src io.Reader = r
	if limit > 0 {
		src = io.LimitReader(r, limit)
	}
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}

	// Copy ra []byte riêng vì buffer sẽ return vào pool
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

// ReadBodyString tiện lợi khi caller chỉ cần string.
func ReadBodyString(r io.Reader, limit int64) (string, error) {
	data, err := ReadBody(r, limit)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
