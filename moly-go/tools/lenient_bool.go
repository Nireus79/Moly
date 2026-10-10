package tools

import (
	"bytes"
	"encoding/json"
	"strings"
)

// LenientBool reads a JSON boolean that the model may have written as a string ("true", "false")
// or a number. Anything it cannot read is false, so a type quirk never fails a safety verdict.
type LenientBool bool

func (b *LenientBool) UnmarshalJSON(data []byte) error {
	s := strings.ToLower(strings.Trim(string(bytes.TrimSpace(data)), `"`))
	*b = s == "true" || s == "yes" || s == "1"
	return nil
}

var _ json.Unmarshaler = (*LenientBool)(nil)
