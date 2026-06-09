// extras.go — Web register init (fetch tokens) + password crypto.
//
// File này gộp 2 file cũ:
//   - init.go    → FetchRegHTML/FetchRegTokens/fetchLoginPageKey + extract helpers + randS
//   - crypto.go  → GenerateEncPassword (AES-GCM + NaCl SealedBox) cho B6
package web

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"

	"HVRIns/internal/proxy"
)

// ─── Init: Fetch initial tokens from Facebook registration page ──────────────
// GET https://m.facebook.com/r.php → extract fb_dtsg, lsd, datr, reg_context, public key.

const defaultRegUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/134.0.1911.158 Mobile/15E148 Safari/604.1"

// FetchRegHTML tải trang đăng ký Facebook (r.php) và trả về raw HTML.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout cho request.
//   - proxy: chuỗi proxy "host:port:user:pass" hoặc rỗng để dùng IP thật.
//   - ua: User-Agent string; rỗng → dùng defaultRegUA.
//
// Hàm này là wrapper tiện lợi của FetchRegHTMLFull, cố định URL tại
// "https://m.facebook.com/r.php". Dùng chủ yếu trong dev/debug để xem
// HTML trang đăng ký mà không cần tạo full RegSession.
func FetchRegHTML(ctx context.Context, proxyStr, ua string) (string, error) {
	return FetchRegHTMLFull(ctx, proxyStr, ua, "https://m.facebook.com/r.php")
}

// FetchRegHTMLFull tải bất kỳ URL nào bằng GET và trả về raw HTML body.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout cho request.
//   - proxy: chuỗi proxy "host:port:user:pass" hoặc rỗng để dùng IP thật.
//   - ua: User-Agent string; rỗng → dùng defaultRegUA.
//   - targetURL: URL đầy đủ cần tải (ví dụ "https://m.facebook.com/r.php").
//
// Response body được giới hạn 5 MB (5<<20 bytes) để tránh OOM khi trang
// trả về nội dung bất thường.
func FetchRegHTMLFull(ctx context.Context, proxyStr, ua, targetURL string) (string, error) {
	client := proxy.CreateClient(proxyStr, 20*time.Second)
	if ua == "" {
		ua = defaultRegUA
	}
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", targetURL, err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	return string(body), nil
}

