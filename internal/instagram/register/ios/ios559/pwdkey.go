// pwdkey.go — pwd_key_fetch GET + WILDE password encryption for ios562.
//
// Gọi GET graph.facebook.com/pwd_key_fetch → RSA public key + key_id,
// sau đó dùng iosapp562.EncryptPasswordWILDE để build #PWD_WILDE:2:{ts}:{b64}.
// Fallback về #PWD_FB4A:0:{ts}:{plaintext} nếu fetch lỗi.
package ios559

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const pwdKeyFetchURL = "https://graph.facebook.com/pwd_key_fetch"

var (
	rePwdPublicKey = regexp.MustCompile(`"public_key":"(.*?)"`)
	rePwdKeyID     = regexp.MustCompile(`"key_id":(\d+)`)
)

// encryptPasswordForReg fetches RSA public key và mã hóa password theo #PWD_WILDE:2.
// Fallback về #PWD_FB4A:0 (plaintext) nếu pwd_key_fetch thất bại.
func encryptPasswordForReg(ctx context.Context, sess *session, p IOSProfile, password string) string {
	pubKey, keyID, ok := fetchPwdKey(ctx, sess, p)
	if !ok {
		ts := time.Now().Unix()
		return fmt.Sprintf("%s%d:%s", pwdPrefix, ts, password)
	}
	enc := encryptPasswordWILDE(password, pubKey, keyID)
	if enc == "" {
		ts := time.Now().Unix()
		return fmt.Sprintf("%s%d:%s", pwdPrefix, ts, password)
	}
	return enc
}

// fetchPwdKey gọi GET /pwd_key_fetch → (publicKey, keyID, ok).
func fetchPwdKey(ctx context.Context, sess *session, p IOSProfile) (string, int, bool) {
	q := url.Values{}
	q.Set("app_version", fbBuildNum)
	q.Set("device_id", p.DeviceID)
	q.Set("fb_api_caller_class", "FBPasswordEncryptionKeyFetchRequest")
	q.Set("fb_api_req_friendly_name", "FBPasswordEncryptionKeyFetchRequest:networkRequest")
	q.Set("flow", "controller_initialization")
	q.Set("format", "json")
	q.Set("locale", p.Locale)
	q.Set("machine_id", p.MachineID)
	q.Set("pretty", "0")
	q.Set("sdk", "ios")
	q.Set("sdk_version", "3")
	q.Set("version", "2")

	body, err := sess.httpGet(ctx, pwdKeyFetchURL+"?"+q.Encode(), buildPwdKeyHeaders(p))
	if err != nil || !strings.Contains(body, `"public_key"`) {
		return "", 0, false
	}

	pkMatch := rePwdPublicKey.FindStringSubmatch(body)
	idMatch := rePwdKeyID.FindStringSubmatch(body)
	if len(pkMatch) < 2 || len(idMatch) < 2 {
		return "", 0, false
	}

	var keyID int
	fmt.Sscanf(idMatch[1], "%d", &keyID)
	if keyID == 0 || pkMatch[1] == "" {
		return "", 0, false
	}
	return pkMatch[1], keyID, true
}

// buildPwdKeyHeaders — minimal GET headers cho /pwd_key_fetch (ios562 style).
func buildPwdKeyHeaders(p IOSProfile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"6628568379","purpose":"none","retry_attempt":"0"}}`
	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip, deflate"},
		{"accept", "*/*"},
		{"connection", "keep-alive"},
		{"x-fb-conn-uuid-client", connUUID()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-meta-zca", `{"e": {"c":7}}`},
		{"authorization", "OAuth " + oauthToken},
		{"x-fb-sim-hni", p.Sim.HNI},
		{"x-fb-connection-type", p.ConnType},
		{"x-fb-integrity-machine-id", p.MachineID},
		{"x-fb-device-id", p.DeviceID},
		{"x-fb-friendly-name", "FBPasswordEncryptionKeyFetchRequest:networkRequest"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
		{"x-cloud-trust-token", p.CloudTrustID},
	}
}

// ─── WILDE password encryption ────────────────────────────────────────────────

// encryptPasswordWILDE mã hóa password theo #PWD_WILDE:2 (RSA+AES-GCM).
// blob = [0x01][keyID][iv 12B][len_uint16_LE][enc_rand_key][tag 16B][enc_pw]
// output = "#PWD_WILDE:2:{unix_ts}:{base64(blob)}"
func encryptPasswordWILDE(password, publicKey string, keyID int) string {
	randKey := make([]byte, 32)
	iv := make([]byte, 12)
	if _, err := rand.Read(randKey); err != nil {
		return ""
	}
	if _, err := rand.Read(iv); err != nil {
		return ""
	}
	pubKey, err := parseRSAPublicKeyWILDE(publicKey)
	if err != nil {
		return ""
	}
	encryptedRandKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, randKey)
	if err != nil {
		return ""
	}
	ts := time.Now().Unix()
	timeBytes := []byte(fmt.Sprintf("%d", ts))
	block, err := aes.NewCipher(randKey)
	if err != nil {
		return ""
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	sealed := aesgcm.Seal(nil, iv, []byte(password), timeBytes)
	ctLen := len(sealed) - 16
	if ctLen < 0 {
		return ""
	}
	var buf []byte
	buf = append(buf, 0x01)
	buf = append(buf, byte(keyID))
	buf = append(buf, iv...)
	lenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenBytes, uint16(len(encryptedRandKey)))
	buf = append(buf, lenBytes...)
	buf = append(buf, encryptedRandKey...)
	buf = append(buf, sealed[ctLen:]...)
	buf = append(buf, sealed[:ctLen]...)
	return fmt.Sprintf("#PWD_WILDE:2:%d:%s", ts, base64.StdEncoding.EncodeToString(buf))
}

func parseRSAPublicKeyWILDE(publicKey string) (*rsa.PublicKey, error) {
	k := strings.ReplaceAll(publicKey, `\n`, "\n")
	k = strings.ReplaceAll(k, `\/`, "/")
	var der []byte
	if strings.Contains(k, "-----BEGIN PUBLIC KEY-----") {
		block, _ := pem.Decode([]byte(k))
		if block == nil {
			return nil, fmt.Errorf("pem decode failed")
		}
		der = block.Bytes
	} else {
		k = strings.ReplaceAll(k, "-----BEGIN PUBLIC KEY-----", "")
		k = strings.ReplaceAll(k, "-----END PUBLIC KEY-----", "")
		k = strings.TrimSpace(strings.ReplaceAll(k, "\n", ""))
		var err error
		der, err = base64.StdEncoding.DecodeString(k)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse pkix: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not RSA key")
	}
	return rsaKey, nil
}
