// bench_test.go — Smoke / reachability bench for ALL temp mail providers.
// For each provider, calls CreateEmail with a 10-second timeout and reports:
//
//	OK  — email returned successfully
//	SKIP — provider is client-side only (email is synthesised locally, no network needed; always OK)
//	FAIL — network error, unexpected response, or context deadline exceeded
//
// Run:
//
//	go test ./internal/email/temp/ -run TestBenchAllProviders -v -timeout=300s
//
// Skip providers that require API keys (mailcx, wemakemail).
package temp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// providerCase describes one provider under test.
type providerCase struct {
	name       string
	clientSide bool // true = CreateEmail never does network; always succeeds locally
	create     func(ctx context.Context) (string, error)
}

// benchProviders returns the ordered list of all providers to bench.
// proxyStr is empty — direct connection.
func benchProviders() []providerCase {
	const proxy = ""

	// NOTE: "client-side" providers synthesise an email address locally (no HTTP call
	// in CreateEmail). We still run CreateEmail to confirm the address is valid-looking,
	// but we label them SKIP in the output since they can never be "unreachable".
	return []providerCase{
		// --- providers that do real network calls during CreateEmail ---
		{
			name: "moakt",
			create: func(ctx context.Context) (string, error) {
				return NewMoakt("", proxy).CreateEmail(ctx)
			},
		},
		{
			name: "mailtm",
			create: func(ctx context.Context) (string, error) {
				return NewMailTm(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "dropmail",
			create: func(ctx context.Context) (string, error) {
				return NewDropmail(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "guerrillamail",
			create: func(ctx context.Context) (string, error) {
				return NewGuerrillaMail(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "spam4me",
			create: func(ctx context.Context) (string, error) {
				return NewSpam4Me(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "mailtd",
			create: func(ctx context.Context) (string, error) {
				return NewMailTd("", proxy).CreateEmail(ctx)
			},
		},
		{
			name: "dismail",
			create: func(ctx context.Context) (string, error) {
				return NewDismail(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "altmails",
			create: func(ctx context.Context) (string, error) {
				return NewAltMails(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "mohmal",
			create: func(ctx context.Context) (string, error) {
				return NewMohmal(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "temporary-mail.net (guerrillamail API)",
			create: func(ctx context.Context) (string, error) {
				return NewTemporaryMailNet(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "mailermnx",
			create: func(ctx context.Context) (string, error) {
				return NewMailerMnx(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "tempforward",
			create: func(ctx context.Context) (string, error) {
				return NewTempForward(proxy).CreateEmail(ctx)
			},
		},
		{
			name: "tempemail",
			create: func(ctx context.Context) (string, error) {
				return NewTempEmailCo(proxy).CreateEmail(ctx)
			},
		},

		// --- client-side providers (CreateEmail = local synthesis, always instant) ---
		{
			name:       "mail1sec",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewMail1sec("", proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "inboxes",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewInboxes(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "mailymg",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewMailymg(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "onesecmail",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewOneSecMail(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "firetempmail",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewFireTempMail(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "fviainboxes",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewFviaInboxes(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "byomde",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewByomDe(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "dinlaan",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewDinlaan(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "cryptogmail",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewCryptoGmail(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "buslink24",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewBuslink24(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "boxmailstore",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewBoxMailStore(proxy).CreateEmail(ctx)
			},
		},
		{
			name:       "tempmail-plus",
			clientSide: true,
			create: func(ctx context.Context) (string, error) {
				return NewTempMailPlus("", proxy).CreateEmail(ctx)
			},
		},
	}
}

// TestBenchAllProviders runs CreateEmail on every provider with a 10s timeout and
// prints a result table.  It never fails the test — we just want visibility into
// which providers are reachable right now.
func TestBenchAllProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bench in -short mode (requires network)")
	}

	providers := benchProviders()

	type result struct {
		name    string
		status  string // "OK", "FAIL", "CLIENT-SIDE-OK"
		email   string
		elapsed time.Duration
		errMsg  string
	}

	results := make([]result, 0, len(providers))

	for _, p := range providers {
		p := p // capture
		t.Run(p.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			start := time.Now()
			email, err := p.create(ctx)
			elapsed := time.Since(start)

			var r result
			r.name = p.name
			r.elapsed = elapsed

			if err != nil {
				r.status = "FAIL"
				r.errMsg = err.Error()
				t.Logf("%-40s  FAIL     %6dms  %s", p.name, elapsed.Milliseconds(), err.Error())
			} else if !strings.Contains(email, "@") {
				r.status = "FAIL"
				r.errMsg = fmt.Sprintf("invalid email returned: %q", email)
				t.Logf("%-40s  FAIL     %6dms  invalid email: %s", p.name, elapsed.Milliseconds(), email)
			} else if p.clientSide {
				r.status = "CLIENT-SIDE-OK"
				r.email = email
				t.Logf("%-40s  CLIENT-SIDE-OK  %3dms  %s", p.name, elapsed.Milliseconds(), email)
			} else {
				r.status = "OK"
				r.email = email
				t.Logf("%-40s  OK       %6dms  %s", p.name, elapsed.Milliseconds(), email)
			}

			results = append(results, r)
		})
	}

	// Print a final summary banner after all sub-tests.
	t.Log("")
	t.Log("============================================================")
	t.Log("PROVIDER REACHABILITY SUMMARY")
	t.Log("============================================================")
	t.Logf("%-40s  %-16s  %8s  %s", "Provider", "Status", "Time(ms)", "Email / Error")
	t.Log("------------------------------------------------------------")
	for _, r := range results {
		if r.status == "FAIL" {
			t.Logf("%-40s  %-16s  %8d  %s", r.name, r.status, r.elapsed.Milliseconds(), r.errMsg)
		} else {
			t.Logf("%-40s  %-16s  %8d  %s", r.name, r.status, r.elapsed.Milliseconds(), r.email)
		}
	}
	t.Log("============================================================")
}
