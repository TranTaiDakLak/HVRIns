// livedie_test.go — Test CheckLiveDieByPicture với UID die + UID live thật.
//
// Chạy: go test ./internal/facebook/verify/verifybase/ -run TestCheckLiveDie -v
package verifybase

import (
	"context"
	"testing"
	"time"
)

func TestCheckLiveDie_RealUIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: cần network")
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
