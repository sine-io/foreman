package openclaw

import (
	"context"
	"encoding/json"

	"github.com/sine-io/foreman/internal/adapters/gateway/manageragent"
)

type Envelope struct {
	SessionID string `json:"session_id"`
	Action    string `json:"action"`
	TaskID    string `json:"task_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

type Response struct {
	Kind    string `json:"kind"`
	TaskID  string `json:"task_id,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type Service interface {
	Handle(context.Context, manageragent.Request) (manageragent.Response, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func DecodeEnvelope(payload []byte) (manageragent.Request, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return manageragent.Request{}, err
	}

	return mapEnvelope(env), nil
}

func EncodeResponse(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func (h *Handler) Handle(ctx context.Context, env Envelope) (Response, error) {
	result, err := h.service.Handle(ctx, mapEnvelope(env))
	if err != nil {
		return Response{}, err
	}

	return encodeResponse(result), nil
}

func encodeResponse(result manageragent.Response) Response {
	return Response{
		Kind:    result.Kind,
		TaskID:  result.TaskID,
		Summary: result.Summary,
	}
}

func mapEnvelope(env Envelope) manageragent.Request {
	return manageragent.Request{
		Kind:      env.Action,
		SessionID: env.SessionID,
		TaskID:    env.TaskID,
		Summary:   env.Summary,
	}
}
