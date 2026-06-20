package result

import (
	"fmt"
	"strings"
	"time"
)

// RegData holds the fields written to a registration result line.
// Pipe-delimited format (6 core fields): uid|pass|cookie|token|time|country
// Followed by optional email-meta suffix: |MM:user@domain:pass
type RegData struct {
	UID       string
	Password  string
	Cookie    string
	Token     string
	Email     string // stored in split-mode files; omitted from core line when empty
	Country   string
	IsNVR     bool   // newly registered (not yet verified)
	EmailMeta string // TempMail credentials suffix "MM:user@domain:pass"
}

// VerifyData holds the fields written to a verify result line.
// Pipe-delimited format (9 fields): uid|pass|2fa|cookie|token|email|fullname|time|country
type VerifyData struct {
	UID      string
	Password string
	TwoFA    string
	Cookie   string
	Token    string
	Email    string
	FullName string
	Country  string
}

// FormatReg serialises d to the pipe-delimited line stored in SuccessReg.txt (and similar files).
// The second argument is reserved for future extension and is currently unused; pass nil.
func FormatReg(d RegData, _ any) string {
	ts := time.Now().Format("2006-01-02 15:04:05")
	parts := []string{d.UID, d.Password, d.Cookie, d.Token, ts, d.Country}
	line := strings.Join(parts, "|")
	if d.EmailMeta != "" {
		line += "|" + d.EmailMeta
	}
	return line
}

// FormatVerify serialises d to the pipe-delimited line stored in SuccessVerify.txt.
// The second argument is reserved for future extension and is currently unused; pass nil.
func FormatVerify(d VerifyData, _ any) string {
	ts := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		d.UID, d.Password, d.TwoFA, d.Cookie, d.Token, d.Email, d.FullName, ts, d.Country)
}
