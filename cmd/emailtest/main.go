package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

type cred struct {
	Email        string
	Pass         string
	RefreshToken string
	ClientID     string
}

var mails = []string{
	"ryanycarlsonavicqje@hotmail.com|xPPg1ILMMX|M.C550_BAY.0.U.-Ck2StgH3Zucu4knuqoEvJx*S0Ah59QvMB5*sLUQ5zdsu3I5v8RomyJZ*ihIU1Y2sZeP6Gf4t31wa67QU*gayKZvOxL3FsRfflVcHYyxxbL5Vwb1doqSvo!T*BBRSjwEaWhLkZ1RB0ethwLMATdfzkJWAv79HyHHQV!ld047R0ClJ6I34s1U2f9qWoBS0CAMHEjOlfC!0okRRco3luNZrsvBtEPDT1xVMAjqENhnIOYVCqaCY1FXzL*2cUgq81lPz*95cQZPfsFwdNodr*FmyBVn4eR6jBYbr2uG2BRBmZA87J0eLBUZh38CTl1bxO*IVMapR4cNTQZmY7sclQkiQCiw67Cn1W7r6TkdhoXOduXDrl6h*okMSEoCzyaj5NJE8rg$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"nicolebgriffinjbmhm@hotmail.com|Ssez25EZUU|M.C524_BAY.0.U.-Cu2S5v5be86U!locfVnERNJALLgvmsN!vhyFj7evEAC72758MrO2BBIPkEGJPuGTzgS9QigrXsdwhg5USssxa0USgLcDtcyoWNIqcqsYPjiHNdee8OSoGYaM!g7ztET7aVhwCN3RayLixU4nRXjYgAMtQXmyTTTV2ZnB2STDnu0VKN7NgzkkQCW6HfdPBXDlqSZZw0VRyKnJDuMW*!oDPIurg3c48ItnwN9k0Ne4zbtPVPyKU3JUAKMW5v7X06S6UP4dEPGU2eNeGF9ETqdg0PZb6buMRkpe6Q7iewODjj48fKMlmIdIVNkftxF6fGuVhyaQlwIES*RKQB0ogecOareddBpCMUtLxOw4plTjY0hIdBOLMLFAHsNumbD7ioT2ug$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"davidnpaynelwuereb@hotmail.com|KARJSNY0ls|M.C554_BAY.0.U.-CjS6NN9Ihi03UiXe1H*HJXKCJnCbdMmHmZstRncpHH5ZXMCLcQg9JEyxdPdOhBBbwxW3ZfuGJFlei5FN3ytTCd98cxmmZUXsQj0L2CmZucGl1DPDYSO54qjQOBQ1MJG3laQpycb!vHTXmLyPGfa!IYYPS56gPRJkMuUwsxNU8sJ0Hf2E8x5YejWcpCOmuvKl2r0r4P!t9jlthln2jx6zo0bVsTOMEzZCNK!gtxokQd97wII1WuSXE4foRMr6qeRWx2Ms*Whh*XDqxE3WjIb173hYrPWAzb8!N64h6ToEP94RTMj3IfPmAxe5yGPcUkphdeEKArRi0AktNIHdDDVs51gyUpnGpM8qnuad7vJODXjcfeC6RI9kesHAoI7Dlucg1g$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"deannayleeschvpy@hotmail.com|ao6JOhGttU|M.C553_BAY.0.U.-Crrc3kKY35xORRXOm2o68ifB5p0GlpP3V1OhXa3snsHlNrUfKqpZV1pcD8aDUxVrN3Cpcwo*VoPVaubsDcevhi6gPQu6rqKpU7AmNzxPZDgUd*W1oCvL*SA8cmYkR7tgEQaXqvzzNsNKZssvO9pjT1cOB2ppYiU9qZtZ8P1dCWB5udMhKoPL!j4CI05Nh5Y1guKL*wotbqlSOnlyTosh54llGGcWqQid!2E0n8CbIDwnf0P9jwOuMtvv4T6s6EjWuTwpYXCBL7IcG!obF5IvNWX1fn*5Q76AM21688xldu9gQOkQOaUwOmutfWgZUY6iWvM18fJ8yyk3MNy9S1nsHC*hI!JHEDzoY*iXJd4FGKRDTibYpZBwdcU0*ljoNeZhZA$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"jamesmphillipsguajp@hotmail.com|LPsQwrmXFA|M.C531_BAY.0.U.-Ck4jXUaaIxXQlgFTnH5GvUzBXKIO*d3xSuSA3masJydf!S1gODuGUroDIKHjiygXoZ4xVnWkEoRErphMyjBiaG6D!GHmP5xPI3rjaSFoyN0LwkuErUqjZxIuUZbjBeDIwSkiPFB2lQNDsMM19jIPb7vFUyNckCySabO6h6GUiv!E*EcLtQ!ozrX3d*mq*Y!B5Oi622ehh50IWmOaaOuk6tldRY0AkDpH!3VLVLNyvv20gAS6taEwdCnArJ8zZhsq5Qio9BCzKpBFiz9vPzD4SfpEcS9EYSfVwzAhW*pZLWrjvu1PyIY7nJnoG7HPAE23tvPx4ezZcQC!nbPFQYxwtTIopitihBkSpznc7mgm6cwXX8US*fyqJjqHusn05jK0Nw$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"jasondfreemanczvrz@hotmail.com|uCVwxWYTIn|M.C516_BAY.0.U.-Cq1mHyovFgtmX11pXEFggdWce56E0opPkFAZs6o16Rtt*kfewIoEgdSH1lPhwiYVnlbbQv91mrPHTWfObXIndZoM5UdYevs2Wh!Te6NEpG5sbHWFE0BFlB9DfoCKeAwLtjYDeAaMJOQJQrCGuK6**GX0HKG9MG!YkLwFpZlrr6Az5P0D6fuUfKjn1VabdzgThpSZlYA8PbYWy3hpmlq4fkeWTfPZP6O87NujIFoBMvu1On3aC4HW1keDoFBQIZth1EP4QLddYyRmgIPWPSRm2uVphCEtrL7GzEr1yzyZVx0DiS43QkywvETqfAVrytnhiB3lbENBP5XML8Gy1oMu4WKvp*sMQmK8m!QXN*eqgdy3ckXlWz4VKBlY3bO!oVMEyw$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"randalloallenjippxd@hotmail.com|uHIN91eb1d|M.C507_BAY.0.U.-CvbEZdwNWH31I04hwbFQJklgpvE4krOZg1zfX2ZiIH3WtlDONX3wPRLrjc!9eXrqmAowkAg4HDQwDQ2qUddYQoWyx6FKMeKbmlaERyOGI7VEz6jeSzzCqgautaGCJF!klKyXLAHACdZ6kXUWKIC*5iZIqQtcczq4Cx1c1!TnnxDa2vuwRd9Bq4dHGJ1JeyhCsJEcH62ySlNuo6QzaK2UfZuNpPG9gmhoKpLCuimsrDTX5RSCjEuqn0BeUud*lJmc2Zl*5KLoakD0W9SneqdRGeyB9tZ6hmQC3VlTgkZkiNwedQvS6jafr63byM9sVjOCCb3gC0vstT3VuFhHEBZQBt!lOqGexI5Uj7o7FSpCQiP7b6wtkEB9MGi7dY5sEP25qA$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"jeffreytcoxfeogld@hotmail.com|a2gCfFysHl|M.C523_BAY.0.U.-Cm!j0qB0*WbHeqSx9m0Ip94TCsrpo6vyvQxfKgp2ZIfLleGkPljhrAfnwKdv5YaZGhuYiKADX0HaGhZOYuQ78PShc6V0DGeM44tZo3!yiNYeLD9i0jX99i!XEVWvgHKGp5fgvzOqwYLM0pz345Ai9kQUR33JmQRsgcyaSZSHXLZnQtNkS5Z2PvdWzjtZWbxpNgTA4WGkGu!pnAsl6qnYkI8nz6Az3j27sIsnWp8Z3TximXk*KuixIURTr1C4jzHHxKKSvqHCYBJzvvyblM*IZpSoJUp4KI0a!IaSHlMnQFtgVbhDgdle2Imtd*fpOrU2BOTzcOBEel6QrqbFCqYtw9hlonztu6RFYrYufnUh*kG*9urXaKyKvIILLea564gC4Q$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"michaelujenningsrajec@hotmail.com|SJKJmqKm3O|M.C542_BAY.0.U.-ClxkKbMBd95UKXnVjWxAXbkIwcG5BFkIwgRK6G*5OX3vFuLHB2xkzWDpnkCOr6TwuSARlhK7wg*5yvVBVo8hcb3EYLkPU5bWZMpzQLdaSi*5S9xTovRr!iJycwwQQpKBGXAX!L9792j5LK4oRmuKYy1G4RLmNOVxN*2uUuXfHkVOeqjX1xNqPyAGEZos26WjhkJQPXnVcODll8PiqrazyeQZEBDviYLOHk2IkHh9jwS3!mHisXlgJkOCTnx7QyCFvYXt29KXHk8HtbEVAu*AQhzjWaPDgt4Y4fULpYIMuVjAZ7xYuQT573y66zPSAFP2ulSdmB6AhSKVaeK341J7*xQbBy3bT0Hit8PxkTKdB3leTAsya5SerBqQyiha4danTA$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
	"laurasrobertsqfrfrck@hotmail.com|81jt5gFagB|M.C528_BAY.0.U.-CilTjqMSaLolchaA1db0FKdnyP9ait1rW8he4HS13Ry1MBnbhKTHGSCPUcanLQ8iCH8qnamKVfzhwDGoFiTOCEr7PbTsVFRVqL**jezIXzMjkKUuzMkf!Ka!7HTJ4l4sMEJNBpvmwIODl1dgSyhbDsabnF!6mHWHVY65PDslIE1tZxN68Nxq8O!rXFQhuAXq3rjzNl7WqUajS1h1IoF08oGEXz7CA9SMQtGMMHMfAArWOL4Y6V80BVLlp9AeqZF5ScLvcoF*a1hUYp31t5ng8h6MX7zp2oAYQJuaWZPmi37jnf!UAKR32E0BL6fUh8wFpRjqypgu0zB*28rRGST7tKYtn2AysoR3oLrsvQntt*hUBT6Z45DBl*9fWFk!hVaJPA$$|9e5f94bc-e8a4-4e73-b8be-63364c29d753",
}

