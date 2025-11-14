[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/forbearing/gst)

## Install gst code generator

```bash
go install github.com/forbearing/gst/cmd/gg@latest
```

## Description

🚀 Golang Lightning Backend Framework

WARNING: Library under active development - expect significant API changes.

## Examples

1.  [basic usage example](./examples/simple/main.go)

2.  [example external project](https://github.com/forbearing/glpi)



## Documents

-   [Router usage](./examples/demo/main.go)
-   [Model usage](./examples/demo/model/user.go)
-   [Database Create](./examples/demo/controller/user_create.go)
-   [Database Delete](./examples/demo/controller/user_delete.go)
-   [Database Update](./examples/demo/controller/user_update.go)
-   [Database List](./examples/demo/controller/user_list.go)
-   [Database Get](./examples/demo/controller/user_get.go)
-   [Controller usage](./controller/READMD.md)
-   [Service usage](./examples/demo/service/user.go)
-   [Client usage](./client/client_test.go)
-   [tunnel usage](./tunnel/session_test.go)
-   Package usage
    -   lru
    -   cmap
    -   sqlite,postgres,mysql
    -   redis
    -   elastic
    -   mongo
    -   minio
    -   mqtt
    -   task



## Data Structure

-   [list](./ds/list)
    -   [arraylist](./ds/list/arraylist/list.go)
    -   [linkedlist](./ds/list/linkedlist/list.go)
    -   [skiplist](./ds/list/skiplist/skiplist.go)
-   [stack](./ds/stack)
    -   [arraystack](./ds/stack/arraystack/stack.go)
    -   [linkedstack](./ds/stack/linkedstack/stack.go)
-   [queue](./ds/queue)
    -   [arrayqueue](./ds/queue/arrayqueue/queue.go)
    -   [linkedqueue](./ds/queue/linkedqueue/queue.go)
    -   [priorityqueue](./ds/queue/priorityqueue/queue.go)
    -   [circularbuffer](./ds/queue/circularbuffer/circularbuffer.go)
-   [tree](./ds/tree)
    -   [rbtree](./ds/tree/rbtree/rbtree.go)
    -   [avltree](./ds/tree/avltree/avltree.go)
    -   [splaytree](./ds/tree/splaytree/splaytree.go)
    -   [trie](./ds/tree/trie/trie.go)
-   [heap](./ds/heap)
    -   [binaryheap](./ds/heap/binaryheap/binaryheap.go)
-   [mapset](./ds/mapset/set.go)
-   [multimap](./ds/multimap/multimap.go)


## Interface

### Logger

```go
type StandardLogger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

type StructuredLogger interface {
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)
}
type ZapLogger interface {
	Debugz(msg string, fields ...zap.Field)
	Infoz(msg string, fields ...zap.Field)
	Warnz(msg string, feilds ...zap.Field)
	Errorz(msg string, fields ...zap.Field)
	Fatalz(msg string, fields ...zap.Field)
}

type Logger interface {
	With(fields ...string) Logger

	WithObject(name string, obj zapcore.ObjectMarshaler) Logger
	WithArray(name string, arr zapcore.ArrayMarshaler) Logger

	WithControllerContext(*ControllerContext, consts.Phase) Logger
	WithServiceContext(*ServiceContext, consts.Phase) Logger
	WithDatabaseContext(*DatabaseContext, consts.Phase) Logger

	StandardLogger
	StructuredLogger
	ZapLogger
}
```

### Database

```go
type Database[M Model] interface {
	Create(objs ...M) error
	Delete(objs ...M) error
	Update(objs ...M) error
	UpdateById(id string, key string, value any) error
	List(dest *[]M, cache ...*[]byte) error
	Get(dest M, id string, cache ...*[]byte) error
	First(dest M, cache ...*[]byte) error
	Last(dest M, cache ...*[]byte) error
	Take(dest M, cache ...*[]byte) error
	Count(*int64) error
	Cleanup() error
	Health() error
	TransactionFunc(fn func(tx any) error) error

	DatabaseOption[M]
}

type DatabaseOption[M Model] interface {
	WithDB(any) Database[M]
	WithTable(name string) Database[M]
	WithDebug() Database[M]
	WithQuery(query M, config ...QueryConfig) Database[M]
	WithCursor(string, bool, ...string) Database[M]
	WithTimeRange(columnName string, startTime time.Time, endTime time.Time) Database[M]
	WithSelect(columns ...string) Database[M]
	WithSelectRaw(query any, args ...any) Database[M]
	WithIndex(indexName string, hint ...consts.IndexHintMode) Database[M]
	WithJoinRaw(query string, args ...any) Database[M]
	WithLock(mode ...consts.LockMode) Database[M]
	WithBatchSize(size int) Database[M]
	WithPagination(page, size int) Database[M]
	WithLimit(limit int) Database[M]
	WithExclude(map[string][]any) Database[M]
	WithOrder(order string) Database[M]
	WithExpand(expand []string, order ...string) Database[M]
	WithPurge(...bool) Database[M]
	WithCache(...bool) Database[M]
	WithOmit(...string) Database[M]
	WithDryRun() Database[M]
	WithoutHook() Database[M]
}
```

### Modal

```go
type Model interface {
	GetTableName() string
	GetID() string
	SetID(id ...string)
	ClearID()
	GetCreatedBy() string
	GetUpdatedBy() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	SetCreatedBy(string)
	SetUpdatedBy(string)
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	Expands() []string
	Excludes() map[string][]any
	Purge() bool
	MarshalLogObject(zapcore.ObjectEncoder) error

	CreateBefore(*ModelContext) error
	CreateAfter(*ModelContext) error
	DeleteBefore(*ModelContext) error
	DeleteAfter(*ModelContext) error
	UpdateBefore(*ModelContext) error
	UpdateAfter(*ModelContext) error
	ListBefore(*ModelContext) error
	ListAfter(*ModelContext) error
	GetBefore(*ModelContext) error
	GetAfter(*ModelContext) error
}
```

### Service

```go
type Service[M Model, REQ Request, RSP Response] interface {
	Create(*ServiceContext, REQ) (RSP, error)
	Delete(*ServiceContext, REQ) (RSP, error)
	Update(*ServiceContext, REQ) (RSP, error)
	Patch(*ServiceContext, REQ) (RSP, error)
	List(*ServiceContext, REQ) (RSP, error)
	Get(*ServiceContext, REQ) (RSP, error)

	CreateMany(*ServiceContext, REQ) (RSP, error)
	DeleteMany(*ServiceContext, REQ) (RSP, error)
	UpdateMany(*ServiceContext, REQ) (RSP, error)
	PatchMany(*ServiceContext, REQ) (RSP, error)

	CreateBefore(*ServiceContext, M) error
	CreateAfter(*ServiceContext, M) error
	DeleteBefore(*ServiceContext, M) error
	DeleteAfter(*ServiceContext, M) error
	UpdateBefore(*ServiceContext, M) error
	UpdateAfter(*ServiceContext, M) error
	PatchBefore(*ServiceContext, M) error
	PatchAfter(*ServiceContext, M) error
	ListBefore(*ServiceContext, *[]M) error
	ListAfter(*ServiceContext, *[]M) error
	GetBefore(*ServiceContext, M) error
	GetAfter(*ServiceContext, M) error

	CreateManyBefore(*ServiceContext, ...M) error
	CreateManyAfter(*ServiceContext, ...M) error
	DeleteManyBefore(*ServiceContext, ...M) error
	DeleteManyAfter(*ServiceContext, ...M) error
	UpdateManyBefore(*ServiceContext, ...M) error
	UpdateManyAfter(*ServiceContext, ...M) error
	PatchManyBefore(*ServiceContext, ...M) error
	PatchManyAfter(*ServiceContext, ...M) error

	Import(*ServiceContext, io.Reader) ([]M, error)
	Export(*ServiceContext, ...M) ([]byte, error)

	Filter(*ServiceContext, M) M
	FilterRaw(*ServiceContext) string

	Logger
}
```

### RBAC

```go
type RBAC interface {
    AddRole(name string) error
    RemoveRole(name string) error

    GrantPermission(role string, resource string, action string) error
    RevokePermission(role string, resource string, action string) error

    AssignRole(subject string, role string) error
    UnassignRole(subject string, role string) error
}
```

Disabled mode behavior:

- When RBAC is disabled or not initialized, the framework returns a safe no-op RBAC implementation. All RBAC operations succeed without side effects, preventing panics and allowing normal data operations.

### Cache

```go
type Cache[T any] interface {
	Get(key string) (T, error)
	Peek(key string) (T, error)
	Set(key string, value T, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) bool
	Len() int
	Clear()
	WithContext(ctx context.Context) Cache[T]
}

type DistributedCache[T any] interface {
	Cache[T]
	SetWithSync(key string, value T, localTTL time.Duration, remoteTTL time.Duration) error
	GetWithSync(key string, localTTL time.Duration) (T, error)
	DeleteWithSync(key string) error
}
```

### Module

```go
type Module[M Model, REQ Request, RSP Response] interface {
	Service() Service[M, REQ, RSP]
	Pub() bool
	Route() string
	Param() string
}
```

### ESDocumenter

```go
type ESDocumenter interface {
	Document() map[string]any
	GetID() string
}
```
