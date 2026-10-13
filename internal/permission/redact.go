package permission

import (
	"regexp"
	"strings"
)

const mask = "***"

// minSecretLen 显式传入的密钥短于此长度时不做原值替换，避免误伤普通文本。
const minSecretLen = 6

var (
	reSK     = regexp.MustCompile(`sk-[A-Za-z0-9_\-]{8,}`)
	reBearer = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._\-~+/=]{8,}`)
	reEnv    = regexp.MustCompile(`(?i)([A-Z0-9_]*(?:API_KEY|TOKEN|SECRET)\s*[=:]\s*["']?)([^\s"']+)`)
)

// Redact 把 secrets 原值、sk-…、Bearer xxx、*_API_KEY= / *_TOKEN= / *_SECRET= 的值替换成 ***。
func Redact(s string, secrets ...string) string {
	if s == "" {
		return s
	}
	for _, k := range secrets {
		if k = strings.TrimSpace(k); len(k) >= minSecretLen {
			s = strings.ReplaceAll(s, k, mask)
		}
	}
	s = reSK.ReplaceAllString(s, mask)
	s = reBearer.ReplaceAllString(s, "${1}"+mask)
	s = reEnv.ReplaceAllStringFunc(s, func(m string) string {
		sub := reEnv.FindStringSubmatch(m)
		if sub[2] == mask {
			return m
		}
		return sub[1] + mask
	})
	return s
}
