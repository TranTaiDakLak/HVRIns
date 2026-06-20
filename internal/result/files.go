package result

import "strings"

// Result file names written into the session output directory.
const (
	FileSuccessReg                  = "SuccessReg.txt"
	FileSuccessVerify               = "SuccessVerify.txt"
	FileSuccessVerifyNo2FA          = "SuccessVerify_No2FA.txt"
	FileDieAfterVerify              = "DieAfterVerify.txt"
	FileUnknownErrorCheckLiveDieApi = "UnknownError_CheckLiveDie.txt"
	FileCheckpoint                  = "Checkpoint.txt"
	FileBlocked                     = "Blocked.txt"
	FileUnknownBlockType            = "UnknownBlock.txt"
	FileSuccessNVREmail             = "SuccessNVR_Email.txt"
	FileSuccessNVRPhone             = "SuccessNVR_Phone.txt"
)

// SuccessRegNVRUGFile returns the per-platform filename used to persist user-agents
// from successful NVR registrations (one file per platform in the session folder).
func SuccessRegNVRUGFile(regInstance string) string {
	if regInstance == "" {
		return "SuccessRegNVR_UA.txt"
	}
	return "SuccessRegNVR_" + regInstance + "_UA.txt"
}

// SuccessVerifyUGFile returns the per-platform filename used to persist user-agents
// from successful verifications.
func SuccessVerifyUGFile(verifyInstance string) string {
	if verifyInstance == "" {
		return "SuccessVerify_UA.txt"
	}
	return "SuccessVerify_" + verifyInstance + "_UA.txt"
}

// ParseEmailMetaFromLine extracts the "MM:..." email-meta token from a result-file line.
// Lines that are purely metadata start with "MM:"; lines that carry meta as a suffix field
// contain "|MM:".  Returns the "MM:..." string, or "" if the line is not email-meta.
func ParseEmailMetaFromLine(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "MM:") {
		return line
	}
	// Meta appended as a trailing pipe-delimited field.
	if idx := strings.Index(line, "|MM:"); idx >= 0 {
		return line[idx+1:]
	}
	return ""
}
