package model

// DefaultAppSettings trả về AppSettings mặc định với 1 profile "default"
func DefaultAppSettings() AppSettings {
	return AppSettings{
		Version:         CurrentVersion,
		ActiveProfileID: "default",
		Global:          DefaultGlobalSettings(),
		Profiles:        []Profile{DefaultProfile("default", "Default")},
		SecretsRef: SecretReference{
			Provider: "",
			KeyRefs:  map[string]string{},
		},
	}
}

// DefaultGlobalSettings trả về global settings mặc định
func DefaultGlobalSettings() GlobalSettings {
	return GlobalSettings{
		LoginPlatform:  "facebook",
		LoginMethod:    0,
		SaveRunColumn:  false,
		BackupDB:       false,
		CloseAfterDone: false,
	}
}

// DefaultProfile tạo profile với default values theo id và name cho trước
func DefaultProfile(id, name string) Profile {
	return Profile{
		ID:   id,
		Name: name,
		Runtime: RuntimeSettings{
			ThreadRequest:    20,
			DelayRequest:     500,
			DelayThread:      0,
			ApiCheckIp:       0,
			ThreadCheckInfo:  10,
			DelayChangeIp:    3,
			CheckIpBeforeRun: false,
		},
		Account: AccountSettings{
			Source:     "folder",
			FolderPath: "",
			CloneHV: CloneHVCredentials{
				Enabled:   false,
				Username:  "",
				Password:  "",
				ProductID: "",
				Amount:    1,
			},
		},
		Proxy: ProxySettings{
			Provider:  "none",
			ProxyList: "",
			ProxyType: "http",
			Providers: map[string]ProxyProviderCfg{
				"xproxy": {Type: "http", RunType: "shared"},
			},
		},
		Verify: VerifySettings{
			Enabled:           false,
			CheckLiveDie:      true, // default ON — bật lớp post-confirm verify cho tất cả platform (2026-05-26)
			TimeDelayCheck:    5,
			TimeDelaySendCode: 30,
			SendAgainCode:     false,
		},
		Register: RegisterSettings{
			Enabled:    false,
			Type:       "normal",
			CookieList: "",
			OutputPath: "",
		},
		Mail: MailSettings{
			Provider: "@i2b.vn",
			MailList: "",
			Providers: map[string]MailProviderCfg{
				"zeusx":  {},
				"dvfb":   {AccountType: "1"},
				"store1s": {},
				"mail30s": {},
			},
		},
		Captcha: CaptchaSettings{
			Provider: "2captcha",
			Keys: map[string]string{
				"2captcha":   "",
				"omocaptcha": "",
				"ezcaptcha":  "",
				"capsolver":  "",
			},
		},
		Output: OutputSettings{
			VerifyPath:   "",
			RegisterPath: "",
		},
		Device: DeviceSettings{
			UAList:      "",
			UaPools:     map[string]string{"iphone": "", "android": "", "chrome": ""},
			UaPoolKey:   "iphone",
			UaPoolFiles: map[string]string{},
		},
	}
}
