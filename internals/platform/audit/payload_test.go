package audit_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yca-software/2chi-go-api/internals/platform/audit"
)

func TestCreatePayload(t *testing.T) {
	raw, err := json.Marshal(audit.CreatePayload(map[string]any{"name": "Acme"}))
	require.NoError(t, err)
	require.JSONEq(t, `{"updated":{"name":"Acme"}}`, string(raw))
}

func TestUpdatePayload(t *testing.T) {
	raw, err := json.Marshal(audit.UpdatePayload(
		map[string]any{"name": "Old"},
		map[string]any{"name": "New"},
	))
	require.NoError(t, err)
	require.JSONEq(t, `{"previous":{"name":"Old"},"updated":{"name":"New"}}`, string(raw))
}

func TestDeletePayload(t *testing.T) {
	raw, err := json.Marshal(audit.DeletePayload(map[string]any{"name": "Gone"}))
	require.NoError(t, err)
	require.JSONEq(t, `{"previous":{"name":"Gone"}}`, string(raw))
}
