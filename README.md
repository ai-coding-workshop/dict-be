# dict-be

A CLI tool that uses LLM providers to translate text between languages.

## Features
- Translate text with automatic language detection.
- Support OpenAI-compatible, Anthropic-compatible, and Gemini LLMs.
- Streamed output for long responses.
- Configurable via YAML file, env vars, or flags.

## Requirements
- Go 1.22

## Install
Build from source:

```shell
go build -o dict-be ./cmd/dict-be
```

Run without building:

```shell
go run ./cmd/dict-be
```

## Quick start
```shell
./dict-be query "你好，世界"
```

## Commands
- `query [text...]`: translate text between languages.
- `llm chat`: send a chat completion request.
- `llm test`: test LLM connectivity.
- `version`: print build version.

### Query options
- `-F, --file`: read query from file, use `-F-` for stdin.
- `--in, --input-language`: input language (default `auto`).
- `--out, --output-language`: output language (default `auto`).
- `--stream`: stream response.
- `--no-stream`: disable streaming response.

### LLM options
Common flags for `llm chat` and `llm test`:
- `--model`: override model name.
- `--url`: override base URL.
- `--token`: override access token.
- `--stream`: stream response.
- `--no-stream`: disable streaming response.

## Configuration
Default config file is `~/.dict-be.yml`. You can override it with
`--config /path/to/config.yml`.

Example config:
```yaml
llm:
  type: openai
  url: https://api.openai.com/v1
  model: gpt-4o-mini
  token: ${OPENAI_API_KEY}
```

Supported `llm.type` values:
- `openai`
- `anthropics`
- `gemini`

### Environment variables
All config keys can be set with the `DICT_BE_` prefix.
For example:
- `DICT_BE_LLM_TYPE`
- `DICT_BE_LLM_URL`
- `DICT_BE_LLM_MODEL`
- `DICT_BE_LLM_TOKEN`

## Examples
Translate with explicit languages:
```shell
./dict-be query --in English --out "Simplified Chinese" "hello world"
```

Translate from a file:
```shell
./dict-be query -F ./input.txt
```

Stream a translation:
```shell
./dict-be query --stream "long text here"
```

Send a direct chat prompt:
```shell
echo "ping" | ./dict-be llm chat --model gpt-4o-mini
```

Test LLM connectivity:
```shell
./dict-be llm test --stream
```

## Development
Run tests:
```shell
go test ./...
```

Inject version at build time:
```shell
go build -ldflags "-X dict-be/internal/version.Version=1.0.0" ./cmd/dict-be
```
