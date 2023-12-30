package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const jsonStr = `
  {
		"name": "John Doe",
		"age": 30,
		"address": {
			"street": "123 Main St",
			"city": "New York",
			"state": "NY",
			"zip": "10001"
		},
		"phones": [
			{
				"type": "home",
				"number": "555-1234"
			},
			{
				"type": "work",
				"number": "555-5678"
			}
		]
	}`

const invalidJsonStr = `
  {
		"name": "John Doe",
		"age": 30,
		"address": {
			"street": "123 Main St",
			"city": "New York",
			"state": "NY",
			"zip": "10001"
		},
  `

func Test_ParseJSON(t *testing.T) {
	t.Run("parse correct json", func(t *testing.T) {
		res, err := ParseJSON(jsonStr)

		expected := map[string]interface{}{
			"name": "John Doe",
			"age":  float64(30),
			"address": map[string]interface{}{
				"street": "123 Main St",
				"city":   "New York",
				"state":  "NY",
				"zip":    "10001",
			},
			"phones": []interface{}{
				map[string]interface{}{
					"type":   "home",
					"number": "555-1234",
				},
				map[string]interface{}{
					"type":   "work",
					"number": "555-5678",
				},
			},
		}
		assert.Nil(t, err, "should not error")
		assert.Equal(t, expected, res)

	})

	t.Run("parse invalid json", func(t *testing.T) {
		_, err := ParseJSON(invalidJsonStr)
		assert.NotNil(t, err, "should err")
	})
}
