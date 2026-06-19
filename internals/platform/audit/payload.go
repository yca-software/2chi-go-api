package audit

// Change payloads stored in audit_logs.data use a consistent shape:
//   - create:  { "updated": { ...fields } }
//   - update:  { "previous": { ... }, "updated": { ... } }
//   - delete:  { "previous": { ... } }
//   - archive/restore: {} when there is no field-level diff

func CreatePayload(updated map[string]any) map[string]any {
	if updated == nil {
		updated = map[string]any{}
	}
	return map[string]any{"updated": updated}
}

func UpdatePayload(previous, updated map[string]any) map[string]any {
	payload := map[string]any{}
	if len(previous) > 0 {
		payload["previous"] = previous
	}
	if len(updated) > 0 {
		payload["updated"] = updated
	}
	return payload
}

func DeletePayload(previous map[string]any) map[string]any {
	if previous == nil {
		previous = map[string]any{}
	}
	return map[string]any{"previous": previous}
}
