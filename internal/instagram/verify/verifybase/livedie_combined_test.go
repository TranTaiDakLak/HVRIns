// livedie_combined_test.go — Test combined check (token + picture) với 4 UID thật từ user.
//
// Chạy: go test ./internal/facebook/verify/verifybase/ -run TestCheckLiveDieCombined -v
package verifybase

import (
	"context"
	"testing"
	"time"
)

func TestCheckLiveDieCombined_RealAccounts(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: cần network")
	}

	cases := []struct {
		name     string
		uid      string
		token    string
		expected string
	}{
		{
			"61589975644217 (Live thật)",
			"61589975644217",
			"EAAAAUaZA8jlABRbg5PAg7QUtCkb6VvAjhqZBR4UABgLd0ZBbC9HcBvDiv6rOZB5DfjJk9q0dAUjBnE7LZAS7srzW4VO0MMGbBAieh7L5ZBg0p4q0torEouD470lcDsO5ZC2yTObnLdSVw4Tih3O1gKTBp3ZAzkqKcy8sNYj7ytct9FJqeoXZCFNDixaf4mlnDwYBSsFoMNgZDZD",
			"Live",
		},
		{
			"61590158936298 (Checkpoint)",
			"61590158936298",
			"EAAAAUaZA8jlABReyYcC7KZBlHGAOThJ4IbZBWVbWumW1tznAjmhTnLEHfOHmfipMdZBnr9EDnldLMsG7sZCkZBLCMgAgtNfC5vHZCKZAQPtCCsXJD1zM6SYT7ih3QhlTFO5GpqZALUZAcRsUDuy4aI9M2UZB2KwcyROZCtg9LGzFzxYW8JizGDYQb3uZC2APG37mO8uKT3LtpmgZDZD",
			"Die",
		},
		{
			"61588798400552 (Checkpoint)",
			"61588798400552",
			"EAAAAUaZA8jlABRS1rmIdJubps6DVGLOIvCKmyrCZAnitWu8uHbHw9kVg29aBTEYoXe5FRm8hHVaRLmosLncqKkPn3PgGzmZCxbsPfy1uezXZAcTZAfwgbXS2jzG2E7s5DNJkfbNdZAQGdZCuGPhlf3soxF0ddPLPC1hpTVgAeRyaZBwUZBYfyM6cyojzyPPKKpVIiukqF9sHjNgZDZD",
			"Die",
		},
		{
			"61589677637323 (Checkpoint)",
			"61589677637323",
			"EAAAAUaZA8jlABRdLpUZBaEzVhOYaUhassrx4U2LliaGmPoQQZByJXYRzMNw2YZBcDDnHOZA8cPALfUcGhCM6Q2afWilsefpDAtsIG0jTJK4Cgt8jQqBcs5yMGSC3UEWFcGCeFqXkYjF54J35Wjuc8WgLZBKQ9TmeDh83xGbr1j3ABpu1X2ZA2KNbB07k0ZCBq1NZANfChEgZDZD",
			"Die",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			got := CheckLiveDieCombined(ctx, "", tc.uid, tc.token)
			if got != tc.expected {
				t.Errorf("UID %s: got %q, expected %q", tc.uid, got, tc.expected)
			} else {
				t.Logf("[OK] %s → %s", tc.name, got)
			}
		})
	}
}
