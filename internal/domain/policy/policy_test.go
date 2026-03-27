package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictPolicyRequiresApprovalForGitPush(t *testing.T) {
	decision := EvaluateStrictAction("git push origin main")

	require.True(t, decision.RequiresApproval)
	require.NotEmpty(t, decision.Reason)
}
