// Package model — AppSettings versioned model cho NullCoreSummer
// Phase 1: nền settings sạch, backward-compatible với config cũ qua adapter
package model

import "encoding/json"

// CurrentVersion — phiên bản hiện tại của AppSettings schema
const CurrentVersion = 1

// AppSettings — root settings model, versioned, profile-aware
type AppSettings struct {
	Version         int             `json:"version"`
	ActiveProfileID string          `json:"activeProfileId"`
	Global          GlobalSettings  `json:"global"`
	Profiles        []Profile       `json:"profiles"`
	SecretsRef      SecretReference `json:"secretsRef"`
}

// GetActiveProfile trả về con trỏ đến profile đang active.
// Ưu tiên profile có ID trùng với ActiveProfileID.
// Fallback: trả về profile đầu tiên nếu ActiveProfileID không match.
// Trả về nil nếu Profiles rỗng — caller phải kiểm tra nil trước khi dùng.
func (a *AppSettings) GetActiveProfile() *Profile {
	for i := range a.Profiles {
		if a.Profiles[i].ID == a.ActiveProfileID {
			return &a.Profiles[i]
		}
	}
	if len(a.Profiles) > 0 {
		return &a.Profiles[0]
	}
	return nil
}

// UpsertProfile thêm profile mới hoặc thay thế profile đã có (theo ID).
// p: profile cần thêm hoặc cập nhật.
// Nếu tồn tại profile có p.ID trùng → replace in-place.
// Nếu không tìm thấy → append vào cuối slice Profiles.
func (a *AppSettings) UpsertProfile(p Profile) {
	for i := range a.Profiles {
		if a.Profiles[i].ID == p.ID {
			a.Profiles[i] = p
			return
		}
	}
	a.Profiles = append(a.Profiles, p)
}

// GlobalSettings — cài đặt áp dụng cho toàn app, không theo profile
type GlobalSettings struct {
	LoginPlatform  string `json:"loginPlatform"`
	LoginMethod    int    `json:"loginMethod"`
	SaveRunColumn  bool   `json:"saveRunColumn"`
	BackupDB       bool   `json:"backupDB"`
	CloseAfterDone bool   `json:"closeAfterDone"`
}

// Profile — một cấu hình chạy có tên, tách biệt với global settings
type Profile struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Runtime     RuntimeSettings  `json:"runtime"`
	Account     AccountSettings  `json:"account"`
	Proxy       ProxySettings    `json:"proxy"`
	Verify      VerifySettings   `json:"verify"`
	Register    RegisterSettings `json:"register"`
	Mail        MailSettings     `json:"mail"`
	Captcha     CaptchaSettings  `json:"captcha"`
	Output      OutputSettings   `json:"output"`
	Device      DeviceSettings   `json:"device"`
	Interaction json.RawMessage  `json:"interaction,omitempty"` // InteractionConfig (Thiết lập chạy)
}

// RuntimeSettings — thread/timing config
type RuntimeSettings struct {
	ThreadRequest    int  `json:"threadRequest"`
	DelayRequest     int  `json:"delayRequest"`
	DelayThread      int  `json:"delayThread"`
	ApiCheckIp       int  `json:"apiCheckIp"`
	ThreadCheckInfo  int  `json:"threadCheckInfo"`
	DelayChangeIp    int  `json:"delayChangeIp"`
	CheckIpBeforeRun bool `json:"checkIpBeforeRun"`
}

// AccountSettings — nguồn tài khoản (folder hoặc API)
type AccountSettings struct {
	Source     string             `json:"source"` // "folder" | "api"
	FolderPath string             `json:"folderPath"`
	CloneHV    CloneHVCredentials `json:"cloneHv"`
}

// CloneHVCredentials — thông tin đăng nhập CloneHV API
// Password: TODO(secrets-phase4) chuyển sang SecretStore khi có encryption
type CloneHVCredentials struct {
	Enabled   bool   `json:"enabled"`
	Username  string `json:"username"`
	Password  string `json:"password"` // TODO(secrets-phase4): move to SecretStore
	ProductID string `json:"productId"`
	Amount    int    `json:"amount"`
}

// ProxySettings — cấu hình proxy/network
type ProxySettings struct {
	Provider  string                     `json:"provider"`
	ProxyList string                     `json:"proxyList"`
	ProxyType string                     `json:"proxyType"`
	Providers map[string]ProxyProviderCfg `json:"providers"`
}

// ProxyProviderCfg — cấu hình cho từng provider proxy
type ProxyProviderCfg struct {
	Keys        string `json:"keys,omitempty"`
	ServiceURL  string `json:"serviceUrl,omitempty"`
	Type        string `json:"type,omitempty"`
	ThreadPerIP int    `json:"threadPerIp,omitempty"`
	RunType     string `json:"runType,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
	List        string `json:"list,omitempty"`
}

// VerifySettings — cấu hình kiểm tra tài khoản
type VerifySettings struct {
	Enabled           bool `json:"enabled"`
	CheckLiveDie      bool `json:"checkLiveDie"`
	TimeDelayCheck    int  `json:"timeDelayCheck"`
	TimeDelaySendCode int  `json:"timeDelaySendCode"`
	SendAgainCode     bool `json:"sendAgainCode"`
}

// RegisterSettings — cấu hình tạo tài khoản tự động
type RegisterSettings struct {
	Enabled    bool   `json:"enabled"`
	Type       string `json:"type"` // "normal" | "tut"
	CookieList string `json:"cookieList"`
	OutputPath string `json:"outputPath"`
}

// MailSettings — cấu hình email provider
type MailSettings struct {
	Provider  string                    `json:"provider"`
	MailList  string                    `json:"mailList"`
	Providers map[string]MailProviderCfg `json:"providers"`
}

// MailProviderCfg — cấu hình cho từng email provider
// APIKey: TODO(secrets-phase4) chuyển sang SecretStore
type MailProviderCfg struct {
	APIKey      string `json:"apiKey,omitempty"`      // TODO(secrets-phase4): move to SecretStore
	AccountCode string `json:"accountCode,omitempty"`
	AccountType string `json:"accountType,omitempty"`
	ProductID   string `json:"productId,omitempty"`
	ProductSlug string `json:"productSlug,omitempty"`
}

// CaptchaSettings — cấu hình captcha solver
// Keys: TODO(secrets-phase4) chuyển sang SecretStore
type CaptchaSettings struct {
	Provider string            `json:"provider"`
	Keys     map[string]string `json:"keys"` // TODO(secrets-phase4): move to SecretStore
}

// OutputSettings — đường dẫn output
type OutputSettings struct {
	VerifyPath   string `json:"verifyPath"`
	RegisterPath string `json:"registerPath"`
}

// DeviceSettings — UA list và device fingerprint
type DeviceSettings struct {
	UAList      string            `json:"uaList"`                // mỗi dòng một UA string (active pool, dùng bởi runner)
	UaPools     map[string]string `json:"uaPools,omitempty"`     // pool key → UA text (manual mode)
	UaPoolKey   string            `json:"uaPoolKey,omitempty"`   // pool đang chọn cho lần chạy này
	UaPoolFiles map[string]string `json:"uaPoolFiles,omitempty"` // pool key → file path (file mode)
}

// SecretReference — điểm tích hợp secret store trong tương lai.
// Phase 1: no-op placeholder (provider = "" = secrets stored inline).
// Phase 4+: provider = "keychain" | "vault" | "env", keyRefs maps setting-key → secret-key.
type SecretReference struct {
	Provider string            `json:"provider"` // "" | "keychain" | "vault" | "env"
	KeyRefs  map[string]string `json:"keyRefs"`
}
