package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestUnmarshal(t *testing.T) {
	t.Run("ReturnsValuesIfItContainsYaml", func(t *testing.T) {

		customProperties := `
action: test
values: |-
  secret:
    letsencryptAccountJson='{}'
    letsencryptAccountKey=abc
`

		// act
		var params params
		err := yaml.Unmarshal([]byte(customProperties), &params)

		if assert.Nil(t, err) {
			assert.Equal(t, "secret:\n  letsencryptAccountJson='{}'\n  letsencryptAccountKey=abc", params.Values)
		}
	})
}
