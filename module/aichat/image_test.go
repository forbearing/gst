package aichat_test

import (
	"testing"

	"github.com/forbearing/gst/client"
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

func TestImageGeneration(t *testing.T) {
	cli, err := client.New(addr+"/ai/images/generate", client.WithToken(token))
	require.NoError(t, err)

	imageModelID := "dall-e-3"

	rsp, err := cli.Create(modelaichat.ImageGenerationReq{
		Prompt: "a cute cat",
		Model:  imageModelID,
	})
	require.NoError(t, err)
	pretty.Println(string(rsp.Data))
}
