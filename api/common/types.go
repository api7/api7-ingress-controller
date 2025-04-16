// +kubebuilder:object:generate=true

package common

import (
	"encoding/json"
	"errors"
)

// Vars represents the route match expressions of APISIX.
type Vars [][]StringOrSlice

// UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (vars *Vars) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var data [][]StringOrSlice
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*vars = data
	return nil
}

// StringOrSlice represents a string or a string slice.
// TODO Do not use interface{} to avoid the reflection overheads.
//
// +kubebuilder:validation:Schemaless
type StringOrSlice struct {
	StrVal   string          `json:"-"`
	SliceVal []StringOrSlice `json:"-"`
}

func (s *StringOrSlice) MarshalJSON() ([]byte, error) {
	if s.SliceVal != nil {
		return json.Marshal(s.SliceVal)
	}
	return json.Marshal(s.StrVal)
}

func (s *StringOrSlice) UnmarshalJSON(p []byte) error {
	if len(p) == 0 {
		return errors.New("empty object")
	}
	if p[0] == '[' {
		return json.Unmarshal(p, &s.SliceVal)
	}
	return json.Unmarshal(p, &s.StrVal)
}
