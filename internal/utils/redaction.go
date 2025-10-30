// internal/utils/redaction.go
package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	RedactionMask = "****"
	maxPreviewLen = 16
)

var sensitiveKeySubstr = []string{
	"token", "secret", "password", "passwd", "apikey", "api_key", "authorization",
	"auth", "credential", "private", "key", "cert", "certificate", "pem",
	"jwt", "bearer", "value",
}

func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, sub := range sensitiveKeySubstr {
		if strings.Contains(k, sub) {
			return true
		}
	}
	return false
}

func RedactString(_ string) string { return RedactionMask }

func TruncatePreview(s string) string {
	if len(s) <= maxPreviewLen {
		return s
	}
	head := s[:maxPreviewLen/2]
	tail := s[len(s)-maxPreviewLen/2:]
	return head + "…" + tail
}

func SafeValue(key string, v any) any {
	if IsSensitiveKey(key) {
		return RedactionMask
	}
	switch x := v.(type) {
	case string:
		return TruncatePreview(x)
	default:
		return v
	}
}

func SafeFields(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch t := v.(type) {
		case map[string]any:
			out[k] = SafeFields(t)
		case map[string]string:
			tmp := make(map[string]any, len(t))
			for kk, vv := range t {
				tmp[kk] = SafeValue(kk, vv)
			}
			out[k] = tmp
		default:
			out[k] = SafeValue(k, v)
		}
	}
	return out
}

func RedactHTTPHeaders(h http.Header) http.Header {
	safe := http.Header{}
	for k, vals := range h {
		if IsSensitiveKey(k) || strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Cookie") {
			safe[k] = []string{RedactionMask}
			continue
		}
		out := make([]string, 0, len(vals))
		for _, v := range vals {
			out = append(out, TruncatePreview(v))
		}
		safe[k] = out
	}
	return safe
}

func RedactURLQuery(raw string, extraSensitiveKeys ...string) string {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return raw
	}
	q := u.Query()
	for key := range q {
		if IsSensitiveKey(key) || containsFold(extraSensitiveKeys, key) {
			q.Set(key, RedactionMask)
		} else {
			q.Set(key, TruncatePreview(q.Get(key)))
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func containsFold(list []string, s string) bool {
	for _, x := range list {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

func RedactJSONBytes(b []byte) []byte {
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return b
	}
	redacted := redactJSONNode(m)
	out, err := json.Marshal(redacted)
	if err != nil {
		return b
	}
	return out
}

func redactJSONNode(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			if IsSensitiveKey(k) {
				out[k] = RedactionMask
				continue
			}
			out[k] = redactJSONNode(vv)
		}
		return out
	case []any:
		for i := range t {
			t[i] = redactJSONNode(t[i])
		}
		return t
	case string:
		return TruncatePreview(t)
	default:
		return t
	}
}

var pemBlockRx = regexp.MustCompile(`-----BEGIN [^-]+-----[\s\S]+?-----END [^-]+-----`)

func RedactPEM(s string) string {
	return pemBlockRx.ReplaceAllString(s, RedactionMask)
}

func RedactBytesChain(body []byte) []byte {
	body = RedactJSONBytes(body)
	body = []byte(RedactPEM(string(body)))
	lower := strings.ToLower(string(body))
	if strings.Contains(lower, "authorization:") || strings.Contains(lower, "bearer ") {
		return []byte(RedactionMask)
	}
	return body
}

func SafeKVString(fields map[string]any) string {
	var buf bytes.Buffer
	first := true
	for k, v := range SafeFields(fields) {
		if !first {
			buf.WriteString(" ")
		}
		first = false
		buf.WriteString(k)
		buf.WriteString("=")
		switch vv := v.(type) {
		case string:
			buf.WriteString(vv)
		default:
			enc, _ := json.Marshal(v)
			buf.Write(enc)
		}
	}
	return buf.String()
}
