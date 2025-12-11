package aichat_test

import (
	"testing"

	"github.com/forbearing/gst/client"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

const (
	authropicID = "019b0d1a-1563-7c19-81f4-2a29087d073c"
	openaiID    = "019b0d99-3876-7c8e-b59c-918a07df2efc"
	ollamaID    = "019b0db0-2295-7c38-8ec0-e2f958b58331"
)

func TestProviderSyncModels(t *testing.T) {
	cli, err := client.New(addr+"/ai/providers/sync-models", client.WithToken(token))
	require.NoError(t, err)

	t.Run("authropic", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: authropicID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Run("openai", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: openaiID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Run("ollama", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: ollamaID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})
}

func TestProviderTestConn(t *testing.T) {
	cli, err := client.New(addr+"/ai/providers/test-conn", client.WithToken(token))
	require.NoError(t, err)

	t.Run("authropic", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: authropicID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Run("openai", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: openaiID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Run("ollama", func(t *testing.T) {
		rsp, err := cli.Create(modelaichat.Provider{
			Base: model.Base{ID: ollamaID},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})
}
