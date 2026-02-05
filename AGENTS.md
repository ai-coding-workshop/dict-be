# AGENTS.md

本文件面向大模型/自动化代理，介绍仓库架构、开发规约与代码格式规范。

## 仓库结构与架构说明

- `cmd/dict-be/`：CLI 入口，`main.go` 负责解析入口参数并调用内部 CLI。
- `internal/cli/`：命令调度层，集中定义子命令、参数与输出。
- `internal/config/`：配置加载与校验，使用 Viper 读取文件/环境变量。
- `internal/llm/`：LLM 客户端适配层（OpenAI/Anthropic/Gemini）。
- `internal/version/`：版本信息，默认 `dev`，支持通过 `-ldflags` 注入。
- `static/`：前端静态资源（GitHub Pages）。
- `go.mod`：模块定义。

约定：
- 业务逻辑放在 `internal/` 下，避免对外暴露。
- 新增命令时在 `internal/cli` 中注册，并保持输出一致。

## 开发规约

- 入口逻辑最小化，避免在 `cmd/` 中堆叠业务逻辑。
- 模块边界清晰，避免跨层直接调用。
- 任何新功能需要有明确的命令或接口入口。
- 重要行为需要可测试（优先添加单元测试）。
- CLI 配置统一走 `internal/config`，不要在命令中直接读取环境变量。
- LLM 相关逻辑集中在 `internal/llm`，避免在命令层直接拼接请求。
- 提示词模板存放在 `internal/cli/*.md`，通过 `embed` 嵌入读取。

## 代码风格与格式

- 语言：Go 1.22。
- 使用 `gofmt` 统一格式。
- 导入顺序：标准库、第三方、项目内。
- 变量命名：使用短且清晰的驼峰命名。
- 错误处理：尽量向上返回，并附带上下文。

## 提交与 Git 约定

- 常用忽略规则位于 `.gitignore`。
- 统一换行规则位于 `.gitattributes`（LF）。
- 提交信息必须遵循 [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/)
- 提交说明使用英文。
- 好的提交说明应该包含多行，第一行是标题，第二行是空行，第三行开始是关于修改的详细描述。
- 在提交说明的详细描述中，需要解释修改原因（why），以及包含对修改的简洁描述，而不是只提供如何修改（how）的描述。
- 提交修改原因从提示词，结合代码的修改进行推断。
- 提交说明单行不超过 72 个字符，超过请折行（不要添加多余的空行）。
- 使用 HereDoc 方式运行 git 提交命令，如: `git commit -F- <<-EOF` 命令，而不要使用多个
  `-m <message>` 参数创建多行提交说明。因为多个 `-m <message>` 会导致在提交说明见插入
  冗余的空白行。

## 运行与构建

- 运行：`go run ./cmd/dict-be`
- 版本注入示例：
  `go build -ldflags "-X dict-be/internal/version.Version=1.0.0" ./cmd/dict-be`

## 对大模型的工作约束

- 改动尽量小且聚焦；避免无关重构。
- 保持现有风格一致，不引入未经请求的依赖。
- 修改后如有可能请补充或更新测试。

