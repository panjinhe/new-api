package common

import "strings"

var disposableEmailDomains = map[string]struct{}{
	"psovv.com":  {},
	"xghff.com":  {},
	"oqqaj.com":  {},
	"wyz12.asia": {},
}

func NormalizeEmailDomain(email string) (string, bool) {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", false
	}
	domain := normalizeDomain(email[at+1:])
	if domain == "" {
		return "", false
	}
	return domain, true
}

func IsDisposableEmail(email string) bool {
	domain, ok := NormalizeEmailDomain(email)
	return ok && IsDisposableEmailDomain(domain)
}

func IsDisposableEmailDomain(domain string) bool {
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}
	if _, ok := disposableEmailDomains[domain]; ok {
		return true
	}
	for blocked := range disposableEmailDomains {
		if strings.HasSuffix(domain, "."+blocked) {
			return true
		}
	}
	return false
}

func normalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}
