[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/forbearing/gst)

## Install gst code generator

```bash
go install github.com/forbearing/gst/cmd/gg@latest
```

## 使用 gg 管理后端项目

`gg` 是 gst 的项目脚手架和代码生成工具。它的核心职责不是替代业务代码，而是根
据 `model` 中的 DSL 声明维护项目骨架、路由注册、service 注册和默认 service
action 文件，面向基于 gst 创建的业务后端项目使用。

### 创建项目

使用 `gg new` 创建一个新的 gst 后端项目：

```bash
gg new github.com/example/myapp
```

该命令会创建项目目录并完成基础初始化：

- 初始化 Go module。
- 创建 `configx`、`cronjob`、`middleware`、`model`、`service`、`module`、
  `router`、`dao`、`provider` 等基础目录和文件。
- 生成应用入口 `main.go`。
- 生成包含基础必要配置的 `config.ini.example`。
- 执行 `go mod tidy`。
- 初始化 git 仓库。

进入项目后，通常先复制并调整配置文件，再开始编写业务模型：

```bash
cd myapp
cp config.ini.example config.ini
```

如需查看框架支持的完整默认配置，可以运行：

```bash
gg config defaults --format ini
```

也可以查看某个配置段，或在 INI、JSON、TOML、YAML 之间转换配置文件：

```bash
gg config list
gg config defaults server --format yaml
gg config defaults server --format toml
gg config convert config.ini config.yaml
gg config convert config.yaml config.json
gg config convert config.json config.toml
```

### 声明模型和接口

业务项目的主要输入是 `model/**/*.go`。普通数据库资源通常嵌入 `model.Base`，
并在 `Design()` 中声明 `Migrate(true)`、`Endpoint(...)`、`Param(...)` 和需
要启用的 CRUD action。没有数据库表、只表示一个动作的接口，优先使用
`model.Empty`，并为当前接口单独定义自己的 `XXXReq`、`XXXRsp`。

示例：

```go
type User struct {
	Name string `json:"name" schema:"name"`

	model.Base
}

func (User) Design() {
	Migrate(true)
	Endpoint("users")

	Create(func() {
		Service(true)
	})
	List(func() {})
}
```

声明了 `Create`、`List`、`Get` 等 action 后，该 action 默认启用；`Enabled(false)`
主要用于显式关闭某个已经声明的 action。
`Service(true)` 表示这个 action 需要业务侧实现或扩展 service；未开启 service
的默认 CRUD action 由框架按模型声明处理。

### 生成和同步代码

修改 `model` DSL 后运行：

```bash
gg gen
```

`gg gen` 会先运行与 `gg check` 相同的约束检查，通过后再生成或同步代码：

- `main.go`
- `model/model.go`
- `service/service.go`
- `router/router.go`
- `service/**/<action>.go` 或 DSL `Filename(...)` 指定的 service action 文件

`gg gen` 会保留已有 service 文件中的业务实现，并按当前 DSL 同步 service 类型、
泛型参数、模型 import 和注册关系。它不会自动删除已经失效的旧 service 文件；
需要清理时使用 `gg prune` 或 `gg gen --prune`。

### 实现业务逻辑

生成后，业务逻辑主要写在 `service/**` 中。service 通常嵌入
`service.Base[M, REQ, RSP]`，然后实现当前 action 对应的方法或 hook。查询和写库
优先使用 `database.Database[T](ctx.DatabaseContext())`，并根据需要组合
`WithQuery`、`WithPagination`、`WithOrder`、`WithSelect` 等 database 选项。

生成文件和手写文件的边界应保持清晰：

- `main.go`、`model/model.go`、`service/service.go`、`router/router.go` 由
  `gg gen` 维护，通常不要手写修改。
- `model/**/*.go` 是接口、字段和 DSL 声明层。
- `service/**/*.go` 是业务实现层。
- `module/` 用于注册内置或自定义模块。
- `configx/`、`cronjob/`、`middleware/` 用于扩展配置、定时任务和中间件。

### 检查项目约束

提交或生成前可以单独运行：

```bash
gg check
```

