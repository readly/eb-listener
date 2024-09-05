package listen

import (
	"encoding/json"
	"time"
)

type Event struct {
	Version    string          `json:"version"`
	ID         string          `json:"id"`
	DetailType string          `json:"detail-type"`
	Source     string          `json:"source"`
	Account    string          `json:"account"`
	Time       time.Time       `json:"time"`
	Region     string          `json:"region"`
	Resources  []interface{}   `json:"resources"`
	Detail     json.RawMessage `json:"detail"`
}
