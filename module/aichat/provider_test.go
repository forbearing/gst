package aichat_test

import (
	"testing"

	"github.com/forbearing/gst/client"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

func TestProviderSyncModels(t *testing.T) {
	cli, err := client.New(addr+"/ai/providers/sync-models", client.WithToken(token))
	require.NoError(t, err)

	rsp, err := cli.Create(modelaichat.Provider{
		Base: model.Base{
			ID: "019b0d1a-1563-7c19-81f4-2a29087d073c",
		},
	})
	require.NoError(t, err)
	pretty.Println(string(rsp.Data))
}
