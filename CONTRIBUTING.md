# Contributing to ymsdk

[Русская версия](#как-внести-вклад-в-ymsdk)

Thank you for your interest in contributing to ymsdk! This document provides guidelines for contributing.

## Getting Started

1. Fork and clone the repository
2. Install Go 1.25+ and [golangci-lint](https://golangci-lint.run/)
3. Run tests: `go test ./...`
4. Run linter: `golangci-lint run --config .golangci.yml`

## Development Workflow

1. Create a feature branch from `master`
2. Make your changes with tests
3. Ensure all tests pass and linter is clean
4. Submit a pull request

## Code Standards

- Follow existing code patterns and architecture
- All public APIs must have doc comments
- Table-driven tests are preferred
- Use `internal/testutil.FakeDoer` for HTTP mocking in tests
- Line length limit: 180 characters
- Import order: stdlib → external → `github.com/rekurt/` packages

## Pull Request Guidelines

- Keep PRs focused on a single change
- Include tests for new functionality
- Update documentation if adding public API
- Reference related issues in the PR description

## Reporting Issues

Use GitHub Issues with the provided templates for bug reports and feature requests.

---

# Как внести вклад в ymsdk

Спасибо за интерес к проекту! Ниже описаны правила для контрибьюторов.

## Начало работы

1. Форкните и клонируйте репозиторий
2. Установите Go 1.25+ и [golangci-lint](https://golangci-lint.run/)
3. Запустите тесты: `go test ./...`
4. Запустите линтер: `golangci-lint run --config .golangci.yml`

## Рабочий процесс

1. Создайте feature-ветку от `master`
2. Внесите изменения с тестами
3. Убедитесь, что тесты проходят и линтер чист
4. Отправьте pull request

## Стандарты кода

- Следуйте существующим паттернам и архитектуре
- Все публичные API должны иметь doc-комментарии
- Предпочтительны table-driven тесты
- Используйте `internal/testutil.FakeDoer` для мокирования HTTP в тестах
- Ограничение длины строки: 180 символов
- Порядок импортов: stdlib → external → `github.com/rekurt/`

## Правила Pull Request

- PR должен быть сфокусирован на одном изменении
- Включайте тесты для новой функциональности
- Обновляйте документацию при добавлении публичного API
- Ссылайтесь на связанные issues в описании PR

## Сообщения об ошибках

Используйте GitHub Issues с предоставленными шаблонами для баг-репортов и запросов функциональности.
