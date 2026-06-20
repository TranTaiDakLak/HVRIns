// livedie_test.go — Test CheckLiveDieByPicture với UID die + UID live thật.
//
// Chạy: go test ./internal/facebook/verify/verifybase/ -run TestCheckLiveDie -v
package verifybase

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCheckLiveDie_RealUIDs(t *testing.T) {
	// Live/network test: gọi FB Graph API thật, account state thay đổi → non-deterministic.
	// Chạy thủ công: RUN_LIVE_TESTS=1 go test -run TestCheckLiveDie -v
	if os.Getenv("RUN_LIVE_TESTS") != "1" {
		t.Skip("requires live account/network (FB Graph API); set RUN_LIVE_TESTS=1 để chạy")
	}

	cases := []struct {
		name     string
		uid      string
		expected string
	}{
		{"DieUID_61589805251621", "61589805251621", "Die"},
		{"LiveUID_Zuck", "4", "Live"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			got := CheckLiveDieByPicture(ctx, "", tc.uid)
			if got != tc.expected {
				t.Errorf("UID %s: got %q, expected %q", tc.uid, got, tc.expected)
			} else {
				t.Logf("[OK] UID %s → %s", tc.uid, got)
			}
		})
	}
}
