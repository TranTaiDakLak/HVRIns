package igcore

import (
	"context"
	"fmt"
	"testing"
)

func TestCheckLiveByCheckerCookie(t *testing.T) {
	cookie := "datr=Nyc9acU231VKavzhv9rCjn8y; ig_did=7EF23952-5BED-4BD1-B85E-0203564F7A17; ps_l=1; ps_n=1; mid=aid9YAALAAH65IYTGOZpDIAoP2zi; csrftoken=U0GHy0X2SjjEWfCeCI2ncVWO1n16fgza; ds_user_id=27066622847; sessionid=27066622847%3Au2zWmFqydZeTyX%3A8%3AAYh6eoPRXCmIXbU2YSbpEoHgKNy8k2Ix6VGIEbyn2Q; wd=887x911"

	users := []string{
		"solar.1000194", "hamster.1000086", "hamster.1000226", "cedar.1000020",
		"lynx.1000139", "pixel.1000123", "summer.1000242", "tiger.1000224",
		"ember.1000218", "maple.1000187", "fox.1000226", "cedar.1000049",
		"fox.1000138", "fox.1000162", "ocean.1000131", "lunar.1000187",
		"hamster.1000175", "winter.1000007", "dolphin.1000220", "otter.1000249",
		"ember.1000191", "winter.1000153", "cedar.1000216", "wolf.1000080",
		"willow.1000004", "hamster.1000039", "panda.1000254", "wolf.1000169",
	}
	ctx := context.Background()

	live, die, unknown := 0, 0, 0
	for _, u := range users {
		result := CheckLiveByCheckerCookie(ctx, cookie, u, "")
		fmt.Printf("%-22s → %s\n", u, result)
		switch result {
		case "live":    live++
		case "die":     die++
		default:        unknown++
		}
	}
	fmt.Printf("\n✅ live=%d  ❌ die=%d  ❓ unknown=%d\n", live, die, unknown)
}
