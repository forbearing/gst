package aichat_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/forbearing/gst/client"
	"github.com/forbearing/gst/database"
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

type ChatData struct {
	Content string `json:"content"`
	Delta   string `json:"delta"`
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
			var rsp *client.Resp
			rsp, err = cli.Create(req)
			require.NoError(t, err)
			pretty.Println(string(rsp.Data))
		})
		t.Run("openai", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  openaiModelID,
				Messages: messages,
				Stream:   false,
			}
			var rsp *client.Resp
			rsp, err = cli.Create(req)
			require.NoError(t, err)
			pretty.Println(string(rsp.Data))
		})

		t.Run("local", func(t *testing.T) {
			req := modelaichat.ChatCompletionReq{
				ModelID:  ollamaModelID,
				Messages: messages,
				Stream:   false,
			}
			var rsp *client.Resp
			rsp, err = cli.Create(req)
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
				var data ChatData
				v1 := fmt.Sprintf("%s", event.Data)
				v2 := []byte(v1)
				_ = json.Unmarshal(v2, &data)
				fmt.Printf("%s", data.Delta)
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
				var data ChatData
				v1 := fmt.Sprintf("%s", event.Data)
				v2 := []byte(v1)
				_ = json.Unmarshal(v2, &data)
				fmt.Printf("%s", data.Delta)
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
				var data ChatData
				v1 := fmt.Sprintf("%s", event.Data)
				v2 := []byte(v1)
				_ = json.Unmarshal(v2, &data)
				fmt.Printf("%s", data.Delta)
				return nil
			})
			require.NoError(t, err)
		})
	})
}

func TestStopMessage(t *testing.T) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		// waiting for ai chat is starting and streaming.
		time.Sleep(5 * time.Second)

		// query the latest message that is "streaming"
		msg := new(modelaichat.Message)
		err := database.Database[*modelaichat.Message](nil).WithQuery(&modelaichat.Message{Status: modelaichat.MessageStatusStreaming}).Last(msg)
		require.NoError(t, err)
		require.NotNil(t, msg)

		// send stop message request.
		cli, err := client.New(addr+"/ai/messages/stop", client.WithToken(token))
		require.NoError(t, err)
		var rsp *client.Resp
		rsp, err = cli.Create(modelaichat.StopMessageReq{
			MessageID: msg.ID,
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	}()

	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token), client.WithTimeout(3*time.Minute))
	require.NoError(t, err)

	err = cli.Stream(modelaichat.ChatCompletionReq{
		ModelID:  ollamaModelID,
		Messages: messages,
		Stream:   true,
	}, func(event types.Event) error {
		var data ChatData
		v1 := fmt.Sprintf("%s", event.Data)
		v2 := []byte(v1)
		_ = json.Unmarshal(v2, &data)
		// fmt.Printf("%s", data.Delta)
		return nil
	})
	require.NoError(t, err)

	<-done
}

