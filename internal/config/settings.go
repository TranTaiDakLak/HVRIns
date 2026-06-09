// Package config — Đọc/ghi JSON settings files
// Mapping từ WeBM JSON_Settings class
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var mu sync.RWMutex

// Load đọc JSON file vào struct
func Load(filename string, v interface{}) error {
	mu.RLock()
	defer mu.RUnlock()

	path := getPath(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File chưa tồn tại — dùng default
		}
		return err
	}

	return json.Unmarshal(data, v)
}

// Save ghi struct thành JSON file
func Save(filename string, v interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	path := getPath(filename)

	// Tạo thư mục nếu chưa có
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// 0600 — settings có thể chứa credentials/keys, không cho user khác trên máy đọc
	return os.WriteFile(path, data, 0600)
}

// getPath trả về đường dẫn file JSON cho một settings file.
// Tất cả settings files được lưu trong thư mục Config/Settings/ cạnh binary.
//
// filename: tên file không có extension (ví dụ: "general", "interaction").
func getPath(filename string) string {
	return filepath.Join("Config", "Settings", filename+".json")
}
