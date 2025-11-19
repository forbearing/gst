# 项目上下文

### 项目介绍

### 代码风格

- 代码风格、测试用例风格、程序输出风格等必须和当前包中保持一致.
- 修改完代码后, 需要同时检查下相关的文档/注释是否需要更新.
- 不要处理以 `.bak `结尾的文件
- golang 中, 表示任意类型请使用 any 而不是 interface{}
- 代码新增功能、bug修复完后, 需要更新其对应的函数、结构体、接口、类的代码注释和文档
- module 中使用自定义 Request 时, 命名偏向于 XXXReq, 例如 SignupReq
- module 中使用自定义 Response 时, 命名偏向于 XXXRsp, 例如 SignupRsp



## 重要工作流程

### 代码修改后的操作

- 修改完 `cmd/gg`、`internal/codegen`、`dsl` 包的代码后，需要及时安装最新版本的 `gg` 命令：

```
go install ./cmd/gg
```

### 任务完成检查清单：

- 确保代码检查能通过 `make check`
- 确保代码能测试通过 `make test`

### 优先使用的包

- 错误处理使用 `github.com/cockroachdb/errors`

### 关键警告

- **禁止在 golib 项目根目录执行 `cmd/gg` 的任何命令** - 很容易破坏当前项目代码。测试 `cmd/gg` 命令请到 `examples/demo`  项目目录下执行。

## Shell 环境

### Fish Shell 兼容性

本项目在 fish shell 环境下开发。生成 shell 命令时：

- 避免 bash 特有的语法结构
- 切换目录时请使用如果使用到了 `cd` 命令时必须使用 `builtin cd mydir` 而不是直接 `cd mydir` 因为在 sandbox autorun 中, `cd` 命令有问题