// FetchRegTokens GET trang đăng ký Facebook và extract tất cả tokens cần thiết
// vào session. Đây là bước khởi tạo bắt buộc — phải gọi trước bất kỳ bước
// B1-B8 nào trong luồng đăng ký.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout cho request.
//   - s: con trỏ tới RegSession; các trường sau sẽ được ghi vào s:
//   - s.Datr: giá trị cookie "datr" từ Set-Cookie header của response.
//   - s.FbDtsg: token bảo mật fb_dtsg (bắt buộc cho mọi POST request).
//   - s.Lsd: token LSD; fallback về "P2XQp3VtEtx1ajSpA8Wbgw" nếu không tìm được.
//   - s.Jazoest: giá trị jazoest; fallback về "21966".
//   - s.Hsi: header session identifier; fallback về chuỗi mặc định.
//   - s.Rev: revision number; fallback về "1036244875".
//   - s.Dyn: dynamic parameter __dyn.
//   - s.S: session hash __s dạng "xxx:yyy:zzz"; sinh ngẫu nhiên nếu r.php không có.
//   - s.RegContext: token ngữ cảnh đăng ký (large encrypted JSON string).
//   - s.PubKeyHex: public key hex 64 ký tự để mã hóa password tại B6.
//   - s.PubKeyID: key ID tương ứng (mặc định "6").
//   - s.PubKeyVer: key version (mặc định "5").
//
// Extraction Set-Cookie: datr được đọc từ resp.Cookies() (parsed header),
// không dùng cookie jar để tránh Facebook trả về cookie cũ từ request trước.
//
// Nếu r.php không trả về publicKey, hàm tự động gọi fetchLoginPageKey để
// thử lấy public key từ m.facebook.com/login/ làm fallback.
func FetchRegTokens(ctx context.Context, s *RegSession) error {
	client := proxy.CreateClient(s.Proxy, 20*time.Second)

	ua := s.UserAgent
	if ua == "" {
		ua = defaultRegUA
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://m.facebook.com/r.php", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET r.php: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	html := string(body)

	// Extract datr cookie từ Set-Cookie header
	for _, c := range resp.Cookies() {
		if strings.EqualFold(c.Name, "datr") {
			s.Datr = c.Value
		}
	}

	// Extract session tokens
	s.FbDtsg = extractFirstReg(html,
		`name="fb_dtsg" value="([^"]+)"`,
		`DTSGInitialData",\[\],\{"token":"([^"]+)"`,
		`"dtsg":\{"token":"([^"]+)"`,
		`initDtsg":"([^"]+)"`,
	)
	s.Lsd = extractFirstReg(html, `LSD",\[\],\{"token":"([^"]+)"`)
	if s.Lsd == "" {
		s.Lsd = "P2XQp3VtEtx1ajSpA8Wbgw"
	}
	s.Jazoest = extractFirstReg(html,
		`name="jazoest" value="(\d+)"`,
		`jazoest=(\d+)`,
	)
	if s.Jazoest == "" {
		s.Jazoest = "21966"
	}
	s.Hsi = extractFirstReg(html, `"hsi":"([^"]+)"`)
	if s.Hsi == "" {
		s.Hsi = "7052297439888351100-0"
	}
	s.Rev = extractFirstReg(html, `"__spin_r":(\d+)`, `"__rev":(\d+)`)
	if s.Rev == "" {
		s.Rev = "1036244875"
	}
	s.Dyn = extractFirstReg(html, `"__dyn":"([^"]+)"`)

	// __s — session identifier "xxx:yyy:zzz", generate nếu không extract được
	s.S = extractFirstReg(html, `"__s":"([a-z0-9:]+)"`)
	if s.S == "" {
		s.S = randS()
	}

	// Extract reg_context — large encrypted token
	s.RegContext = extractJSONStr(html, "reg_context")

	// Extract public key — cho bước B6 (encrypt password)
	// C# WeBM dùng "publicKey" (camelCase), pattern rộng như C#: "publicKey":"(.*?)"
	s.PubKeyHex = extractFirstReg(html,
		`"publicKey":"([^"]+)"`,
		`"publicKey"\s*:\s*"([^"]+)"`,
		`"public_key":"([^"]+)"`,
		`"public_key"\s*:\s*"([^"]+)"`,
		`encrypt\[public_key\]=([^\s&]+)`,
	)
	s.PubKeyID = extractFirstReg(html,
		`"keyId"\s*:\s*(\d+)`,
		`"keyId"\s*:\s*"(\d+)"`,
		`"key_id"\s*:\s*"?(\d+)"?`,
		`encrypt\[key_id\]=(\d+)`,
	)
	s.PubKeyVer = extractFirstReg(html,
		`"version"\s*:\s*"(\d+)"`,
		`encrypt\[version\]=(\d+)`,
	)
	if s.PubKeyVer == "" {
		s.PubKeyVer = "5"
	}
	if s.PubKeyID == "" {
		s.PubKeyID = "6"
	}

	// Nếu r.php không có publicKey → thử lấy từ m.facebook.com/login/
	if s.PubKeyHex == "" {
		if loginKey, loginID := fetchLoginPageKey(ctx, client, ua); loginKey != "" {
			s.PubKeyHex = loginKey
			if loginID != "" && s.PubKeyID == "6" {
				s.PubKeyID = loginID
			}
		}
	}

	return nil
}

// fetchLoginPageKey lấy publicKey và keyId từ trang m.facebook.com/login/
// làm fallback khi r.php không trả về public key.
//
// Public key nằm trong Bloks DSL của trang login dưới dạng JSON-encoded HTML.
// Vì HTML là JSON string, các dấu nháy đôi bị escape thành \", dẫn đến format:
//
//	EncryptPassword, ..., KEY_ID, \"HEX_KEY\", ...
//
// Hàm delegate toàn bộ việc parse cho extractBloksEncryptKey.
// Trả về ("", "") nếu request thất bại hoặc không tìm được key.
func fetchLoginPageKey(ctx context.Context, client *http.Client, ua string) (pubKey, keyID string) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://m.facebook.com/login/", nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.6,en;q=0.5")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	html := string(body)
	return extractBloksEncryptKey(html)
}

