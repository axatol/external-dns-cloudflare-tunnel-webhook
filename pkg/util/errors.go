package util

import (
	"encoding/json"
	"strings"
)

type ErrorList []error

func (e *ErrorList) Error() string {
	if len(*e) == 0 {
		return ""
	}

	raw := make([]string, 0, len(*e))
	for _, err := range *e {
		raw = append(raw, err.Error())
	}

	return strings.Join(raw, "; ")
}

func (e *ErrorList) Add(err error) {
	*e = append(*e, err)
}

func (e *ErrorList) MarshalJSON() ([]byte, error) {
	return json.Marshal(*e)
}
