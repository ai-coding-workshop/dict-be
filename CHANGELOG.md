# Changelog

All notable changes to this project will be documented in this file.

## v0.2.0 - 2026-02-05

### Features
- Add config file support.
- Add query command.
- Add OpenAI-compatible LLM client.
- Add Anthropic-compatible LLM support.
- Add Gemini LLM client.
- Add language flags to query.
- Auto-detect query languages.
- Add query translation prompts.
- Add streaming output to query.

### Maintenance
- Remove duplicate query prompts.

## v0.1.1 - 2026-02-05

### Features
- Bootstrap CLI skeleton.
- Migrate CLI to cobra and viper.
- Add release builds for Pages downloads.
- Allow deploy on v tags.
- Move Pages static assets into repo.

### Infrastructure
- Add GitHub Pages deploy workflow.
- Add Go CI workflow.

### Maintenance
- Add husky hooks and fix CLI init.
- Add Makefile for dev commands.
- Add MIT License.
