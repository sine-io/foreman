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

func TestOpenClawHandlerDelegatesToManagerService(t *testing.T) {
	svc := fakeManagerService{
		response: manageragent.Response{
			Kind:   "completion",
			TaskID: "task-1",
		},
	}

	handler := NewHandler(&svc)
	resp, err := handler.Handle(context.Background(), Envelope{
		SessionID: "oc-1",
		Action:    "create_task",
		Summary:   "Bootstrap board",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", resp.Kind)
	require.Equal(t, manageragent.Request{
		Kind:      "create_task",
		SessionID: "oc-1",
		Summary:   "Bootstrap board",
	}, svc.received)
}

type fakeManagerService struct {
	response manageragent.Response
	received manageragent.Request
}

func (f *fakeManagerService) Handle(ctx context.Context, req manageragent.Request) (manageragent.Response, error) {
	f.received = req
	return f.response, nil
}
