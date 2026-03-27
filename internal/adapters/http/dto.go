package http

type reprioritizeRequest struct {
	Priority int `json:"priority"`
}

type taskActionResponse struct {
	State string `json:"state"`
}
