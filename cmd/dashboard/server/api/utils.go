package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// getIntVal gets int value by key from url request, it returns default value when key not found
func getIntVal(vars url.Values, key string, defaultVal, minVal int) (int, error) {
	if v, ok := vars[key]; ok {
		l, err := strconv.Atoi(v[0])
		if err != nil {
			return 0, err

		}
		if l < minVal {
			return 0, fmt.Errorf("wrong limit %d", l)
		}
		return l, nil
	}
	return defaultVal, nil
}
