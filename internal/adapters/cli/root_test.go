package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeApp struct {
	serveCalled bool
}

func (f *fakeApp) Serve(context.Context) error {
	f.serveCalled = true
	return nil
}

func TestRootCommandRequiresSubcommand(t *testing.T) {
	cmd := NewRootCommand(&fakeApp{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.Error(t, err)
}

func TestServeCommandCallsAppServe(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.True(t, app.serveCalled)
}
