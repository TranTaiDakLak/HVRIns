// Package web — Facebook Web platform (m.facebook.com endpoints)
// Mapping từ C#: API/FBWebApi/ + Automation/FacebookRegisterAutomation.cs + VerifyAccountAPIAutomation.cs
package web

import "HVRIns/internal/instagram"

// Type aliases — bridge migration từ package register/verify sang facebook/web.
// Cho phép các file trong web/ dùng RegInput, RegSession, RegResult, StatusCallback
// trực tiếp mà không cần thêm prefix facebook. vào mọi type reference.
type (
	RegInput       = instagram.RegInput
	RegSession     = instagram.RegSession
	RegResult      = instagram.RegResult
	StatusCallback = func(string)

	VerifyConfig = instagram.VerifyConfig
	VerifyResult = instagram.VerifyResult
)
