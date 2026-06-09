package temp

// snapshot_impl.go — Snapshotter + Releaser cho 38 temp email providers.
//
// Pattern chung: temp providers dùng session-based mail (free, không refund).
// Snapshot encode email + provider-specific token/session id để verify Restore
// và đọc inbox đã có.
//
// Tất cả Release returns nil — temp providers không có refund API (mail tự
// expire sau TTL ngắn).
//
// Variations:
//   1. EmailOnly: chỉ cần address để fetch (~10 providers)
//   2. Email+Token: email + 1 session/csrf token (~15 providers)
//   3. Email+User+Domain: providers split user/domain (~5 providers)
//   4. Custom: dropmail (token+sessID), mailtm (pass+token), tempmailso (POW state)

import (
	"context"
	"encoding/json"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

// emailOnlySnap — JSON shape cho email-only providers.
type emailOnlySnap struct {
	Email string `json:"email"`
}

// emailTokenSnap — email + 1 token string.
type emailTokenSnap struct {
	Email string `json:"email"`
	Token string `json:"token,omitempty"`
}

// emailUserDomainSnap — providers split user + domain.
type emailUserDomainSnap struct {
	Email  string `json:"email"`
	User   string `json:"user,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// marshalSnap — JSON encode helper. Trả "" + nil nếu Email empty.
func marshalSnap(v any, email string) (string, error) {
	if email == "" {
		return "", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ─── EmailOnly providers (10) ──────────────────────────────────────────────

func (p *Buslink24) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *Buslink24) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *Buslink24) Release(ctx context.Context) error { return nil }

func (p *BoxMailStore) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *BoxMailStore) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *BoxMailStore) Release(ctx context.Context) error { return nil }

func (p *CryptoGmail) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *CryptoGmail) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *CryptoGmail) Release(ctx context.Context) error { return nil }

func (p *Dinlaan) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *Dinlaan) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *Dinlaan) Release(ctx context.Context) error { return nil }

func (p *FireTempMail) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *FireTempMail) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *FireTempMail) Release(ctx context.Context) error { return nil }

func (p *Inboxes) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *Inboxes) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *Inboxes) Release(ctx context.Context) error { return nil }

func (p *Mailymg) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *Mailymg) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *Mailymg) Release(ctx context.Context) error { return nil }

func (p *Mohmal) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *Mohmal) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *Mohmal) Release(ctx context.Context) error { return nil }

func (p *TempEmailCo) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *TempEmailCo) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *TempEmailCo) Release(ctx context.Context) error { return nil }

func (p *TenMinuteMail) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *TenMinuteMail) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *TenMinuteMail) Release(ctx context.Context) error { return nil }

func (p *TmpInbox) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *TmpInbox) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *TmpInbox) Release(ctx context.Context) error { return nil }

// ─── Email+Token providers (15) ────────────────────────────────────────────

func (p *AltMails) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.csrfToken}, p.email)
}
func (p *AltMails) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.csrfToken = s.Email, s.Token
	return nil
}
func (p *AltMails) Release(ctx context.Context) error { return nil }

func (p *Dismail) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.mailID}, p.email)
}
func (p *Dismail) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.mailID = s.Email, s.Token
	return nil
}
func (p *Dismail) Release(ctx context.Context) error { return nil }

func (p *MailCx) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.apiToken}, p.email)
}
func (p *MailCx) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.apiToken = s.Email, s.Token
	return nil
}
func (p *MailCx) Release(ctx context.Context) error { return nil }

// mailTdSnap — email + token + accountID (mail.td cần cả 3 để fetch messages).
type mailTdSnap struct {
	Email     string `json:"email"`
	Token     string `json:"token,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}

func (p *MailTd) Snapshot() (string, error) {
	b, err := json.Marshal(mailTdSnap{Email: p.email, Token: p.token, AccountID: p.accountID})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *MailTd) Restore(c string) error {
	var s mailTdSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token, p.accountID = s.Email, s.Token, s.AccountID
	return nil
}
func (p *MailTd) Release(ctx context.Context) error { return nil }

func (p *TempForward) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.token}, p.email)
}
func (p *TempForward) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token = s.Email, s.Token
	return nil
}
func (p *TempForward) Release(ctx context.Context) error { return nil }

func (p *TempMailLol) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.token}, p.email)
}
func (p *TempMailLol) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token = s.Email, s.Token
	return nil
}
func (p *TempMailLol) Release(ctx context.Context) error { return nil }

func (p *TempMailOrg) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.token}, p.email)
}
func (p *TempMailOrg) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token = s.Email, s.Token
	return nil
}
func (p *TempMailOrg) Release(ctx context.Context) error { return nil }

func (p *TempMailOrgPremium) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.sid}, p.email)
}
func (p *TempMailOrgPremium) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.sid = s.Email, s.Token
	return nil
}
func (p *TempMailOrgPremium) Release(ctx context.Context) error { return nil }

