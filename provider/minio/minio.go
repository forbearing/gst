package minio

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/util"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var (
	initialized bool
	client      *minio.Client
	mu          sync.RWMutex
)

// Init initializes the global MinIO client.
// It reads MinIO configuration from config.App.Minio.
// If MinIO is not enabled, it returns nil.
// The function is thread-safe and ensures the client is initialized only once.
func Init() (err error) {
	cfg := config.App.Minio
	if !cfg.Enable {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if initialized {
		return nil
	}

	if client, err = New(cfg); err != nil {
		return errors.Wrap(err, "failed to create minio client")
	}

	// Try to establish a connection to MinIO and verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Multiple buckets separated by comma.
	buckets := strings.Split(cfg.Bucket, ",")
	if err := EnsureBucket(ctx, buckets...); err != nil {
		return err
	}

	zap.S().Infow("successfully connected to minio", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket, "region", cfg.Region)

	initialized = true
	return nil
}

// New returns a new MinIO client with given configuration.
func New(cfg config.Minio) (cli *minio.Client, err error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("minio endpoint is empty")
	}

	// Set up credentials options
	var creds *credentials.Credentials
	switch {
	case cfg.UseIAM:
		// Use IAM based credentials
		creds = credentials.NewIAM(cfg.IAMEndpoint)
	case cfg.UseSTS:
		// Use STS based credentials
		if creds, err = credentials.NewSTSAssumeRole(cfg.STSEndpoint, credentials.STSAssumeRoleOptions{
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		}); err != nil {
			return nil, errors.New("failed to create sts assume role credentials")
		}
	case cfg.SessionToken != "":
		// Use temporary credentials with session token
		creds = credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken)
	default:
		// Use standard access/secret key
		creds = credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	}

	// Create MinIO client opts
	opts := &minio.Options{
		Creds:  creds,
		Secure: cfg.Secure,
		Region: cfg.Region,
	}
	// Configure transport with TLS if enabled
	if cfg.EnableTLS {
		var tlsConfig *tls.Config
		var transport *http.Transport
		if tlsConfig, err = util.BuildTLSConfig(cfg.CertFile, cfg.KeyFile, cfg.CAFile, cfg.InsecureSkipVerify); err != nil {
			return nil, errors.Wrap(err, "failed to build TLS config")
		}
		if transport, err = minio.DefaultTransport(cfg.Secure); err != nil {
			return nil, errors.Wrap(err, "failed to create transport")
		}
		transport.TLSClientConfig = tlsConfig
		opts.Transport = transport
	}

	// Create the client
	if cli, err = minio.New(cfg.Endpoint, opts); err != nil {
		return nil, errors.Wrap(err, "failed to create minio client")
	}
	if cfg.Trace {
		cli.TraceOn(os.Stdout)
	}
	return cli, nil
}

// Client returns the global MinIO client.
// It returns nil if the client is not initialized.
func Client() *minio.Client {
	mu.RLock()
	defer mu.RUnlock()
	return client
}

func Put(reader io.Reader, size int64) (filename string, err error) {
	region := config.App.Minio.Region
	bucket := config.App.Minio.Bucket
	if len(region) > 0 {
		err = client.MakeBucket(context.TODO(), bucket, minio.MakeBucketOptions{Region: region})
	} else {
		err = client.MakeBucket(context.TODO(), bucket, minio.MakeBucketOptions{})
	}
	if err != nil {
		exists, errBucketExists := client.BucketExists(context.TODO(), config.App.Minio.Bucket)
		if errBucketExists == nil && exists {
			goto CONTINUE
		}
		return filename, err
	}
CONTINUE:
	filename = util.UUID()
	_, err = client.PutObject(context.TODO(), bucket, filename, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return filename, err
}

func Get(filename string) ([]byte, error) {
	object, err := client.GetObject(context.TODO(), config.App.Minio.Bucket, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, object); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EnsureBucket ensures that the bucket exists, creates it if not exists.
func EnsureBucket(ctx context.Context, buckets ...string) error {
	for _, bucket := range buckets {
		bucket = strings.TrimSpace(bucket)
		if len(bucket) == 0 {
			continue
		}
		if err := ensureBucket(ctx, bucket); err != nil {
			return err
		}
	}
	return nil
}

// ensureBucket ensures that the bucket exists, creates it if not exists.
func ensureBucket(ctx context.Context, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: config.App.Minio.Region}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}
