package common

import "fmt"

const (
	NAME_NOT_AVAILABLE         = "anonymous"
	ANTITHESIS_SDK_MODULE      = "github.com/antithesishq/antithesis-sdk-go"
	ASSERT_PACKAGE             = "assert"
	INSTRUMENTATION_PACKAGE    = "instrumentation"
	NOTIFIER_MODULE_NAME       = "antithesis.notifier"
	GENERATED_SUFFIX           = "_antithesis_catalog.go"
	INSTRUMENTED_SOURCE_FOLDER = "customer"
	SYMBOLS_FOLDER             = "symbols"
	SYMBOLS_FILE_HASH_PREFIX   = "go"
	SYMBOLS_FILE_SUFFIX        = ".sym.tsv"
	NOTIFIER_FOLDER            = "notifier"
	GENERATED_NOTIFIER_SOURCE  = "notifier.go"
	NOTIFIER_PACKAGE_PREFIX    = "z"
)

func SDKPackageName(packageName string) string {
	return fmt.Sprintf("%s/%s", ANTITHESIS_SDK_MODULE, packageName)
}

func AssertPackageName() string {
	return SDKPackageName(ASSERT_PACKAGE)
}

func InstrumentationPackageName() string {
	return SDKPackageName(INSTRUMENTATION_PACKAGE)
}

// package z4a1b45a05078
func NotifierPackage(filesHash string) string {
	return fmt.Sprintf("%s%s", NOTIFIER_PACKAGE_PREFIX, filesHash)
}

// require antithesis.notifier/z4a1b45a05078
func FullNotifierName(filesHash string) string {
	packageName := NotifierPackage(filesHash)
	return fmt.Sprintf("%s/%s", NOTIFIER_MODULE_NAME, packageName)
}
