### 代码风格

- 代码风格、测试用例风格、程序输出风格等必须和当前包中保持一致.
- 修改完代码后, 需要同时检查下相关的文档/注释是否需要更新.
- 不要处理以 `.bak `结尾的文件
- golang 中, 表示任意类型请使用 any 而不是 interface{}
- 代码新增功能、bug修复完后, 需要更新其对应的函数、结构体、接口、类的代码注释和文档
- module 中使用自定义 Request 时, 命名偏向于 XXXReq, 例如 SignupReq
- module 中使用自定义 Response 时, 命名偏向于 XXXRsp, 例如 SignupRsp

### Sandbox

sandbox 中切换目录时，会使用 `builtin cd` 而不是 `cd`

### 开发中

- 修改完 `cmd/gg`、`internal/codegen`、`dsl` 包的代码后，需要及时安装最新版本的 `gg` 命令：

```
go install ./cmd/gg
```

- 优先使用的包: 错误处理使用 `github.com/cockroachdb/errors `而不是 golang 内置的 errors 包.

- 禁止在 golib 项目根目录执行 `cmd/gg` 的任何命令\*\* - 很容易破坏当前项目代码。测试 `cmd/gg` 命令请到 `examples/demo` 项目目录下执行。

- 修改了代码，当前代码如果有测试用例，必须确保测试用例通过

### 开发完后

- 必须执行 `make check` 确保代码检查能通过
- 必须执行 `make test` 确保代码能测试通过 `make test`, 如果没有修改相关代码则不用执行这一步骤