func parseCred(line string) cred {
	parts := strings.Split(line, "|")
	if len(parts) < 4 {
		return cred{}
	}
	return cred{Email: parts[0], Pass: parts[1], RefreshToken: parts[2], ClientID: parts[3]}
}

type dvfbResp struct {
	Code     string `json:"code"`
	Messages []struct {
		Code    string `json:"code"`
		From    string `json:"from"`
		Message string `json:"message"`
	} `json:"messages"`
}

func fetchOTP(ctx context.Context, c cred) (string, time.Duration, error) {
	payload := map[string]string{
		"email":         c.Email,
		"pass":          c.Pass,
		"refresh_token": c.RefreshToken,
		"client_id":     c.ClientID,
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST",
		"https://tools.dongvanfb.net/api/get_messages_oauth2",
		bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/124.0.0.0")

	t0 := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(t0)
	if err != nil {
		return "", elapsed, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	var r dvfbResp
	json.Unmarshal(body, &r)

	if r.Code != "" {
		return r.Code, elapsed, nil
	}
	for _, m := range r.Messages {
		if m.Code != "" {
			return m.Code, elapsed, nil
		}
	}
	return "", elapsed, nil
}

func main() {
	creds := make([]cred, 0, len(mails))
	for _, line := range mails {
		c := parseCred(line)
		if c.Email != "" {
			creds = append(creds, c)
		}
	}

	fmt.Printf("Test %d hotmail accounts đồng thời...\n\n", len(creds))

	var wg sync.WaitGroup
	type result struct {
		email   string
		code    string
		elapsed time.Duration
		err     error
	}
	results := make([]result, len(creds))

	start := time.Now()
	for i, c := range creds {
		wg.Add(1)
		go func(idx int, cr cred) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			code, elapsed, err := fetchOTP(ctx, cr)
			results[idx] = result{email: cr.Email, code: code, elapsed: elapsed, err: err}
		}(i, c)
	}
	wg.Wait()
	total := time.Since(start)

	// In kết quả
	var sumMs int64
	var errCount, emptyCount, codeCount int
	for i, r := range results {
		status := "empty"
		if r.err != nil {
			status = "ERR: " + r.err.Error()
			errCount++
		} else if r.code != "" {
			status = "code=" + r.code
			codeCount++
		} else {
			emptyCount++
		}
		fmt.Printf("[%2d] %-42s %6dms  %s\n", i+1, r.email, r.elapsed.Milliseconds(), status)
		sumMs += r.elapsed.Milliseconds()
	}

	fmt.Printf("\n--- Tổng kết ---\n")
	fmt.Printf("Tổng thời gian (song song): %dms\n", total.Milliseconds())
	fmt.Printf("Trung bình mỗi call:        %dms\n", sumMs/int64(len(results)))
	fmt.Printf("Có code: %d | Inbox rỗng: %d | Lỗi: %d\n", codeCount, emptyCount, errCount)
}
