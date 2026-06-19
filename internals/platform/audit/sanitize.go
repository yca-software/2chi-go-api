package audit

import (
	"encoding/json"
	"strings"

	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
)

func ToPublicAuditLog(log *models.AuditLog) models.AuditLogPublic {
	var data json.RawMessage
	if log.Data != nil {
		data = SanitizeAuditDataJSON(*log.Data)
	} else {
		data = json.RawMessage(`null`)
	}

	return models.AuditLogPublic{
		ID:             log.ID,
		CreatedAt:      log.CreatedAt,
		OrganizationID: log.OrganizationID,
		ActorID:        log.ActorID,
		ActorInfo:      log.ActorInfo,
		Action:         log.Action,
		ResourceType:   log.ResourceType,
		ResourceID:     log.ResourceID,
		ResourceName:   log.ResourceName,
		Data:           data,
	}
}

func SanitizeAuditDataJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`null`)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return json.RawMessage(`{}`)
	}
	sanitized := SanitizeAuditValue(v, "")
	out, err := json.Marshal(sanitized)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

func SanitizeAuditValue(v any, keyHint string) any {
	switch x := v.(type) {
	case map[string]any:
		return SanitizeAuditMap(x)
	case []any:
		out := make([]any, len(x))
		for i, el := range x {
			out[i] = SanitizeAuditValue(el, keyHint)
		}
		return out
	case string:
		if IsSensitiveAuditKey(keyHint) {
			return "[redacted]"
		}
		return x
	default:
		if IsSensitiveAuditKey(keyHint) {
			return "[redacted]"
		}
		return x
	}
}

func SanitizeAuditMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if IsSensitiveAuditKey(k) {
			out[k] = "[redacted]"
			continue
		}
		switch child := v.(type) {
		case map[string]any:
			out[k] = SanitizeAuditMap(child)
		default:
			out[k] = SanitizeAuditValue(v, k)
		}
	}
	return out
}

func IsSensitiveAuditKey(k string) bool {
	kl := strings.ToLower(k)
	switch kl {
	case "password", "currentpassword", "newpassword", "token", "secrettoken", "accesstoken",
		"refreshtoken", "authorization", "cookie", "apisecret", "clientsecret", "webhooksecret",
		"privatekey", "keyhash", "key_hash", "paddlecustomerid":
		return true
	default:
		return strings.Contains(kl, "password") ||
			strings.Contains(kl, "secret") ||
			strings.Contains(kl, "token") ||
			kl == "api_key" ||
			strings.HasSuffix(kl, "apikey")
	}
}