// extractBloksEncryptKey tìm và trả về public key hex và key_id từ HTML
// chứa Bloks EncryptPassword action.
//
// Bloks DSL lưu public key dưới dạng array literal trong JSON-encoded HTML.
// Ba pattern fallback theo thứ tự ưu tiên:
//  1. Bloks DSL với JSON-encoded quotes (backslash-escaped):
//     ,\s*\d+,\s*\"[0-9a-fA-F]{64}\" — format phổ biến nhất trên login page.
//  2. Plain JSON không escape: ,\s*\d+,\s*"[0-9a-fA-F]{64}" — r.php hoặc API response.
//  3. Standard JSON field: "publicKey":"..." hoặc "public_key":"..." — format thẳng.
func extractBloksEncryptKey(html string) (pubKey, keyID string) {
	// Bloks DSL: backslash-escaped quotes → ,\s*KEY_ID,\s*\"HEX\"
	pubKey = extractFirstReg(html,
		`,\s*\d+,\s*\\"([0-9a-fA-F]{64})\\"`,    // Bloks DSL (JSON-encoded)
		`,\s*\d+,\s*"([0-9a-fA-F]{64})"`,        // plain JSON / unescaped
		`"publicKey"\s*:\s*"([0-9a-fA-F]{64})"`, // standard JSON field
		`"public_key"\s*:\s*"([0-9a-fA-F]{64})"`,
	)
	keyID = extractFirstReg(html,
		`,\s*(\d+),\s*\\"[0-9a-fA-F]{64}\\"`, // Bloks DSL
		`,\s*(\d+),\s*"[0-9a-fA-F]{64}"`,     // plain JSON
		`"keyId"\s*:\s*(\d+)`,
		`"key_id"\s*:\s*"?(\d+)"?`,
	)
	return pubKey, keyID
}

// extractFirstReg thử lần lượt các regex pattern và trả về capture group 1
// của pattern đầu tiên có match.
//
// Pattern được thử theo thứ tự cung cấp. Pattern đầu tiên trả về m[1] != ""
// thì dừng và trả về m[1]. Nếu tất cả pattern thất bại → trả về "".
func extractFirstReg(html string, patterns ...string) string {
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		m := re.FindStringSubmatch(html)
		if len(m) >= 2 && m[1] != "" {
			return m[1]
		}
	}
	return ""
}

// extractJSONStr tìm và trả về giá trị JSON string của một key trong văn bản.
//
// Unescape logic: dùng json.NewDecoder để decode JSON string value một cách
// chính xác, hỗ trợ đầy đủ JSON escape sequences thay vì replace thủ công.
func extractJSONStr(text, key string) string {
	search := `"` + key + `":`
	idx := strings.Index(text, search)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len(search):])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	var result string
	d := json.NewDecoder(strings.NewReader(rest))
	if err := d.Decode(&result); err != nil {
		return ""
	}
	return result
}

// randS sinh giá trị giả ngẫu nhiên cho tham số __s của Facebook.
//
// __s là session identifier mà Facebook nhúng vào HTML và yêu cầu gửi kèm
// trong mọi POST request Bloks để track browser session phía server.
// Định dạng: "xxxxxx:yyyyyy:zzzzzz" — 3 segment, mỗi segment gồm đúng 6
// ký tự từ bảng chữ cái lowercase a-z0-9, nối nhau bằng dấu ":".
func randS() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	seg := func() string {
		b := make([]byte, 6)
		for i := range b {
			b[i] = chars[r.Intn(len(chars))]
		}
		return string(b)
	}
	return seg() + ":" + seg() + ":" + seg()
}

// ─── Password encryption (AES-256-GCM + NaCl SealedBox) ──────────────────────
//
// Port từ C# GenerateEncPassword():
//
//	AES-256-GCM (zero IV, AAD=timestamp) + NaCl SealedBox (crypto_box_seal)
//
// Output: "#PWD_BROWSER:{version}:{timestamp}:{base64}".

