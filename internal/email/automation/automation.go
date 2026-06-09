// Package automation — Mail service automation interface
// Mapping từ C#: IMailServiceAutomation + MailServerAutomationInstanceType
//
// Provides a unified interface over rent and temp mail providers for the
// automated verify flow: create/buy session → login → lookup messages → extract OTPs.
//
// TODO: Port from C# IMailServiceAutomation full implementation
package automation

import "context"

// InstanceType identifies which mail service backend is in use.
// Mapping từ C#: MailServerAutomationInstanceType
type InstanceType string

const (
	InstanceZeusX      InstanceType = "zeus-x"
	InstanceDongVanFB  InstanceType = "dongvanfb"
	InstanceStore1s    InstanceType = "store1s"
	InstanceMail30s    InstanceType = "mail30s"
	InstanceMoakt      InstanceType = "moakt"
	InstanceMail1sec   InstanceType = "mail1sec"
	InstanceMohmal     InstanceType = "mohmal"
)

// Session holds a mail session retrieved from the automation layer.
// Mapping từ C#: MailSessionModel
type Session struct {
	// Email address of this mailbox
	Email string
	// Password or token for accessing the inbox
	Password string
	// Provider-specific session token (cookie/API key)
	Token    string
	// Instance identifies which backend owns this session
	Instance InstanceType
}

// OTP holds a one-time password extracted from an email message.
type OTP struct {
	Code    string
	Subject string
	From    string
}

// Service is the unified mail automation interface.
// Mapping từ C#: IMailServiceAutomation
type Service interface {
	// CreateOrBuySession retrieves or purchases a mailbox session.
	// Returns the session ready to receive messages.
	CreateOrBuySession(ctx context.Context) (*Session, error)

	// LoginIfRequired logs into the mailbox if authentication is needed.
	// For rent providers (ZeusX, DongVanFB) this may be a no-op.
	LoginIfRequired(ctx context.Context, session *Session) error

	// LookupMessages fetches unread messages for the session's email.
	// Returns raw message bodies for OTP extraction.
	LookupMessages(ctx context.Context, session *Session) ([]string, error)

	// LookupOTPs extracts OTP codes from messages matching the given subject filter.
	// subjectFilter: substring to match against message Subject (case-insensitive).
	LookupOTPs(ctx context.Context, session *Session, subjectFilter string) ([]OTP, error)

	// InstanceType returns which backend this service uses.
	InstanceType() InstanceType
}
