package sdjson

import (
	"encoding/json"
	"strings"
)

func isFloat(n json.Number) bool {
	return strings.Contains(n.String(), ".")
}
