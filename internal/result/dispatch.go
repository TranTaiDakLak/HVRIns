package result

// DetailEntry describes a single file-write produced by a Dispatch* function.
type DetailEntry struct {
	File    string // filename within the session root
	Content string // line content to write
	Upsert  bool   // true → UpsertUID; false → Append
}

// DispatchRegDetails returns additional detail-file writes derived from
// sub-status information embedded in the register result message.
// Returns nil for messages that do not require extra dispatch files.
func DispatchRegDetails(status, message, line string) []DetailEntry {
	return nil
}

// DispatchVerifyDetails returns additional detail-file writes derived from
// sub-status information embedded in the verify result message.
// Returns nil for messages that do not require extra dispatch files.
func DispatchVerifyDetails(status, message, line string) []DetailEntry {
	return nil
}
