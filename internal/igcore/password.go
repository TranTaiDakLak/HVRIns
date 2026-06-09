// password.go — mã hóa password theo scheme #PWD_INSTAGRAM:4.
//
// Cơ chế (IG iOS v4, giống IG Android):
//  1. random AES-256 key (32B) + nonce (12B)
//  2. AES-256-GCM mã hóa password, AAD = unix_timestamp string
//  3. RSA-PKCS1v15 mã hóa AES key bằng public key server (key-id từ qe/sync)
//  4. blob = [0x01][key_id][nonce 12B][len_uint16_LE(encKey)][encKey][gcm_tag 16B][ciphertext]
//  5. output = "#PWD_INSTAGRAM:4:<unix_ts>:<base64(blob)>"
package igcore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"strconv"
)

func encryptPasswordInstagram(password, pubKeyB64, keyIDStr string) (string, error) {
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return "", fmt.Errorf("key_id không hợp lệ: %w", err)
	}
	pub, err := parseRSAPubKey(pubKeyB64)
	if err != nil {
		return "", fmt.Errorf("parse pub-key: %w", err)
	}

	aesKey := make([]byte, 32)
	nonce := make([]byte, 12)
	if _, err := rand.Read(aesKey); err != nil {
		return "", err
	}
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ts := nowUnix()
	tsBytes := []byte(strconv.FormatInt(ts, 10))

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	// Seal: ciphertext||tag, AAD = timestamp string.
	sealed := gcm.Seal(nil, nonce, []byte(password), tsBytes)
	ctLen := len(sealed) - 16
	if ctLen < 0 {
		return "", fmt.Errorf("sealed quá ngắn")
	}
	ciphertext := sealed[:ctLen]
	tag := sealed[ctLen:]

	encKey, err := rsa.EncryptPKCS1v15(rand.Reader, pub, aesKey)
	if err != nil {
		return "", fmt.Errorf("rsa encrypt: %w", err)
	}

	var buf []byte
	buf = append(buf, 0x01)
	buf = append(buf, byte(keyID))
	buf = append(buf, nonce...)
	lenB := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenB, uint16(len(encKey)))
	buf = append(buf, lenB...)
	buf = append(buf, encKey...)
	buf = append(buf, tag...)
	buf = append(buf, ciphertext...)

	return fmt.Sprintf("#PWD_INSTAGRAM:4:%d:%s", ts, base64.StdEncoding.EncodeToString(buf)), nil
}

// parseRSAPubKey nhận base64 của PEM "-----BEGIN PUBLIC KEY-----".
func parseRSAPubKey(b64 string) (*rsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 outer: %w", err)
	}
	blk, _ := pem.Decode(der)
	if blk == nil {
		return nil, fmt.Errorf("pem decode fail")
	}
	key, err := x509.ParsePKIXPublicKey(blk.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse pkix: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("không phải RSA key")
	}
	return rsaKey, nil
}