func TestRegenerateMessage(t *testing.T) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		// waiting for ai chat is starting and streaming.
		time.Sleep(5 * time.Second)

		// query the latest message that is "streaming"
		msg := new(modelaichat.Message)
		err := database.Database[*modelaichat.Message](nil).WithQuery(&modelaichat.Message{Status: modelaichat.MessageStatusStreaming}).Last(msg)
		require.NoError(t, err)
		require.NotNil(t, msg)

		// send stop message request.
		cli, err := client.New(addr+"/ai/messages/stop", client.WithToken(token))
		require.NoError(t, err)
		var rsp *client.Resp
		rsp, err = cli.Create(modelaichat.StopMessageReq{
			MessageID: msg.ID,
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))

		time.Sleep(3 * time.Second)

		//
		// regenerate messsage.
		//

		// query the latest message that is "stopped"
		require.NoError(t, database.Database[*modelaichat.Message](nil).WithQuery(&modelaichat.Message{Status: modelaichat.MessageStatusStopped}).Last(msg))
		require.NotNil(t, msg)
		// send regenerate message request
		cli, err = client.New(addr+"/ai/messages/regenerate", client.WithToken(token))
		require.NoError(t, err)
		err = cli.Stream(modelaichat.RegenerateMessageReq{
			MessageID: msg.ID,
			Stream:    true,
		}, func(event types.Event) error {
			var data ChatData
			v1 := fmt.Sprintf("%s", event.Data)
			v2 := []byte(v1)
			_ = json.Unmarshal(v2, &data)
			fmt.Printf("%s", data.Delta)
			return nil
		})
		require.NoError(t, err)
	}()

	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token), client.WithTimeout(3*time.Minute))
	require.NoError(t, err)

	err = cli.Stream(modelaichat.ChatCompletionReq{
		ModelID:  ollamaModelID,
		Messages: messages,
		Stream:   true,
	}, func(event types.Event) error {
		var data ChatData
		v1 := fmt.Sprintf("%s", event.Data)
		v2 := []byte(v1)
		_ = json.Unmarshal(v2, &data)
		// fmt.Printf("%s", data.Delta)
		return nil
	})
	require.NoError(t, err)

	<-done
}

func TestConversationTitle(t *testing.T) {
	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token), client.WithTimeout(3*time.Minute))
	require.NoError(t, err)

	messages := []string{
		"", // empty message.
		"", // empty message.
		"你现在是一个精通 go 语言的专家, 简单介绍下 golang 这门语言,包括它的特性、用法和适用场景。",
	}

	t.Run("non-stream", func(t *testing.T) {
		req := modelaichat.ChatCompletionReq{
			ModelID:  ollamaModelID,
			Messages: messages,
			Stream:   false,
		}
		var rsp *client.Resp
		rsp, err = cli.Create(req)
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Run("stream", func(t *testing.T) {
		req := modelaichat.ChatCompletionReq{
			ModelID:  ollamaModelID,
			Messages: messages,
			Stream:   true,
		}
		err = cli.Stream(req, func(event types.Event) error {
			var data ChatData
			v1 := fmt.Sprintf("%s", event.Data)
			v2 := []byte(v1)
			_ = json.Unmarshal(v2, &data)
			fmt.Printf("%s", data.Delta)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestSubmitMessageFeedback(t *testing.T) {
	cli, err := client.New(addr+"/ai/conversations/chat", client.WithToken(token), client.WithTimeout(3*time.Minute))
	require.NoError(t, err)

	req := modelaichat.ChatCompletionReq{
		ModelID:  ollamaModelID,
		Messages: messages,
		Stream:   true,
	}
	err = cli.Stream(req, func(event types.Event) error {
		var data ChatData
		v1 := fmt.Sprintf("%s", event.Data)
		v2 := []byte(v1)
		_ = json.Unmarshal(v2, &data)
		fmt.Printf("%s", data.Delta)
		return nil
	})
	require.NoError(t, err)

	msg := new(modelaichat.Message)
	require.NoError(t, database.Database[*modelaichat.Message](nil).Last(msg))

	cli, err = client.New(addr+"/ai/messages/feedback", client.WithToken(token))
	require.NoError(t, err)
	t.Run("first feedback", func(t *testing.T) {
		var rsp *client.Resp
		rsp, err = cli.Create(modelaichat.SubmitMessageFeedbackReq{
			MessageID: msg.ID,
			Type:      modelaichat.FeedbackLike,
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})

	t.Log("wait for 3 seconds to feedback again")
	time.Sleep(3 * time.Second)
	t.Run("second feedback", func(t *testing.T) {
		var rsp *client.Resp
		rsp, err = cli.Create(modelaichat.SubmitMessageFeedbackReq{
			MessageID: msg.ID,
			Type:      modelaichat.FeedbackDislike,
			Categories: []modelaichat.FeedbackCategory{
				modelaichat.FeedbackCategoryNotHelpful,
				modelaichat.FeedbackCategoryIncomplete,
			},
		})
		require.NoError(t, err)
		pretty.Println(string(rsp.Data))
	})
}
