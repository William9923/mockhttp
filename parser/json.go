package parser

import (
	"encoding/json"
)

func ParseJSON(jsonText string) (map[string]interface{}, error) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(jsonText), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