func (p *TempMailTo) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.csrfToken}, p.email)
}
func (p *TempMailTo) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.csrfToken = s.Email, s.Token
	return nil
}
func (p *TempMailTo) Release(ctx context.Context) error { return nil }

func (p *TempoMintraccoon) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.token}, p.email)
}
func (p *TempoMintraccoon) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token = s.Email, s.Token
	return nil
}
func (p *TempoMintraccoon) Release(ctx context.Context) error { return nil }

func (p *OneSecEmail) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.csrfToken}, p.email)
}
func (p *OneSecEmail) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.csrfToken = s.Email, s.Token
	return nil
}
func (p *OneSecEmail) Release(ctx context.Context) error { return nil }

func (p *TempMail100) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.token}, p.email)
}
func (p *TempMail100) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token = s.Email, s.Token
	return nil
}
func (p *TempMail100) Release(ctx context.Context) error { return nil }

// ─── Email+User+Domain providers (5) ───────────────────────────────────────

func (p *ByomDe) Snapshot() (string, error) {
	return marshalSnap(emailUserDomainSnap{Email: p.email, User: p.localPart}, p.email)
}
func (p *ByomDe) Restore(c string) error {
	var s emailUserDomainSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.localPart = s.Email, s.User
	return nil
}
func (p *ByomDe) Release(ctx context.Context) error { return nil }

func (p *FviaInboxes) Snapshot() (string, error) {
	return marshalSnap(emailUserDomainSnap{Email: p.email, User: p.user, Domain: p.domain}, p.email)
}
func (p *FviaInboxes) Restore(c string) error {
	var s emailUserDomainSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.user, p.domain = s.Email, s.User, s.Domain
	return nil
}
func (p *FviaInboxes) Release(ctx context.Context) error { return nil }

func (p *MailTempCom) Snapshot() (string, error) {
	return marshalSnap(emailUserDomainSnap{Email: p.email, User: p.user, Domain: p.domain}, p.email)
}
func (p *MailTempCom) Restore(c string) error {
	var s emailUserDomainSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.user, p.domain = s.Email, s.User, s.Domain
	return nil
}
func (p *MailTempCom) Release(ctx context.Context) error { return nil }

func (p *OneSecMail) Snapshot() (string, error) {
	return marshalSnap(emailUserDomainSnap{Email: p.email, User: p.user, Domain: p.domain}, p.email)
}
func (p *OneSecMail) Restore(c string) error {
	var s emailUserDomainSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.user, p.domain = s.Email, s.User, s.Domain
	return nil
}
func (p *OneSecMail) Release(ctx context.Context) error { return nil }

func (p *Mail1sec) Snapshot() (string, error) {
	return marshalSnap(emailUserDomainSnap{Email: p.email, User: p.user, Domain: p.domain}, p.email)
}
func (p *Mail1sec) Restore(c string) error {
	var s emailUserDomainSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.user, p.domain = s.Email, s.User, s.Domain
	return nil
}
func (p *Mail1sec) Release(ctx context.Context) error { return nil }

// ─── Special providers (custom shapes) ─────────────────────────────────────

// Dropmail — token (random hex16 cho endpoint URL) + sessID + email.
type dropmailSnap struct {
	Email  string `json:"email"`
	Token  string `json:"token"`
	SessID string `json:"sess_id"`
}

func (p *Dropmail) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(dropmailSnap{Email: p.email, Token: p.token, SessID: p.sessID})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *Dropmail) Restore(c string) error {
	var s dropmailSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.token, p.sessID = s.Email, s.Token, s.SessID
	return nil
}
func (p *Dropmail) Release(ctx context.Context) error { return nil }

// MailTm — email + pass + token (JWT) + msgBase URL.
type mailTmSnap struct {
	Email   string `json:"email"`
	Pass    string `json:"pass"`
	Token   string `json:"token"`
	MsgBase string `json:"msg_base"`
}

func (p *MailTm) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(mailTmSnap{Email: p.email, Pass: p.pass, Token: p.token, MsgBase: p.msgBase})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *MailTm) Restore(c string) error {
	var s mailTmSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.pass, p.token, p.msgBase = s.Email, s.Pass, s.Token, s.MsgBase
	return nil
}
func (p *MailTm) Release(ctx context.Context) error { return nil }

// TempMailSo — POW (proof-of-work) state. Hash + nonce phải save để Restore
// có thể dùng cùng challenge.
type tempMailSoSnap struct {
	Email       string `json:"email"`
	Nonce       string `json:"nonce"`
	PowTimeSpan int64  `json:"pow_ts"`
	PowResult   int    `json:"pow_result"`
}

func (p *TempMailSo) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	// PoW fields đã bỏ (API mới không dùng PoW), chỉ lưu email
	b, err := json.Marshal(tempMailSoSnap{Email: p.email})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *TempMailSo) Restore(c string) error {
	var s tempMailSoSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *TempMailSo) Release(ctx context.Context) error { return nil }

