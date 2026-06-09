package main

import (
	"fmt"
	"regexp"
)

func main() {
	sample := `for (;;);{"success_response":"{\"error_code\":null,\"uid\":61573350035998}","something":"else"}`
	saveCred := `(bk.action.caa.login.SaveCredential, (bk.action.core.GetArg, 0), \"61573350035998\", \"new_to_family_fb_default\"`

	test := func(label, pattern, text string) {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Printf("[%s] COMPILE ERROR: %v\n", label, err)
			return
		}
		m := re.FindStringSubmatch(text)
		if len(m) > 1 {
			fmt.Printf("[%s] OK → %q\n", label, m[1])
		} else {
			fmt.Printf("[%s] NO MATCH\n", label)
		}
	}

	test("P3 uid", `\\"uid\\":(\d+)`, sample)
	test("P4 old SaveCred[^,]", `SaveCredential[^,]+,\s*\\"(\d{10,18})\\"`, saveCred)
	test("P4 new SaveCred.*?", `SaveCredential.*?\\"(\d{10,18})\\"`, saveCred)
}
