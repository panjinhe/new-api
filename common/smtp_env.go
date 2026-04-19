package common

import (
	"os"
	"strconv"
	"strings"
)

var smtpOptionEnvNames = map[string]string{
	"SMTPServer":         "SMTP_SERVER",
	"SMTPPort":           "SMTP_PORT",
	"SMTPSSLEnabled":     "SMTP_SSL_ENABLED",
	"SMTPForceAuthLogin": "SMTP_FORCE_AUTH_LOGIN",
	"SMTPAccount":        "SMTP_ACCOUNT",
	"SMTPFrom":           "SMTP_FROM",
	"SMTPToken":          "SMTP_TOKEN",
}

var smtpOptionEnvOverrides = map[string]bool{}

func LoadSMTPEnvOptions() {
	smtpOptionEnvOverrides = make(map[string]bool, len(smtpOptionEnvNames))

	loadSMTPStringEnv("SMTPServer", &SMTPServer)
	loadSMTPIntEnv("SMTPPort", &SMTPPort)
	loadSMTPBoolEnv("SMTPSSLEnabled", &SMTPSSLEnabled)
	loadSMTPBoolEnv("SMTPForceAuthLogin", &SMTPForceAuthLogin)
	loadSMTPStringEnv("SMTPAccount", &SMTPAccount)
	loadSMTPStringEnv("SMTPFrom", &SMTPFrom)
	loadSMTPStringEnv("SMTPToken", &SMTPToken)
}

func IsSMTPEnvOverride(key string) bool {
	return smtpOptionEnvOverrides[key]
}

func GetSMTPOptionValue(key string) string {
	switch key {
	case "SMTPServer":
		return SMTPServer
	case "SMTPPort":
		return strconv.Itoa(SMTPPort)
	case "SMTPSSLEnabled":
		return strconv.FormatBool(SMTPSSLEnabled)
	case "SMTPForceAuthLogin":
		return strconv.FormatBool(SMTPForceAuthLogin)
	case "SMTPAccount":
		return SMTPAccount
	case "SMTPFrom":
		return SMTPFrom
	case "SMTPToken":
		return SMTPToken
	default:
		return ""
	}
}

func loadSMTPStringEnv(key string, target *string) {
	value, ok := os.LookupEnv(smtpOptionEnvNames[key])
	if !ok {
		return
	}
	*target = strings.TrimSpace(value)
	smtpOptionEnvOverrides[key] = true
}

func loadSMTPIntEnv(key string, target *int) {
	value, ok := os.LookupEnv(smtpOptionEnvNames[key])
	if !ok {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		SysLog("invalid SMTP env " + smtpOptionEnvNames[key] + ", ignore override")
		return
	}
	*target = parsed
	smtpOptionEnvOverrides[key] = true
}

func loadSMTPBoolEnv(key string, target *bool) {
	value, ok := os.LookupEnv(smtpOptionEnvNames[key])
	if !ok {
		return
	}
	*target = strings.EqualFold(strings.TrimSpace(value), "true")
	smtpOptionEnvOverrides[key] = true
}
