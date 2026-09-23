package model

type SetRequest struct {
	Value string `json:"value"`
	TTL   string `json:"ttl,omitempty"`
}

type SetNXRequest struct {
	Value string `json:"value"`
}
