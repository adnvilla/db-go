## [2.0.0](https://github.com/adnvilla/db-go/compare/v1.0.1...v2.0.0) (2026-02-07)

### ⚠ BREAKING CHANGES

* Config.TracingAnalyticsRate changed from float64 to *float64

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>

### Bug Fixes

* resolve race conditions, connection leak, and transaction safety ([#7](https://github.com/adnvilla/db-go/issues/7)) ([dbdbbf4](https://github.com/adnvilla/db-go/commit/dbdbbf47e9ab3fcbfd6ec716c80684cc5af75121))

## [1.0.1](https://github.com/adnvilla/db-go/compare/v1.0.0...v1.0.1) (2025-10-17)

### Bug Fixes

* update logger-go and datadog-agent dependencies to latest versions ([#5](https://github.com/adnvilla/db-go/issues/5)) ([fb16f1b](https://github.com/adnvilla/db-go/commit/fb16f1b7841dbdb071d6c5ad60926908a22e3686))

## 1.0.0 (2025-10-14)

### Features

* add GitHub Actions workflows for Go build and release processes ([#4](https://github.com/adnvilla/db-go/issues/4)) ([edef478](https://github.com/adnvilla/db-go/commit/edef4786be1b0255273655d07a1b93b7d48f9fe8))
* **tracing:** add Datadog tracing support for GORM operations ([32ecfbe](https://github.com/adnvilla/db-go/commit/32ecfbe0359a44d2f1d13164960745490c4f1ef0))

### Bug Fixes

* update branch references from 'main' to 'master' in workflow ([#3](https://github.com/adnvilla/db-go/issues/3)) ([7d8f51d](https://github.com/adnvilla/db-go/commit/7d8f51d19712cdefb1494ff0dbbfc2378be66cf3))
