### 规则

1. 在编写任何代码前，先描述你的方法并等待批准
2. 如果我给出的需求模糊，请在编写代码前提出澄清问题
3. 完成任何代码编写后，列出边缘案例并建议覆盖它们的测试用例
4. 如果任务需要修改超过3个文件，先停止并将其拆分成更小的任务
5. 出现bug时，先编写能重现该bug的测试，再修复直到测试通过
6. 每次我纠正你时，反思你做错了什么，并制定永不再犯的计划
7. 如果需求没有完全理解，请继续向我提问，直到完全清楚需求了才开始写代码！

### 代码规范

- 代码风格、测试用例风格、程序输出风格等必须和当前包中保持一致.
- 修改完代码后, 需要同时检查下相关的文档/注释是否需要更新.
- 代码新增功能、bug修复完后, 需要更新其对应的函数、结构体、接口、类的代码注释和文档、代码注释请使用英文
- module, internal/model 中使用自定义 Request 时, 命名偏向于 XXXReq, 例如 SignupReq，Response 命名偏向于 XXXRsp, 例如 SignupRsp
- 总是按照最佳实践方式来实现代码、代码注释需要符合 golang 规范；新需求代码需要有足够的注释；如果发现现有注释有问题或不符合代码逻辑也需要优化注释。
- 对于不直接走框架默认 CRUD 资源模型的自定义接口，service 的 model 优先使用空模型；不同接口之间禁止复用 REQ、RSP，即使字段完全相同，也必须为当前接口单独定义自己的 REQ、RSP。
- 代码中的结构体、变量、常量、类型别名等定义需要按功能归类摆放；同一功能域的内容必须相邻放在一起，必要时使用空行分隔不同功能组，避免同类定义被无关内容打散。



### moudle 开发规范

开发 module 时，每个接口对应的【model/REQ/RSP】、【业务逻辑】必须写在自己对应的单独代码文件中，禁止将多个接口的【model/REQ/RSP】写在同一个 model 代码文件中，禁止将多个不同接口的【业务逻辑】写在同一个 service 代码文件中。三种场景如下：

- 完全不同的业务逻辑和接口：/api/users，/api/groups，那么需要两个 model 文件和两个 service 文件
- 同一资源对象则走框架提供的 curd：POST /api/configs、DELETE /api/configs/:id、 DELETE /api/configs、PUT /api/configs/:id、PATCH /api/configs/:id、GET /api/configs、GET /api/configs/:id，只需要一个 model 文件且 model 文件中没有自定义 REQ 和 RSP，service 文件中只有一个结构体，在结构体上加上不同的 hooks。
- 同一资源对象走自定义业务逻辑：GET /api/iam/sessions、DELETE /api/iam/sessions/:id。还是只需要一个 model 文件和一个 service 文件，但是都有自己的 REQ、RSP service结构体：
  - model 代码文件中的结构体：`SessionsListReq`、`SessionsListRsp`、`SessionsDeleteReq`、`SessionsDeleteRsp`。
  - service 结构体方法：
    `func (s *SessionsListService) List(ctx *types.ServiceContext, req *modeliamsession.SessionsListReq) (rsp *modeliamsession.SessionsListRsp, err error)`、
    `func (s *SessionsDeleteService) Delete(ctx *types.ServiceContext, req *modeliamsession.SessionsDeleteReq) (rsp *modeliamsession.SessionsDeleteRsp, err error)`

module 包中的接口测试用例规范：

- 测试文件名名要符合子 moudle 名，例如 module/iam/session_test.go 就是专门用来存放 session 相关接口的测试用例，其对应的接口实现放在 internal/{model,service}/session 目录中。
- 测试组织方式要改成一个接口对应一个顶层测试函数，各个顶层测试函数应该尽量避免相互影响。
- 如果同一个接口有多种场景，则在这个接口对应的测试函数里 用 t.Run(...) 做子测试，如果只有一个场景，则不需要额外使用 t.Run(...) 来运行子测试。
- 测试用到的辅助函数应该放在其对应的测试文件中，例如 session 子模块相关的测试辅助函数应该放在 session_test.go 中，account 子模块相关的测试辅助函数应该放在 account_test.go 中。并且测试用例使用到的辅助函数尽量放在顶层测试函数之后。



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



### commit 

如果我让你给出 git commit 建议，你给出的 git commit 建议必须符合如下规则：

- 需要根据代码变更内容给出一个或多个 git commit，当修改的代码内容涉及多个主题时则需要多个 git commit。
- 如果给出多个 git commit，则需要提供每个 commit 对应的代码文件。
- commit 必须符合 conventional commit 规范
- 只查看暂存区的代码变更。
- commit 内容必须是英文。
- 给出的 commit 内容的 title 和 body 需要放在一起，方便我直接复制



### 开发完后

- 必须执行 `make check` 确保代码检查能通过
