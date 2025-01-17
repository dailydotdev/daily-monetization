package util

import (
	"encoding/json"

	"github.com/imdario/mergo"
)

func MergeJsonToStructWithOverride[T any](t *T, j json.RawMessage) error {
	var override T
	if err := json.Unmarshal(j, &override); err != nil {
		return err
	}

	return mergo.Merge(t, override, mergo.WithOverride)
}
