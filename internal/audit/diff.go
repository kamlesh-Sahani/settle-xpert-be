package audit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	auditmodels "lead-crm/backend/internal/audit/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FieldDiff represents a single field change between old and new state.
type FieldDiff struct {
	Field        string `json:"field"`
	OlderValue   string `json:"older_value"`
	ChangedValue string `json:"changed_value"`
}

// Ignored internal fields that should not appear in diffs
var ignoredFields = map[string]bool{
	"_id":                            true,
	"id":                             true,
	"tenant_id":                      true,
	"created_at":                     true,
	"updated_at":                     true,
	"deleted_at":                     true,
	"password":                       true,
	"hash":                           true,
	"tenantid":                       true,
	"createdat":                      true,
	"updatedat":                      true,
	"deletedat":                      true,
	"fake_username_prevent_autofill": true,
	"fake_password_prevent_autofill": true,
}

// ComputeDiff calculates field-level differences and the total number of changes.
func ComputeDiff(oldData, newData any, action string) ([]FieldDiff, int) {
	action = strings.ToUpper(strings.TrimSpace(action))
	newMap := toDataMap(newData)
	oldMap := toDataMap(oldData)

	var diffs []FieldDiff

	// 1. CREATE Action: all fields in newMap are new
	if action == auditmodels.ActionCreate || len(oldMap) == 0 {
		keys := getSortedKeys(newMap)
		for _, k := range keys {
			if isIgnored(k) {
				continue
			}
			newValStr := formatVal(newMap[k])
			if newValStr == "" || newValStr == "NA" {
				continue
			}
			diffs = append(diffs, FieldDiff{
				Field:        HumanizeField(k),
				OlderValue:   "NA",
				ChangedValue: newValStr,
			})
		}
		return diffs, len(diffs)
	}

	// 2. DELETE Action: record deleted state
	if action == auditmodels.ActionDelete || len(newMap) == 0 {
		keys := getSortedKeys(oldMap)
		for _, k := range keys {
			if isIgnored(k) {
				continue
			}
			oldValStr := formatVal(oldMap[k])
			if oldValStr == "" || oldValStr == "NA" {
				continue
			}
			diffs = append(diffs, FieldDiff{
				Field:        HumanizeField(k),
				OlderValue:   oldValStr,
				ChangedValue: "Deleted",
			})
		}
		return diffs, len(diffs)
	}

	// 3. UPDATE Action: compare oldMap and newMap
	// If newMap is a partial update (contains fewer fields than oldMap, e.g. updateFields map),
	// only the candidate fields explicitly present in newMap are compared.
	var candidateKeys []string
	if len(newMap) < len(oldMap) {
		candidateKeys = getSortedKeys(newMap)
	} else {
		allKeys := make(map[string]bool)
		for k := range oldMap {
			if !isIgnored(k) {
				allKeys[k] = true
			}
		}
		for k := range newMap {
			if !isIgnored(k) {
				allKeys[k] = true
			}
		}
		candidateKeys = make([]string, 0, len(allKeys))
		for k := range allKeys {
			candidateKeys = append(candidateKeys, k)
		}
		sort.Strings(candidateKeys)
	}

	for _, k := range candidateKeys {
		if isIgnored(k) {
			continue
		}

		oldVal, oldExists := oldMap[k]
		newVal, newExists := newMap[k]

		oldValStr := "NA"
		if oldExists && oldVal != nil {
			oldValStr = formatVal(oldVal)
		}

		newValStr := "NA"
		if newExists && newVal != nil {
			newValStr = formatVal(newVal)
		}

		if !valuesEqual(oldVal, newVal) && (oldValStr != newValStr) {
			diffs = append(diffs, FieldDiff{
				Field:        HumanizeField(k),
				OlderValue:   oldValStr,
				ChangedValue: newValStr,
			})
		}
	}

	return diffs, len(diffs)
}

func isIgnored(key string) bool {
	return ignoredFields[strings.ToLower(key)]
}

// toSnakeCase converts camelCase, TitleCase, or kebab-case into snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(s[i-1])
			if prev != '_' && (prev < 'A' || prev > 'Z') {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return strings.ReplaceAll(string(result), "-", "_")
}

// toDataMap converts any struct, map, or BSON object into a map with canonical snake_case keys.
func toDataMap(data any) map[string]any {
	if data == nil {
		return nil
	}

	// 1. Direct type switches for BSON/Map types
	switch v := data.(type) {
	case primitive.D:
		res := make(map[string]any, len(v))
		for _, elem := range v {
			res[toSnakeCase(elem.Key)] = normalizeValue(elem.Value)
		}
		return res
	case *primitive.D:
		if v == nil {
			return nil
		}
		res := make(map[string]any, len(*v))
		for _, elem := range *v {
			res[toSnakeCase(elem.Key)] = normalizeValue(elem.Value)
		}
		return res
	case primitive.M:
		res := make(map[string]any, len(v))
		for k, val := range v {
			res[toSnakeCase(k)] = normalizeValue(val)
		}
		return res
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, val := range v {
			res[toSnakeCase(k)] = normalizeValue(val)
		}
		return res
	}

	// 2. Fallback via JSON for Go structs
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}

	res := make(map[string]any, len(raw))
	for k, v := range raw {
		res[toSnakeCase(k)] = normalizeValue(v)
	}
	return res
}

func normalizeValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case primitive.D:
		res := make(map[string]any, len(val))
		for _, elem := range val {
			res[toSnakeCase(elem.Key)] = normalizeValue(elem.Value)
		}
		return res
	case *primitive.D:
		if val == nil {
			return nil
		}
		res := make(map[string]any, len(*val))
		for _, elem := range *val {
			res[toSnakeCase(elem.Key)] = normalizeValue(elem.Value)
		}
		return res
	case primitive.A:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = normalizeValue(item)
		}
		return arr
	case primitive.M:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[toSnakeCase(k)] = normalizeValue(item)
		}
		return res
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[toSnakeCase(k)] = normalizeValue(item)
		}
		return res
	default:
		return val
	}
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare numeric types numerically
	aFloat, aIsNum := toFloat(a)
	bFloat, bIsNum := toFloat(b)
	if aIsNum && bIsNum {
		return aFloat == bFloat
	}

	// Compare string types
	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			return strings.TrimSpace(aStr) == strings.TrimSpace(bStr)
		}
	}

	// Compare ObjectIDs
	if aObj, ok := a.(primitive.ObjectID); ok {
		if bObj, ok := b.(primitive.ObjectID); ok {
			return aObj == bObj
		}
		if bStr, ok := b.(string); ok {
			return aObj.Hex() == bStr
		}
	}
	if bObj, ok := b.(primitive.ObjectID); ok {
		if aStr, ok := a.(string); ok {
			return bObj.Hex() == aStr
		}
	}

	if reflect.DeepEqual(a, b) {
		return true
	}

	return formatVal(a) == formatVal(b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func formatVal(val any) string {
	if val == nil {
		return "NA"
	}
	switch v := val.(type) {
	case primitive.ObjectID:
		return v.Hex()
	case primitive.DateTime:
		return v.Time().Format("2006-01-02 15:04")
	case time.Time:
		return v.Format("2006-01-02 15:04")
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return "NA"
		}
		// If string contains JSON array or object, parse and humanize it
		if (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) ||
			(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) {
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return formatVal(parsed)
			}
		}
		return trimmed
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		f := float64(v)
		if f == float64(int64(f)) {
			return fmt.Sprintf("%d", int64(f))
		}
		return strconv.FormatFloat(f, 'f', -1, 32)
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case []any:
		return formatSlice(v)
	case primitive.A:
		arr := make([]any, len(v))
		for i, item := range v {
			arr[i] = item
		}
		return formatSlice(arr)
	case []string:
		if len(v) == 0 {
			return "NA"
		}
		return strings.Join(v, ", ")
	case map[string]any:
		return formatMap(v)
	case primitive.M:
		return formatMap(normalizeValue(v).(map[string]any))
	case primitive.D:
		return formatMap(normalizeValue(v).(map[string]any))
	default:
		// Try map conversion
		if m, ok := toItemMap(v); ok && len(m) > 0 {
			return formatMap(m)
		}
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		str := string(b)
		if str == "null" || str == `""` || str == "{}" || str == "[]" {
			return "NA"
		}
		return str
	}
}

func toItemMap(item any) (map[string]any, bool) {
	if item == nil {
		return nil, false
	}
	switch m := item.(type) {
	case map[string]any:
		res := make(map[string]any, len(m))
		for k, v := range m {
			res[toSnakeCase(k)] = v
		}
		return res, true
	case primitive.M:
		res := make(map[string]any, len(m))
		for k, v := range m {
			res[toSnakeCase(k)] = v
		}
		return res, true
	case primitive.D:
		res := make(map[string]any, len(m))
		for _, e := range m {
			res[toSnakeCase(e.Key)] = e.Value
		}
		return res, true
	default:
		b, err := json.Marshal(item)
		if err != nil {
			return nil, false
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			return nil, false
		}
		res := make(map[string]any, len(raw))
		for k, v := range raw {
			res[toSnakeCase(k)] = v
		}
		return res, true
	}
}

func formatMap(m map[string]any) string {
	if len(m) == 0 {
		return "NA"
	}
	keys := getSortedKeys(m)
	var parts []string
	for _, k := range keys {
		if isIgnored(k) {
			continue
		}
		vStr := formatVal(m[k])
		if vStr == "" || vStr == "NA" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", HumanizeField(k), vStr))
	}
	if len(parts) == 0 {
		return "NA"
	}
	return strings.Join(parts, ", ")
}

func formatSlice(items []any) string {
	if len(items) == 0 {
		return "NA"
	}
	var parts []string
	for _, it := range items {
		if it == nil {
			continue
		}
		formatted := formatVal(it)
		if formatted != "" && formatted != "NA" {
			parts = append(parts, formatted)
		}
	}
	if len(parts) == 0 {
		return "NA"
	}
	return strings.Join(parts, ", ")
}

func getSortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// HumanizeField converts camelCase or snake_case to Title Case (e.g. "team_id" -> "Team ID")
func HumanizeField(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Handle camelCase
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' && s[i-1] >= 'a' && s[i-1] <= 'z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}

	words := strings.Fields(string(result))
	for i, w := range words {
		if len(w) > 0 {
			upper := strings.ToUpper(w)
			if upper == "SKU" || upper == "HSN" || upper == "SAC" || upper == "ID" || upper == "URL" || upper == "PAN" || upper == "DOB" {
				words[i] = upper
			} else {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
	}
	return strings.Join(words, " ")
}
