package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexInt menerima JSON number ATAUPUN JSON string berisi angka saat decode
// (mis. 5 atau "5"). Karena underlying type-nya int, konversi ke int biasa
// cukup pakai type cast langsung: int(myFlexInt) - tidak perlu method.
type FlexInt int

func (f *FlexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		*f = 0
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("nilai %q bukan angka bulat yang valid", s)
		}
		*f = FlexInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("nilai %s bukan angka bulat yang valid", string(data))
	}
	*f = FlexInt(n)
	return nil
}

func (f FlexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(f))
}

// FlexFloat sama seperti FlexInt tapi untuk float64 (mis. price, ppn value).
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		*f = 0
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("nilai %q bukan angka yang valid", s)
		}
		*f = FlexFloat(n)
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("nilai %s bukan angka yang valid", string(data))
	}
	*f = FlexFloat(n)
	return nil
}

func (f FlexFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}