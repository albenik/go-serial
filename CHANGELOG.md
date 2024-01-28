# Changelog

## 2.7.0

- CI: Supported go versions now are `go1.19`, `go1.20`, `go1.21`
- Applied `go mod tidy -go 1.19`
- Package `github.com/albenik/go-serial/enumerator` was removed as of broken build with `go1.21`,
  please use `github.com/bugst/go-serial/enumerator` — the original well maintained source of the removed package.
- Minor code fixes (typo, linter recommendations, etc...)
- Dependencies updated

## 2.6.1

- BUGFIX: Linux, "bad address" while setting DTR (#41)

## 2.6.0

- `go mod tudy -go 1.18`.
- CI Tests: `go1.18`, `go1.19`, `go1.20`.
- CI Cross-build: cleanup.
- `golangci-lint` added & code cleaned.
- obsolete `darwin/386` code removed.

## 2.5.1

- `ppc64le` build supported [#33](https://github.com/albenik/go-serial/pull/33).

## 2.5.0

- `GOOS=android` build supported [#29](https://github.com/albenik/go-serial/issues/29).
- Unused second argument for unix build in method `Port.SetTimeoutEx()` was made optional in backward compatibility
  manner.
- `go 1.13` errors supported: `PortError.Unwrap()` method added, `PortError.Cause()` method marked as deprecated.

## 2.4.0

- `GOOS=darwin GOARCH=arm64` build supported [#25](https://github.com/albenik/go-serial/pull/25).
- Fixed regression in `GOOS=darwin` build was introduced in `v2.3.0`

## 2.3.0

- Some fixes backported from https://github.com/bugst/go-serial [#22](https://github.com/albenik/go-serial/pull/22).

## 2.2.0

- `PortError.Cause()` method added

## 2.1.0

- MacOS extended baudrate support added [#14](https://github.com/albenik/go-serial/pull/14).
- MacOS wrong generated syscall fixed [#15](https://github.com/albenik/go-serial/issues/15).

## 2.0.0

- New Go Module import path `github.com/albenik/go-serial/v2`
- `serial.Port` interface discarded in favor of `serial.Port` structure (similar to `os.File`)
- `serial.Mode` discarded and replaced with `serial.Option`
- `serial.Open()` method changed to use `serila.Option`)
- `port.SetMode(mode *Mode)` replaced with `port.Reconfigure(opts ...Option)`
- `Disable HUPCL by default` [#7](https://github.com/albenik/go-serial/pull/7)
- `WithHUPCL(bool)` option introduced
- Minor bugfix & refactoring

## 1.x.x

- Forked from https://github.com/bugst/go-serial
- Minor but incompatible interface & logic changes implemented
- Import path altered
