package mock

import (
	"encoding/json"
	"fmt"
)

type MarshaledError struct {
	Data any
}

func (m *MarshaledError) Error() string {
	return fmt.Sprintf("%v", m.Data)
}

func (m *MarshaledError) MarshalJSON() ([]byte, error) {
	type custom struct {
		Whoops any `json:"whoops"`
	}

	return json.Marshal(custom{Whoops: m.Data})
}

type HealthCheck struct {
	Result any
	Err    error
}

func (m *HealthCheck) HealthCheck() (any, error) {
	return m.Result, m.Err
}
