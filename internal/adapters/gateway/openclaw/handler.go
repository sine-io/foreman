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

type commandBus interface {
	Dispatch(context.Context, manageragent.Command) (manageragent.Result, error)
}

type Handler struct {
	commands commandBus
	queries  any
}

func NewHandler(commands commandBus, queries any) *Handler {
	return &Handler{
		commands: commands,
		queries:  queries,
	}
}

func DecodeEnvelope(payload []byte) (manageragent.Command, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return manageragent.Command{}, err
	}

	return mapEnvelope(env), nil
}

func EncodeResponse(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func (h *Handler) Handle(ctx context.Context, env Envelope) (Response, error) {
	result, err := h.commands.Dispatch(ctx, mapEnvelope(env))
	if err != nil {
		return Response{}, err
	}

	return EncodeDomainResult(result), nil
}

func EncodeDomainResult(result manageragent.Result) Response {
	return Response{
		Kind:    result.Kind,
		TaskID:  result.TaskID,
		Summary: result.Summary,
	}
}

func mapEnvelope(env Envelope) manageragent.Command {
	return manageragent.Command{
		Kind:      env.Action,
		SessionID: env.SessionID,
		TaskID:    env.TaskID,
		Summary:   env.Summary,
	}
}
