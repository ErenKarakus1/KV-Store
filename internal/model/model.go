package model

type SetRequest struct {
	Value string `json:"value"`
	TTL   string `json:"ttl,omitempty"`
}
