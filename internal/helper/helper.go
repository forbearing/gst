package helper

import (
	"encoding/json"
	"testing"

	"github.com/forbearing/gst/client"
	"github.com/forbearing/gst/response"
	"github.com/stretchr/testify/require"
)

func TestResp[RSP any](t *testing.T, resp *client.Resp, checkFn func(t *testing.T, rsp RSP)) {
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	require.Equal(t, resp.Code, response.CodeSuccess.Code())
	require.Equal(t, resp.Msg, response.CodeSuccess.Msg())
	require.NotEmpty(t, resp.RequestID)
	require.NotEmpty(t, resp.Data)

	var rsp RSP
	require.NoError(t, json.Unmarshal(resp.Data, &rsp))
	if checkFn != nil {
		checkFn(t, rsp)
	}
}
