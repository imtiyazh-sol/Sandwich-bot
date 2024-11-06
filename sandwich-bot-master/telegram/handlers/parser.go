package handlers

import (
	"fmt"
	"strings"
)

func ParseResponseMap(obj map[string]interface{}, soughtField string) (interface{}, error) {
	fields := strings.Split(soughtField, ".")
	currentObj := obj
	for _i, field := range fields {
		if val, ok := currentObj[field]; ok {
			if _i+1 == len(fields) {
				return val, nil
			} else {
				if nextObj, ok := val.(map[string]interface{}); ok {
					currentObj = nextObj
				}
			}
		} else {
			return nil, fmt.Errorf("Field %s not found in object", field)
		}
	}

	return currentObj, nil
}
