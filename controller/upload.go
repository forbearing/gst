package controller

import (
	"io"

	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/provider/minio"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type upload struct{}

var Upload = new(upload)

func (*upload) Put(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("Put"))

	// NOTE:字段为 file 必须和前端协商好.
	file, err := c.FormFile("file")
	if err != nil {
		log.Error(err)
		ResponseJSON(c, CodeFailure)
		return
	}
	// check file size.
	if file.Size > MAX_UPLOAD_SIZE {
		log.Error(CodeTooLargeFile)
		ResponseJSON(c, CodeTooLargeFile)
		return
	}
	fd, err := file.Open()
	if err != nil {
		log.Error(err)
		ResponseJSON(c, CodeFailure)
		return
	}
	defer fd.Close()

	info, putErr := minio.Put(c.Request.Context(), file.Filename, fd, &minio.PutOptions{
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
	})
	if putErr != nil {
		err = putErr
		zap.S().Error(err)
		ResponseJSON(c, CodeFailure)
		return
	}
	ResponseJSON(c, CodeSuccess, gin.H{
		"filename": info.Key,
	})
}

func (*upload) Preview(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("Preview"))
	data, info, getErr := minio.Get(c.Request.Context(), c.Param(consts.PARAM_FILE))
	if getErr != nil {
		log.Error(getErr)
		ResponseJSON(c, CodeFailure)
		return
	}
	defer data.Close()

	content, readErr := io.ReadAll(data)
	if readErr != nil {
		log.Error(readErr)
		ResponseJSON(c, CodeFailure)
		return
	}
	headers := map[string]string{}
	if info != nil && info.ContentType != "" {
		headers["Content-Type"] = info.ContentType
	}
	ResponseDATA(c, content, headers)
}