// GenerateEncPassword encrypts a password using Facebook's scheme.
// publicKeyHex: 32-byte Curve25519 public key as hex string.
// keyIDStr: key_id from page (e.g. "6").
// version: encryption version from page (e.g. "5").
func GenerateEncPassword(password, publicKeyHex, keyIDStr, version string) (string, error) {
	timestamp := time.Now().Unix()

	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid public key hex: %w", err)
	}
	if len(pubKeyBytes) != 32 {
		return "", fmt.Errorf("public key must be 32 bytes, got %d", len(pubKeyBytes))
	}

	// Random 32-byte AES-256 key
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}

	// AES-256-GCM: zero IV (12 bytes), AAD = timestamp as string
	iv := make([]byte, 12) // all zeros — matches C# iv = new byte[12]
	aad := []byte(fmt.Sprintf("%d", timestamp))

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// gcm.Seal appends ciphertext then 16-byte GCM tag
	sealedData := gcm.Seal(nil, iv, []byte(password), aad)
	cipherText := sealedData[:len(sealedData)-16]
	tag := sealedData[len(sealedData)-16:]

	// NaCl SealedBox: wrap the AES key with Facebook's Curve25519 public key
	// Result is 80 bytes: 32 (ephPub) + 32 (key) + 16 (MAC)
	encryptedKey, err := naclSealedBox(aesKey, pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("sealed box: %w", err)
	}

	// Binary payload: [0x01][keyId_byte][encKeyLen_LE_2B][encryptedKey][tag_16B][ciphertext]
	keyIDUint, _ := strconv.ParseUint(keyIDStr, 10, 8)
	lenLE := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenLE, uint16(len(encryptedKey)))

	var payload []byte
	payload = append(payload, 0x01, byte(keyIDUint))
	payload = append(payload, lenLE...)
	payload = append(payload, encryptedKey...)
	payload = append(payload, tag...)
	payload = append(payload, cipherText...)

	return fmt.Sprintf("#PWD_BROWSER:%s:%d:%s",
		version, timestamp, base64.StdEncoding.EncodeToString(payload)), nil
}

// naclSealedBox implements libsodium crypto_box_seal:
//
//	ephemeral keypair + BLAKE2b nonce + NaCl box
//
// Input: plaintext message, 32-byte Curve25519 recipient public key.
// Output: ephPub(32) || nacl_box_ciphertext(len(msg)+16).
func naclSealedBox(message, recipientPublicKey []byte) ([]byte, error) {
	// Generate ephemeral private key (32 random bytes)
	var ephPriv [32]byte
	if _, err := rand.Read(ephPriv[:]); err != nil {
		return nil, err
	}

	// Compute ephemeral public key: X25519(ephPriv, basepoint)
	ephPubBytes, err := curve25519.X25519(ephPriv[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("X25519 pubkey: %w", err)
	}
	var ephPub [32]byte
	copy(ephPub[:], ephPubBytes)

	// Nonce = first 24 bytes of BLAKE2b(ephPub || recipientPK)
	h, err := blake2b.New(24, nil)
	if err != nil {
		return nil, fmt.Errorf("blake2b: %w", err)
	}
	h.Write(ephPub[:])
	h.Write(recipientPublicKey)
	var nonce [24]byte
	copy(nonce[:], h.Sum(nil))

	// Compute NaCl shared key: DH(ephPriv, recipientPK) → HSalsa20
	var recipPK [32]byte
	copy(recipPK[:], recipientPublicKey)
	var sharedKey [32]byte
	box.Precompute(&sharedKey, &recipPK, &ephPriv)

	// Encrypt using NaCl secretbox (Salsa20 + Poly1305)
	// SealAfterPrecomputation output: 16-byte MAC prepended + ciphertext
	encrypted := box.SealAfterPrecomputation(nil, message, &nonce, &sharedKey)

	// Final: ephPub(32) || encrypted(len(msg)+16)
	result := make([]byte, 32+len(encrypted))
	copy(result[:32], ephPub[:])
	copy(result[32:], encrypted)
	return result, nil
}
