### 规则

1. 在编写任何代码前，先描述你的方法并等待批准
2. 如果我给出的需求模糊，请在编写代码前提出澄清问题
3. 完成任何代码编写后，列出边缘案例并建议覆盖它们的测试用例
4. 如果任务需要修改超过3个文件，先停止并将其拆分成更小的任务
5. 出现bug时，先编写能重现该bug的测试，再修复直到测试通过
6. 每次我纠正你时，反思你做错了什么，并制定永不再犯的计划
7. 如果需求没有完全理解，请继续向我提问，直到完全清楚需求了才开始写代码！

### 代码风格

- 代码风格、测试用例风格、程序输出风格等必须和当前包中保持一致.
- 修改完代码后, 需要同时检查下相关的文档/注释是否需要更新.
- 代码新增功能、bug修复完后, 需要更新其对应的函数、结构体、接口、类的代码注释和文档、代码注释请使用英文
- module, internal/model 中使用自定义 Request 时, 命名偏向于 XXXReq, 例如 SignupReq，Response 命名偏向于 XXXRsp, 例如 SignupRsp
- 开发 module 时，每个接口对应的【model/REQ/RSP】、【业务逻辑】必须写在自己对应的单独代码文件中，禁止将多个接口的【model/REQ/RSP】写在同一个 model 代码文件中，禁止将多个不同接口的【业务逻辑】写在同一个 service 代码文件中。举例如下：
  - 假设有接口 /api/users，/api/groups，那么需要两个 model 文件和 service 文件
  - 假设有接口 POST /api/configs、DELETE /api/configs/:id、 DELETE /api/configs、PUT /api/configs/:id、PATCH /api/configs/:id、GET /api/configs、GET /api/configs/:id，依然是一个 model 文件和一个 service 文件，因为这是同一个资源对象，只需要在一个 service 结构体的不同方法中实现业务逻辑。

- 总是按照最佳实践方式来实现代码、代码注释需要符合 golang 规范；新需求代码需要有足够的注释；如果发现现有注释有问题或不符合代码逻辑也需要优化注释。

### Sandbox

sandbox 中切换目录时，必须使用 `builtin cd` 而不是 `cd`

### 开发中

- 修改完 `cmd/gg`、`internal/codegen`、`dsl` 包的代码后，需要及时安装最新版本的 `gg` 命令：

```
go install ./cmd/gg
```

- 优先使用的包: 错误处理使用 `github.com/cockroachdb/errors `而不是 golang 内置的 errors 包.
- 禁止在 golib 项目根目录执行 `cmd/gg` 的任何命令\*\* - 很容易破坏当前项目代码。测试 `cmd/gg` 命令请到 `examples/demo` 项目目录下执行。
- 修改了代码，当前代码如果有测试用例，必须确保测试用例通过
- internal/model 不需要使用 dsl 来定义接口行为
- 如果我让你给出 git commit 建议，你给出的 git commit 建议必须符合如下规则：

  - 需要根据代码变更内容给出一个或多个 git commit，当修改的代码内容涉及多个主题时则需要多个 git commit。
  - 如果给出多个 git commit，则需要提供每个 commit 对应的代码文件。
  - commit 必须符合 conventional commit 规范
  - 只查看暂存区的代码变更。
  - commit 内容必须是英文。
  - 给出的 commit 内容的 title 和 body 需要放在一起，方便我直接复制

  

### 开发完后

- 必须执行 `make check` 确保代码检查能通过
