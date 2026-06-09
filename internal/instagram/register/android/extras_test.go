package android

import "testing"

func TestParseRegisterResponsePrefersUserAccessToken(t *testing.T) {
	body := `{"token":"EAARxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","access_token":"EAAAAUyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy","cookies":[{"name":"c_user","value":"61512345678901"},{"name":"xs","value":"xs1"},{"name":"fr","value":"fr1"},{"name":"datr","value":"datr1"}]}`

	resp, err := parseRegisterResponse(body, "en_US")
	if err != nil {
		t.Fatalf("parseRegisterResponse error: %v", err)
	}
	if resp.AccessToken == "" || resp.AccessToken[:6] != "EAAAAU" {
		t.Fatalf("expected EAAAAU token, got %q", resp.AccessToken)
	}
}
