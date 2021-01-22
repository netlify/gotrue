package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFunctionHooksUnmarshalJSON(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{`{ "signup" : "identity-signup" }`, true},
		{`{ "signup" : ["identity-signup"] }`, true},
		{`{ "signup" : {"foo" : "bar"} }`, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var f FunctionHooks
			err := json.Unmarshal([]byte(tt.in), &f)
			if tt.ok {
				assert.NoError(t, err)
				assert.Equal(t, FunctionHooks{"signup": {"identity-signup"}}, f)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
