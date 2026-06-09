// Package facebook — URL and app version constants
// Mapping từ C#: FbAppVersionsConstants + UrlSingleton
package instagram

// Facebook API endpoint base URLs
const (
	BaseURLMobile  = "https://m.facebook.com"
	BaseURLBGraph  = "https://b-graph.facebook.com"
	BaseURLGraph   = "https://graph.facebook.com"
	BaseURLAPI     = "https://api.facebook.com"
	BaseURLWWW     = "https://www.facebook.com"

	// Wbloks endpoint for Bloks CAA (Create Account API)
	EndpointWbloks = "https://m.facebook.com/api/graphql/"

	// Mobile login endpoint
	EndpointLogin  = "https://m.facebook.com/login/"

	// Registration page
	EndpointRegister = "https://m.facebook.com/r.php"
)

// Facebook Android app version constants
// Mapping từ C#: FbAppVersionsConstants
const (
	AndroidAppVersion    = "514.0.0.45.108"
	AndroidBuildVersion  = "514"
	AndroidLocale        = "en_US"
	AndroidClientCountry = "us"
	AndroidAPIVersion    = "v21.0"
)

// Facebook Graph API version
const GraphAPIVersion = "v21.0"

// Android Register API constants
// Mapping từ C#: FacebookApiFormDataBuilder + FbAppVersionsConstants
const (
	// OAuth app token cho Android API (không cần access_token user)
	AndroidOAuthToken = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"

	// GraphQL friendly_name cho register
	AndroidRegFriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"

	// client_doc_id cho register (ChangeAndConfirmContactpointMobiledocid_v3)
	AndroidRegDocID = "1199408042526631289603660492"

	// Bloks versioning ID (cập nhật khi FB thay đổi)
	AndroidBloksVersionID = "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999"

	// X-Meta-Zca header value (base64 encoded device attestation config)
	AndroidMetaZCA = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoiIiwiZXJyb3JzIjpbIktFWVNUT1JFX0RJU0FCTEVEX0JZX0NPTkZJRyJdfSwiZ3BpYSI6eyJ0b2tlbiI6IiIsImVycm9ycyI6WyJQTEFZX0lOVEVHUklUWV9ESVNBQkxFRF9CWV9DT05GSUciXX19fQ"

	// Batch call friendly_name cho fetch eligibility_hash (X-Zero-EH)
	AndroidBatchFriendlyName = "fetchLoginData-batch"

	// FB app version mới hơn cho register (từ captured API S23)
	AndroidRegAppVersion = "554.0.0.57.70"
	AndroidRegBuildNum   = "918990560"
)
