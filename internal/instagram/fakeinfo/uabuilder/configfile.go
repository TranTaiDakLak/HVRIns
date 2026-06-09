// configfile.go — ConfigFileUABuilder port C# ConfigFileUserAgentBuilder.cs.
//
// Đọc 1 UA random từ pool prebuilt (Config/UserAgent/Android_UG.txt, iOS_UG.txt, Request_UG.txt).
// Cả AddVirtualSpecs lẫn UseBuildNumFile bị IGNORE — UA đã prebuilt sẵn, không build dynamic.
//
// Source pool được wire qua function adapter ConfigFileUASource (xem builder.go).
// Adapter mặc định nên dùng fakeinfo.RandomUAFromPool(UAKindAndroid|...).
//
// Caller không cần biết bên trong là kind nào — UABuilder interface ẩn đi.
package uabuilder

import "errors"

type ConfigFileUABuilder struct {
	Source ConfigFileUASource
}

func (b *ConfigFileUABuilder) Kind() UABuilderKind { return KindConfigFile }

var errConfigFileEmptyPool = errors.New("uabuilder: config file UA pool empty")

func (b *ConfigFileUABuilder) Build(opts UAOptions) (UABuildResult, error) {
	if b.Source == nil {
		return UABuildResult{}, errors.New("uabuilder: ConfigFileUABuilder.Source not set")
	}
	ua := b.Source()
	if ua == "" {
		return UABuildResult{}, errConfigFileEmptyPool
	}
	locale := opts.Locale
	if locale == "" {
		locale = "en_US"
	}
	return UABuildResult{
		UserAgent: ua,
		Kind:      KindConfigFile,
		Locale:    locale,
	}, nil
}
