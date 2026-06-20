// dispatch.go — map status message/code → file phân loại chi tiết.
//
// Port từ C# FMain.cs flow:
//   result.StatusCode != Success → SaveFile($"VerifyFailed_{status}.txt", data)
//   AddEmailResult.FbStatusCode  → SaveFile($"LoginFbFailed_{status}.txt", data)
//   ConfirmEmailResult           → SaveFile($"Cfem_LoginFbFailed_{status}.txt", data)
//   etc.
//
// Go model hiện chỉ có AccountResult{Status: Live/Die/Unknown, Message: string},
// nên hàm ở đây detect sub-status từ Message để dispatch vào đúng file.
package result

import (
	"strings"
)

// DetailDispatch mô tả 1 dòng kết quả cần ghi vào file phân loại.
// File: tên file (có thể dynamic qua helpers VerifyFailedFile, LoginFbFailedFile...)
// Content: nội dung ghi (thường là line đã format sẵn).
// Upsert: nếu true dùng UpsertByUID thay vì AppendLine (chỉ DieAfterVerify).
type DetailDispatch struct {
	File    string
	Content string
	Upsert  bool
}

// DispatchVerifyDetails sinh danh sách DetailDispatch từ verify outcome.
//
// status:  "live" | "die" | "unknown"  (case-insensitive).
// message: Message field từ AccountResult — dùng để detect sub-status.
// line:    data đã format (result.FormatVerify output).
//
// Trả về list các file cần ghi — thường 1-2 file (main + UA).
// Caller iterate và gọi Writer.Append/UpsertUID theo Upsert flag.
func DispatchVerifyDetails(status, message, line string) []DetailDispatch {
	s := strings.ToLower(strings.TrimSpace(status))
	low := strings.ToLower(message)

	// Detect checkpoint specifically (C# VeriAccountAutoStatusCode.Checkpoint)
	isCheckpoint := strings.Contains(low, "checkpoint")

	switch s {
	case "live":
		// Success flow — chỉ append vào SuccessVerify (caller đã handle No2FA)
		return nil
	case "die":
		// saveVerifyOutcome đã ghi FileDieAfterVerify — chỉ append detail files.
		var out []DetailDispatch
		if isCheckpoint {
			out = append(out, DetailDispatch{
				File: VerifyFailedFile("Checkpoint"), Content: line,
			})
		} else if code := detectVerifyFailCode(low); code != "" {
			out = append(out, DetailDispatch{
				File: VerifyFailedFile(code), Content: line,
			})
		}
		return out
	default: // unknown / error
		// saveVerifyOutcome đã ghi FileUnknownErrorCheckLiveDieApi — chỉ trả về detail files.
		var out []DetailDispatch
		// Detect specific API errors
		if strings.Contains(low, "cant get code") ||
			strings.Contains(low, "can't get code") ||
			strings.Contains(low, "cantgetcode") {
			// Rent mail (TQ) vs buy mail: heuristic
			file := FileBuyMailCantGetCode
			if strings.Contains(low, "china") || strings.Contains(low, "rent") {
				file = FileChinaMailCantGetCode
			}
			out = append(out, DetailDispatch{File: file, Content: line})
		}
		if strings.Contains(low, "login fb failed") || strings.Contains(low, "addmailfailed") {
			code := detectVerifyFailCode(low)
			if code == "" {
				code = "UnknownError"
			}
			out = append(out, DetailDispatch{
				File: LoginFbFailedFile(code), Content: line,
			})
		} else if strings.Contains(low, "confirm") && strings.Contains(low, "fail") {
			code := detectVerifyFailCode(low)
			if code == "" {
				code = "UnknownError"
			}
			out = append(out, DetailDispatch{
				File: CfemLoginFbFailedFile(code), Content: line,
			})
		}
		if strings.Contains(low, "login gmail") || strings.Contains(low, "logingmail") {
			code := detectLoginGmailCode(low)
			if code == "" {
				code = "UnknownError"
			}
			out = append(out, DetailDispatch{
				File: LoginGmailFile(code), Content: line,
			})
		}
		return out
	}
}

// DispatchRegDetails sinh detail files cho register outcome.
//
// status:  "live"/"success" | "checkpoint" | "blocked" | "unknown".
// message: Message từ RegResult.
// line:    data đã format (FormatReg output).
func DispatchRegDetails(status, message, line string) []DetailDispatch {
	s := strings.ToLower(strings.TrimSpace(status))
	low := strings.ToLower(message)

	switch s {
	case "live", "success":
		// Không có detail file extra — Writer đã append SuccessReg.txt ở saveRegOutcome
		return nil
	case "checkpoint":
		return nil // Writer đã append Checkpoint.txt
	case "blocked", "block":
		return nil // Writer đã append Blocked.txt
	default:
		// Unknown — thêm detail nếu detect được
		if strings.Contains(low, "check live") || strings.Contains(low, "checklive") {
			return []DetailDispatch{
				{File: FileSuccessButErrorCheckLive, Content: line},
			}
		}
		return nil
	}
}

// detectVerifyFailCode — map Message về 1 status code ngắn để đặt tên file.
// Port C# VeriAccountAutoStatusCode enum names.
//
// low phải đã lowercased bởi caller.
func detectVerifyFailCode(low string) string {
	switch {
	case strings.Contains(low, "cantgetcode"), strings.Contains(low, "cant get code"):
		return "CantGetCode"
	case strings.Contains(low, "createtempmail"):
		return "CreateTempMailFailed"
	case strings.Contains(low, "rentmail"):
		return "RentMailFailed"
	case strings.Contains(low, "addmail"):
		return "AddMailFailed"
	case strings.Contains(low, "confirmmail"):
		return "ConfirmMailFailed"
	case strings.Contains(low, "logingmail"):
		return "LoginGmailFailed"
	case strings.Contains(low, "chromeerror"):
		return "ChromeError"
	case strings.Contains(low, "checkpoint"):
		return "Checkpoint"
	case strings.Contains(low, "unknowncheck"):
		return "UnknownCheckLiveUid"
	case strings.Contains(low, "success_but"):
		// Các Success_but_*_failed variants
		return "SuccessButFailed"
	}
	return ""
}

// detectLoginGmailCode map LoginGmailStatusCode từ Message.
func detectLoginGmailCode(low string) string {
	switch {
	case strings.Contains(low, "captcha"):
		return "Captcha"
	case strings.Contains(low, "wronguser"), strings.Contains(low, "wrong user"):
		return "WrongUser"
	case strings.Contains(low, "wrongpass"), strings.Contains(low, "wrong pass"):
		return "WrongPass"
	case strings.Contains(low, "timeout"):
		return "Timeout"
	case strings.Contains(low, "verify"):
		return "Verify"
	}
	return ""
}
