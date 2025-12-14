package minio_test

import (
	"context"
	"os"
	"testing"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/logger/zap"
	"github.com/forbearing/gst/provider/minio"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv(config.MINIO_ENABLE, "true")
	os.Setenv(config.MINIO_BUCKET, "test-bucket")

	// os.Setenv(config.MINIO_ACCESS_KEY, "minio-access-key")
	// os.Setenv(config.MINIO_SECRET_KEY, "minio-secret-key")

	if err := config.Init(); err != nil {
		panic(err)
	}
	if err := zap.Init(); err != nil {
		panic(err)
	}
	if err := minio.Init(); err != nil {
		panic(err)
	}
}

func TestEnsureBucket(t *testing.T) {
	require.NoError(t, minio.EnsureBucket(context.TODO(), "bucket1", "bucket2", "bucket3"))

	exists1, err1 := minio.Client().BucketExists(context.TODO(), "bucket1")
	exists2, err2 := minio.Client().BucketExists(context.TODO(), "bucket2")
	exists3, err3 := minio.Client().BucketExists(context.TODO(), "bucket3")
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
	require.True(t, exists1)
	require.True(t, exists2)
	require.True(t, exists3)
}
