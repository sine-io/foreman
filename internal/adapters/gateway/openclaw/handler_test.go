package openclaw

import (
	"context"
	"testing"

	"github.com/sine-io/foreman/internal/adapters/gateway/manageragent"
	"github.com/stretchr/testify/require"
)

func TestOpenClawEnvelopeMapsToCreateTaskCommand(t *testing.T) {
	payload := []byte(`{"session_id":"oc-session-1","action":"create_task","summary":"Build board query"}`)

	cmd, err := DecodeEnvelope(payload)
	require.NoError(t, err)
	require.Equal(t, "create_task", cmd.Kind)
}

func TestOpenClawEncodesApprovalNeededResponse(t *testing.T) {
	msg, err := EncodeResponse(Response{
		Kind:    "approval_needed",
		TaskID:  "task-1",
		Summary: "git push origin main requires approval",
	})
	require.NoError(t, err)
	require.Contains(t, string(msg), "approval_needed")
}

func TestOpenClawHandlerReturnsCompletionResponse(t *testing.T) {
	handler := NewHandler(
		fakeCommandBus{
			result: manageragent.Result{
				Kind:    "completion",
				TaskID:  "task-1",
				Summary: "task created",
			},
		},
		fakeQueryBus{},
	)

	resp, err := handler.Handle(context.Background(), Envelope{
		SessionID: "oc-session-1",
		Action:    "create_task",
		Summary:   "task created",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Kind)
	require.Equal(t, "completion", resp.Kind)
}

type fakeCommandBus struct {
	result manageragent.Result
}

func (f fakeCommandBus) Dispatch(ctx context.Context, cmd manageragent.Command) (manageragent.Result, error) {
	return f.result, nil
}

type fakeQueryBus struct{}
