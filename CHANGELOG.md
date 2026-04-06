# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Godoc comments on all exported types, functions, and constants.
- `ym.Ptr[T]` generic helper for optional pointer fields in request structs.
- `ym.ValidateRecipient` shared validation for ChatID/Login recipient parameters.
- Root-level and package-level `doc.go` files with usage examples.
- `CHANGELOG.md` (this file).
- `Makefile` with `test`, `lint`, `release-patch`, and `release-minor` targets.
- GitHub Actions release workflow (`.github/workflows/release.yml`).
- Architecture section in README with package layout diagram.
- Versioning section in README.

### Changed
- Fixed incorrect import paths in README code examples (`config` -> `ymerrors`).
- Fixed aggregator references in README (`sdk.ClientSet` -> `client.YMClient`).
- Replaced `map[string]interface{}` with `map[string]any` in middleware.
- Translated `sdk.go` comments from Russian to English godoc.

### Removed
- Duplicated `validateRecipient` functions in messages and polls packages
  (replaced by shared `ym.ValidateRecipient`).