`gg check` 会检查当前 gst 业务项目是否符合框架约束：

- `service` 不能调用其他业务 service。
- `dao` 和 `model` 不能调用业务 service。
- `model` 目录和模型文件应使用单数命名，文件名不要使用连字符。
- 模型结构体的 `json` tag 应使用 `snake_case`。
- 子目录中的 model package 名称应与目录名一致。
- gst 业务项目根目录只允许约定的目录结构，避免生成器无法识别的组织方式。

`gg gen` 内部也会执行这些检查；如果检查失败，会停止生成，避免继续写入不一致的
注册代码。

### 清理废弃 service 文件

当你关闭某个 action 的 `Service(true)`、删除 model、调整 `Filename(...)`，或者
重命名 model 路径后，旧的 service action 文件可能已经不再被当前 DSL 使用。可以
运行：

```bash
gg prune
```

也可以在生成时联动清理：

```bash
gg gen --prune
```

`gg prune` 会扫描当前 model 定义和已有 service 文件，列出将要删除的废弃文件，
并在删除前要求确认。它只清理标准 action 文件和识别为 action service 的文件；
不会盲目删除整个 `service` 目录。

如果某些手写 service 文件需要长期保留，可以在项目根目录添加 `.gg.yaml`：

```yaml
prune:
  ignore:
    - "service/legacy/"
    - "service/custom_.*\\.go"
```

`ignore` 支持正则匹配；无法作为正则解析时，会按字符串包含关系匹配。

### 推荐日常流程

1. 使用 `gg new <module>` 创建业务项目。
2. 在 `model/**/*.go` 中声明资源模型、动作模型和 DSL。
3. 每次修改 DSL 后运行 `gg gen`。
4. 在生成的 `service/**` 文件中实现业务逻辑和 hook。
5. 运行 `gg check` 检查项目结构和依赖边界。
6. 删除或关闭 action 后运行 `gg prune`，或使用 `gg gen --prune` 同步生成并清理。
7. 修改带 `Migrate(true)` 的数据库模型字段后，再根据实际数据库配置运行
   `gg migrate` 处理 schema 迁移。

开发过程中如果频繁修改 model，也可以使用 `gg watch` 监听 model 目录并自动执行
`gg gen`。

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

### Initializer

```go
type Initializer interface {
	Init() error
}
```

`Initializer` 用于启动阶段只执行一次的初始化组件。`Init` 在必需的配置、连接
或运行时资源初始化失败时应返回错误。

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
	Warnz(msg string, fields ...zap.Field)
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

`StandardLogger` 提供普通日志和 printf 风格日志方法；`StructuredLogger`
使用交替的 key/value 字段；`ZapLogger` 接收类型化的 `zap.Field`。`Logger.With`
和上下文辅助方法会返回携带额外结构化上下文的派生日志器。

### Database

```go
type Database[M Model] interface {
	Create(objs ...M) error
	Delete(objs ...M) error
	Update(objs ...M) error
	UpdateByID(id string, key string, value any) error
	List(dest *[]M) error
	Get(dest M, id string) error
	First(dest M) error
	Last(dest M) error
	Take(dest M) error
	Count(*int64) error
	Cleanup() error
	Health() error
	Transaction(fn func(txDB Database[M]) error) error
	TransactionFunc(fn func(tx any) error) error

	DatabaseOption[M]
}

type DatabaseOption[M Model] interface {
	WithDB(any) Database[M]
	WithTx(tx any) Database[M]
	WithTable(name string) Database[M]
	WithDebug() Database[M]
	WithQuery(query M, config ...QueryConfig) Database[M]
	WithCursor(string, bool, ...string) Database[M]
	WithTimeRange(columnName string, startTime time.Time, endTime time.Time) Database[M]
	WithSelect(columns ...string) Database[M]
	WithIndex(indexName string, hint ...consts.IndexHintMode) Database[M]
	WithRollback(rollbackFunc func()) Database[M]
	WithLock(mode ...consts.LockMode) Database[M]
	WithBatchSize(size int) Database[M]
	WithPagination(page, size int) Database[M]
	WithLimit(limit int) Database[M]
	WithOffset(offset int) Database[M]
	WithExclude(map[string][]any) Database[M]
	WithOrder(order string) Database[M]
	WithExpand(expand []string, order ...string) Database[M]
	WithPurge(...bool) Database[M]
	WithCache(...bool) Database[M]
	WithOmit(...string) Database[M]
	WithBuildSQL(statements *[]SQLStatement) Database[M]
	WithDryRun() Database[M]
	WithoutHook() Database[M]
}
```

