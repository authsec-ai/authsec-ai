package services

import (
	"encoding/json"
	"gorm.io/datatypes"
)

// MapToJSON helper to convert map to datatypes.JSON
func MapToJSON(m map[string]interface{}) (datatypes.JSON, error) {
	if m == nil {
		return datatypes.JSON("{}"), nil // Return empty JSON object
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}
