// Package web — DEPRECATED: Cookie scrape token fetch đã removed.
//
// Trước đây file này chứa FetchTokenFromCookie (port WeBM GetTokenEAAG/B).
// Đã verify FAIL với FB redirect /confirmemail.php cho account NVR.
// Replaced bằng FetchAndroidTokenLegacy ở android_token_legacy.go — REST `/auth/login`
// stable, không phụ thuộc cookie state.
package web
