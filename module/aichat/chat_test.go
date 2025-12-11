package aichat_test

import (
	"testing"

	"github.com/forbearing/gst/client"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

const (
	authropicModelID = "claude-sonnet-4-5-20250929"
	openaiModelID    = "gpt-4o"
	ollamaModelID    = "gpt-oss:20b"
)

func TestChatCompletion(t *testing.T) {
	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token))
	require.NoError(t, err)

	t.Run("non-stream", func(t *testing.T) {
		t.Run("authropic", func(t *testing.T) {})
		t.Run("openai", func(t *testing.T) {})

		t.Run("local", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID: ollamaModelID,
				Messages: []string{
					"你现在是一个精通 go 语言的专家",
					"简单介绍下 Golang 这门语言,包括它的特性、用法和适用场景。",
				},
				Stream: false,
			}

			require.NoError(t, err)

			rsp, err := cli.Create(req)
			require.NoError(t, err)
			pretty.Println(rsp.Data)
		})
	})

	t.Run("stream", func(t *testing.T) {
		t.Run("authropic", func(t *testing.T) {})
		t.Run("openai", func(t *testing.T) {})
		t.Run("local", func(t *testing.T) {})
	})
}
