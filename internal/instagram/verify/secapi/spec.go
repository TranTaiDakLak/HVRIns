// Package secapi cung cấp implementation dùng chung cho Security Feature
// Android API (TutVer 1) — port từ C# FacebookSecurityFeatureAPIAndroid.
//
// Các biến thể S23/S555-S559 chia sẻ ~97% code; phần khác biệt chỉ là vài
// hằng số (doc_id, bloks_versioning_id, meta_zca, theme_params) và switch
// is_push_on. Package này gói toàn bộ logic, mỗi biến thể chỉ cần truyền
// một Spec để cấu hình.
package secapi

// Spec mô tả các giá trị thay đổi giữa các biến thể FB API version.
// Mỗi variant package (s23/s555/...) chỉ cần khởi tạo 1 instance Spec
// rồi gọi NewClient(spec, ...) — toàn bộ logic dùng chung.
type Spec struct {
	// doc_id cho từng endpoint Bloks. AddSubEmail dùng doc_id riêng;
	// SendCode + ConfirmTwoStep + ConfirmSubEmail dùng chung ContactPoint.
	DocIDAddSubEmail     string
	DocIDContactPoint    string
	DocIDConfirmSubEmail string

	// bloks_versioning_id chung cho tất cả contact-point ops trong variant này.
	BloksVerContact string

	// X-Meta-Zca header — base64 blob (s23/s555/s556/s557) hoặc "empty_token" (s558+).
	MetaZcaValue string

	// Snippet JSON cho theme_params trong nt_context. Mỗi variant đặt giá
	// trị cố định ứng với bộ design system FB nó dùng (FDS-only cho s23,
	// XMDS+FDS cho s555+).
	ThemeParamsJSON string

	// is_push_on flag trong nt_context. s23=true, s555+=false.
	IsPushOn bool
}

// ThemeS23 — theme_params dùng riêng cho S23 (FDS-only, BLUEPRINT_TEST values).
const ThemeS23 = `[{"value":["BLUEPRINT_TEST_GUTTER","BLUEPRINT_TEST_ROUNDED_CORNERS_NO_GUTTERS"],"design_system_name":"FDS"}]`

// ThemeXMDS_FDS — theme_params dùng cho S555/S556/S557/S558/S559
// (XMDS three_neutral_gray + FDS DARKER_PRIMARY_DEEMPHASIZED).
const ThemeXMDS_FDS = `[{"value":["three_neutral_gray"],"design_system_name":"XMDS"},{"value":["DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST"],"design_system_name":"FDS"}]`

// ThemeFDSOnly — theme_params observed in S559 May 2026 contact/confirm traffic.
const ThemeFDSOnly = `[{"value":[],"design_system_name":"FDS"}]`

// MetaZcaBase64 — giá trị X-Meta-Zca chuẩn (S23/S555/S556/S557) — base64
// JSON {"android":{"aka":..,"gpia":..}} với errors KEYSTORE_DISABLED + PLAY_INTEGRITY_DISABLED.
const MetaZcaBase64 = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoiIiwiZXJyb3JzIjpbIktFWVNUT1JFX0RJU0FCTEVEX0JZX0NPTkZJRyJdfSwiZ3BpYSI6eyJ0b2tlbiI6IiIsImVycm9ycyI6WyJQTEFZX0lOVEVHUklUWV9ESVNBQkxFRF9CWV9DT05GSUciXX19fQ"

// DocIDAddSubEmailDefault — doc_id cho AddSubEmail, không thay đổi giữa các biến thể.
const DocIDAddSubEmailDefault = "6970150443042883"

// Friendly names — không đổi giữa các biến thể.
const (
	FnAddSubEmail     = "FbBloksActionRootQuery-com.bloks.www.fx.settings.contact_point.add.async"
	FnSendCodeVerify  = "FbBloksActionRootQuery-com.bloks.www.two_step_verification.send_code.async"
	FnConfirmTwoStep  = "FbBloksActionRootQuery-com.bloks.www.two_step_verification.verify_code.async"
	FnConfirmSubEmail = "FbBloksActionRootQuery-com.bloks.www.fx.settings.contact_point.verify.async"
)

// graphURL — endpoint chung (KHÔNG phải b-graph).
const graphURL = "https://graph.facebook.com/graphql"
