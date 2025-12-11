package aichat_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/forbearing/gst/client"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

const (
	anthropicModelID = "claude-3-7-sonnet-20250219"
	openaiModelID    = "gpt-4o"
	ollamaModelID    = "llama3:latest"
)

var messages = []string{
	"你现在是一个精通 go 语言的专家",
	"简单介绍下 golang 这门语言,包括它的特性、用法和适用场景。",
}

func TestChatCompletion(t *testing.T) {
	cliSync, err := client.New(addr+"/ai/providers/sync-models", client.WithToken(token))
	require.NoError(t, err)
	// sync model first
	_, err = cliSync.Create(modelaichat.Provider{
		Base: model.Base{ID: anthropicID},
	})
	require.NoError(t, err)
	_, err = cliSync.Create(modelaichat.Provider{
		Base: model.Base{ID: openaiID},
	})
	require.NoError(t, err)
	_, err = cliSync.Create(modelaichat.Provider{
		Base: model.Base{ID: ollamaID},
	})
	require.NoError(t, err)

	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token), client.WithTimeout(3*time.Minute))
	require.NoError(t, err)
	t.Run("non-stream", func(t *testing.T) {
		t.Run("anthropic", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  anthropicModelID,
				Messages: messages,
				Stream:   false,
			}
			rsp, err := cli.Create(req)
			require.NoError(t, err)
			pretty.Println(string(rsp.Data))
		})
		t.Run("openai", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  openaiModelID,
				Messages: messages,
				Stream:   false,
			}
			rsp, err := cli.Create(req)
			require.NoError(t, err)
			pretty.Println(string(rsp.Data))
		})

		t.Run("local", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  ollamaModelID,
				Messages: messages,
				Stream:   false,
			}
			rsp, err := cli.Create(req)
			require.NoError(t, err)
			pretty.Println(string(rsp.Data))
		})
	})

	t.Run("stream", func(t *testing.T) {
		t.Run("anthropic", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  anthropicModelID,
				Messages: messages,
				Stream:   true,
			}
			err = cli.Stream(req, func(event types.Event) error {
				fmt.Printf("%s", event.Data)
				return nil
			})
			require.NoError(t, err)
		})
		t.Run("openai", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  openaiModelID,
				Messages: messages,
				Stream:   true,
			}
			err = cli.Stream(req, func(event types.Event) error {
				fmt.Printf("%s", event.Data)
				return nil
			})
			require.NoError(t, err)
		})
		t.Run("local", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  ollamaModelID,
				Messages: messages,
				Stream:   true,
			}
			err = cli.Stream(req, func(event types.Event) error {
				fmt.Printf("%s", event.Data)
				return nil
			})
			require.NoError(t, err)
		})
	})
}
