package bootstrap

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/forbearing/gst/authn/jwt"
	"github.com/forbearing/gst/authz/rbac/basic"
	"github.com/forbearing/gst/authz/rbac/tenant"
	"github.com/forbearing/gst/cache"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/controller"
	"github.com/forbearing/gst/cronjob"
	"github.com/forbearing/gst/database/clickhouse"
	"github.com/forbearing/gst/database/helper"
	"github.com/forbearing/gst/database/mysql"
	"github.com/forbearing/gst/database/postgres"
	"github.com/forbearing/gst/database/sqlite"
	"github.com/forbearing/gst/database/sqlserver"
	"github.com/forbearing/gst/debug/gops"
	"github.com/forbearing/gst/debug/pprof"
	"github.com/forbearing/gst/debug/statsviz"
	"github.com/forbearing/gst/grpc"
	"github.com/forbearing/gst/logger/logrus"
	pkgzap "github.com/forbearing/gst/logger/zap"
	"github.com/forbearing/gst/metrics"
	"github.com/forbearing/gst/middleware"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/provider/cassandra"
	"github.com/forbearing/gst/provider/elastic"
	"github.com/forbearing/gst/provider/etcd"
	"github.com/forbearing/gst/provider/feishu"
	"github.com/forbearing/gst/provider/influxdb"
	"github.com/forbearing/gst/provider/kafka"
	"github.com/forbearing/gst/provider/ldap"
	"github.com/forbearing/gst/provider/memcached"
	"github.com/forbearing/gst/provider/minio"
	"github.com/forbearing/gst/provider/mongo"
	"github.com/forbearing/gst/provider/mqtt"
	"github.com/forbearing/gst/provider/nats"
	"github.com/forbearing/gst/provider/otel"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/provider/rethinkdb"
	"github.com/forbearing/gst/provider/rocketmq"
	"github.com/forbearing/gst/router"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/task" // nolint:staticcheck
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

var (
	initialized bool
	mu          sync.Mutex
)

func Bootstrap() error {
	_, _ = maxprocs.Set(maxprocs.Logger(pkgzap.New("").Infof))

	mu.Lock()
	defer mu.Unlock()
	if initialized {
		return nil
	}

	Register(
		config.Init,
		pkgzap.Init,
		logrus.Init,
		metrics.Init,

		// cache
		cache.Init,

		// database
		sqlite.Init,
		postgres.Init,
		mysql.Init,
		clickhouse.Init,
		sqlserver.Init,
	)
	if err := Init(); err != nil {
		return err
	}
	// Wait for all database table and records to be created.
	helper.Wait()

	Register(
		// provider
		redis.Init,
		otel.Init,
		elastic.Init,
		mongo.Init,
		minio.Init,
		nats.Init,
		mqtt.Init,
		kafka.Init,
		etcd.Init,
		nats.Init,
		cassandra.Init,
		influxdb.Init,
		memcached.Init,
		rethinkdb.Init,
		rocketmq.Init,
		feishu.Init,
		ldap.Init,

		// Authorization and Authentication
		basic.Init,
		tenant.Init,
		jwt.Init,

		// service
		service.Init,

		controller.Init,
		middleware.Init,
		router.Init,
		grpc.Init,

		// job
		task.Init, // nolint:staticcheck
		cronjob.Init,

		// module system must be the last to be initialized.
		module.Init,
	)

	RegisterCleanup(redis.Close)
	RegisterCleanup(kafka.Close)
	RegisterCleanup(etcd.Close)
	RegisterCleanup(nats.Close)
	RegisterCleanup(cassandra.Close)
	RegisterCleanup(influxdb.Close)
	RegisterCleanup(memcached.Close)
	RegisterCleanup(rethinkdb.Close)
	RegisterCleanup(rocketmq.Close)
	RegisterCleanup(ldap.Close)
	RegisterCleanup(controller.Clean)
	RegisterCleanup(pkgzap.Clean)
	RegisterCleanup(config.Clean)

	initialized = true

	return Init()
}

func Run() error {
	defer Cleanup()

	// Module system may register model after initialization.
	// Wait for all tables and records registered by module system to be created.
	helper.Wait()

	RegisterGo(
		router.Run,
		grpc.Run,
		statsviz.Run,
		pprof.Run,
		gops.Run,
	)

	RegisterCleanup(router.Stop)
	RegisterCleanup(grpc.Stop)
	RegisterCleanup(statsviz.Stop)
	RegisterCleanup(pprof.Stop)
	RegisterCleanup(gops.Stop)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	errCh := make(chan error, 1)

	go func() {
		errCh <- Go()
	}()
	select {
	case sig := <-sigCh:
		zap.S().Infow("canceled by signal", "signal", sig)
		return nil
	case err := <-errCh:
		return err
	}
}