`Database` 按模型类型划分作用域。每个独立操作都应从一次新的
`database.Database[M](ctx)` 调用开始，并以一个终止操作结束，例如 `Create`、
`List`、`Get`、`Count`、`Cleanup`、`Health`、`Transaction` 或
`TransactionFunc`。不要在无关操作之间复用同一个 database 句柄，因为底层
GORM session 可能保留子句。

重要操作语义：

- `Create` 会设置框架 ID 和时间戳，除非启用了 `WithDryRun`。
- `Delete` 使用 `WithPurge`、模型的 `Purge()` 设置，默认行为是软删除。
- `Update` 会保存完整模型值并更新时间戳，除非启用了 `WithDryRun`。
- `Cleanup` 会永久删除软删除记录；`WithDryRun().Cleanup()` 只构建 cleanup SQL。
- `Health` 仍会执行真实连接检查，不会被 `WithDryRun` 禁用。
- `Transaction` 是单模型事务辅助方法，会传入绑定事务的 `Database`。
- `TransactionFunc` 用于多模型事务；回调中使用的每个 database 句柄都必须调用 `WithTx(tx)`。

重要选项语义：

- `WithDB` 接收自定义 `*gorm.DB`，并可能自动迁移模型，除非 `WithTable` 禁用了迁移。
- `WithTable` 设置自定义表名，并禁用当前链路的自动迁移。
- `WithBuildSQL` 只构建下一次终止操作的 SQL，并把可执行的 `Query`、`Args` 与便于调试复制的 `RenderedSQL` 追加到传入的 `[]SQLStatement`。
- `WithDryRun` 只构建 SQL，不执行数据库 I/O、框架 hook、缓存变更或对象字段填充。
- 选项只作用于下一个终止操作，操作结束后会被重置。

### Model

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

`Model` 是框架中持久化资源和动作模型的契约。持久化资源通常嵌入
`model.Base`；不落库的动作模型可以使用 `model.Empty` 或 `model.Any`。
`Purge()` 控制 `Delete` 默认是否硬删除，生命周期 hook 会在对应 CRUD 阶段执行。

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

`Service` 是 controller 调用的业务操作契约。生成的 controller 会通过它执行
CRUD、批量 CRUD、生命周期 hook、导入导出、过滤和日志相关逻辑。自定义动作应
定义当前接口专用的 `REQ`/`RSP` 类型，不要复用其他 endpoint 的请求/响应类型。

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

禁用模式行为：

- 当 RBAC 被禁用或未初始化时，框架会返回安全的 no-op RBAC 实现。所有 RBAC 操作都会成功且没有副作用，避免 panic，并允许正常数据操作继续执行。
- 在 `RevokePermission(role, resource, action)` 中，空的 resource/action 参数有明确含义：它们会扩大指定角色的权限撤销范围。

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

`Cache` 在 `Get` 或 `Peek` 未命中时返回 `types.ErrEntryNotFound`。`Set` 使用
TTL，零值表示不过期。`WithContext` 返回使用指定 context 的缓存句柄，用于追踪
或取消传播。

`DistributedCache` 在 `Cache` 基础上增加显式的本地加远端同步辅助方法。

### Module

```go
type Module[M Model, REQ Request, RSP Response] interface {
	Service() Service[M, REQ, RSP]
	Route() string
	Pub() bool
	Param() string
}
```

`Module` 提供路由元数据、公开或私有访问标记、URL 参数名，以及生成 controller
使用的 service 实现。

### ESDocumenter

```go
type ESDocumenter interface {
	Document() map[string]any
	GetID() string
}
```