// Spam4Me — sidToken + alias.
type spam4MeSnap struct {
	Email    string `json:"email"`
	SidToken string `json:"sid_token"`
	Alias    string `json:"alias"`
}

func (p *Spam4Me) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(spam4MeSnap{Email: p.email, SidToken: p.sidToken, Alias: p.alias})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *Spam4Me) Restore(c string) error {
	var s spam4MeSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.sidToken, p.alias = s.Email, s.SidToken, s.Alias
	return nil
}
func (p *Spam4Me) Release(ctx context.Context) error { return nil }

// GuerrillaMailWWW — sidToken + alias + email (cùng pattern Spam4Me).
func (p *GuerrillaMailWWW) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(spam4MeSnap{Email: p.email, SidToken: p.sidToken, Alias: p.alias})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *GuerrillaMailWWW) Restore(c string) error {
	var s spam4MeSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.sidToken, p.alias = s.Email, s.SidToken, s.Alias
	return nil
}
func (p *GuerrillaMailWWW) Release(ctx context.Context) error { return nil }

// GuerrillaMail (temporary_mail_net) — chỉ sidToken + email (không có alias).
func (p *GuerrillaMail) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.sidToken}, p.email)
}
func (p *GuerrillaMail) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.sidToken = s.Email, s.Token
	return nil
}
func (p *GuerrillaMail) Release(ctx context.Context) error { return nil }

// MailerMnx — email + domains[] (random pick).
type mailerMnxSnap struct {
	Email   string   `json:"email"`
	Domains []string `json:"domains,omitempty"`
}

func (p *MailerMnx) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(mailerMnxSnap{Email: p.email, Domains: p.domains})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *MailerMnx) Restore(c string) error {
	var s mailerMnxSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.domains = s.Email, s.Domains
	return nil
}
func (p *MailerMnx) Release(ctx context.Context) error { return nil }

// TempMailPlus — domains + user + domain + email.
type tempMailPlusSnap struct {
	Email   string   `json:"email"`
	User    string   `json:"user"`
	Domain  string   `json:"domain"`
	Domains []string `json:"domains,omitempty"`
}

func (p *TempMailPlus) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(tempMailPlusSnap{
		Email: p.email, User: p.user, Domain: p.domain, Domains: p.domains,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *TempMailPlus) Restore(c string) error {
	var s tempMailPlusSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.user, p.domain, p.domains = s.Email, s.User, s.Domain, s.Domains
	return nil
}
func (p *TempMailPlus) Release(ctx context.Context) error { return nil }

// Moakt — domain + customUsername (FmUserTmpMail).
type moaktSnap struct {
	Email          string `json:"email"`
	Domain         string `json:"domain"`
	CustomUsername string `json:"custom_username,omitempty"`
}

func (p *Moakt) Snapshot() (string, error) {
	if p.email == "" {
		return "", nil
	}
	b, err := json.Marshal(moaktSnap{
		Email: p.email, Domain: p.domain, CustomUsername: p.customUsername,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (p *Moakt) Restore(c string) error {
	var s moaktSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.domain, p.customUsername = s.Email, s.Domain, s.CustomUsername
	return nil
}
func (p *Moakt) Release(ctx context.Context) error { return nil }

// PriyoEmail — email + apiKey (apiKey từ config; snapshot vẫn giữ để đảm bảo
// Restore work nếu config thay đổi giữa reg và verify).
func (p *PriyoEmail) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.apiKey}, p.email)
}
func (p *PriyoEmail) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.apiKey = s.Email, s.Token
	return nil
}
func (p *PriyoEmail) Release(ctx context.Context) error { return nil }

// I2bMail — email only.
func (p *I2bMail) Snapshot() (string, error) {
	return marshalSnap(emailOnlySnap{Email: p.email}, p.email)
}
func (p *I2bMail) Restore(c string) error {
	var s emailOnlySnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email = s.Email
	return nil
}
func (p *I2bMail) Release(ctx context.Context) error { return nil }

// VietXF — email + apiKey.
func (p *VietXF) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.apiKey}, p.email)
}
func (p *VietXF) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.apiKey = s.Email, s.Token
	return nil
}
func (p *VietXF) Release(ctx context.Context) error { return nil }

// MailHV — email + apiKey.
func (p *MailHV) Snapshot() (string, error) {
	return marshalSnap(emailTokenSnap{Email: p.email, Token: p.apiKey}, p.email)
}
func (p *MailHV) Restore(c string) error {
	var s emailTokenSnap
	if err := json.Unmarshal([]byte(c), &s); err != nil {
		return err
	}
	p.email, p.apiKey = s.Email, s.Token
	return nil
}
func (p *MailHV) Release(ctx context.Context) error { return nil }
