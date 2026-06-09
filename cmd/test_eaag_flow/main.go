// cmd/test_eaag_flow — LOCAL-ONLY inspector for the captured EAAG Messenger iOS flow.
// It does NOT send requests. It reads SuccessReg-style data and reports whether the
// account rows contain enough material to reproduce the captured EAAG flow and which
// token/header mode the current project flow would use.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Row struct {
	Raw     string
	UID     string
	Pass    string
	Cookie  string
	Token   string
	Date    string
	Country string
	Flag    string
}

var reDatr = regexp.MustCompile(`(?:^|;)\s*datr=([^;]+)`)

func tokenKind(tok string) string {
	switch {
	case strings.HasPrefix(tok, "EAAG"):
		return "EAAG"
	case strings.HasPrefix(tok, "EAAAAAY"):
		return "EAAAAAY(iOS user-token)"
	case strings.HasPrefix(tok, "EAAAAU"):
		return "EAAAAU(Android user-token)"
	case strings.HasPrefix(tok, "EAA"):
		return "EAA(other)"
	case tok == "":
		return "EMPTY"
	default:
		if len(tok) > 8 {
			return tok[:8] + "..."
		}
		return tok
	}
}

func parseLine(line string) (Row, bool) {
	f := strings.Split(line, "|")
	if len(f) < 7 {
		return Row{Raw: line}, false
	}
	r := Row{Raw: line, UID: f[0], Pass: f[1], Cookie: f[2], Token: f[3], Date: f[4], Country: f[5], Flag: f[6]}
	if len(r.UID) < 8 || r.UID == "2147483647" {
		return r, false
	}
	return r, true
}

func readRows(path string, limit int) ([]Row, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	var rows []Row
	s := bufio.NewScanner(fp)
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 10*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if r, ok := parseLine(line); ok {
			rows = append(rows, r)
			if limit > 0 && len(rows) >= limit {
				break
			}
		}
	}
	return rows, s.Err()
}

func docSummary(docDir string) {
	fmt.Println("\n=== Captured doc flow summary ===")
	files, _ := filepath.Glob(filepath.Join(docDir, "*request*.txt"))
	sort.Strings(files)
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(b)
		name := filepath.Base(p)
		friendly := findHeader(s, "X-FB-Friendly-Name")
		auth := findHeader(s, "Authorization")
		kind := "unknown"
		if strings.Contains(auth, "OAuth EAAG") {
			kind = "EAAG"
		} else if strings.Contains(auth, "|") {
			kind = "app-token"
		}
		fmt.Printf("- %-48s auth=%-8s friendly=%s\n", name, kind, friendly)
		if strings.Contains(s, "spectra_guardian_token") {
			fmt.Println("  body: has spectra_guardian_token")
		}
		if strings.Contains(s, "sso_accounts_auth_data") {
			fmt.Println("  body: has sso_accounts_auth_data")
		}
		if strings.Contains(s, "crypted_user_id") {
			fmt.Println("  body: has crypted_user_id")
		}
		if strings.Contains(s, "reg_info") {
			fmt.Println("  body: has reg_info")
		}
	}
}

func findHeader(s, key string) string {
	prefix := strings.ToLower(key) + ":"
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), prefix) {
			v := strings.TrimSpace(line[len(key)+1:])
			if len(v) > 90 {
				return v[:90] + "..."
			}
			return v
		}
	}
	return ""
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/test_eaag_flow <SuccessReg.txt> [limit] [docDir]")
		os.Exit(2)
	}
	path := os.Args[1]
	limit := 5
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &limit)
	}
	docDir := `E:\WEMAKE\DocWeMake\FlowRegFB_IOS\VerMessByTokenEaag`
	if len(os.Args) >= 4 {
		docDir = os.Args[3]
	}

	rows, err := readRows(path, limit)
	if err != nil {
		fmt.Println("read:", err)
		os.Exit(1)
	}
	fmt.Printf("=== EAAG flow local inspector | rows=%d | source=%s ===\n", len(rows), path)
	for i, r := range rows {
		datr := ""
		if m := reDatr.FindStringSubmatch(r.Cookie); len(m) > 1 {
			datr = m[1]
		}
		fmt.Printf("\n[#%d] uid=%s country=%s flag=%s\n", i+1, r.UID, r.Country, r.Flag)
		fmt.Printf("  password: %t len=%d\n", r.Pass != "", len(r.Pass))
		fmt.Printf("  cookie: datr=%t c_user=%t xs=%t fr=%t locale=%t\n", datr != "", strings.Contains(r.Cookie, "c_user="), strings.Contains(r.Cookie, "xs="), strings.Contains(r.Cookie, "fr="), strings.Contains(r.Cookie, "locale="))
		fmt.Printf("  stored token: %s len=%d\n", tokenKind(r.Token), len(r.Token))
		fmt.Println("  captured EAAG flow prerequisites:")
		fmt.Println("    send_login_request: needs uid+password+device/family/waterfall; login response should provide user-token + crypted_user_id")
		fmt.Println("    bottomsheet/change_email/add-mail: need crypted_user_id + AAC + reg_info/flow context from login response")
		fmt.Println("  data-row suitability:")
		fmt.Printf("    can attempt login-first if run live: %t\n", r.UID != "" && r.Pass != "")
		fmt.Printf("    has existing EAAG token in file: %t\n", strings.HasPrefix(r.Token, "EAAG"))
		fmt.Printf("    has existing iOS EAAAAAY token in file: %t\n", strings.HasPrefix(r.Token, "EAAAAAY"))
		fmt.Println("  current project flow note:")
		fmt.Println("    verify/ios/iosmess currently uses app-token for post-login steps unless patched to SendStepWithToken(loginToken).")
	}
	docSummary(docDir)
	fmt.Println("\nRESULT: local inspection complete; no network requests were sent.")
}
