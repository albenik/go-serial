# Changelog

## v2.5.0

* `GOOS=android` build supported [#29](https://github.com/albenik/go-serial/issues/29)
* Unused second argument for unix build in mathod `Port.SetTimeoutEx()` was made optional in backward compatibility
  manner.
* `go 1.13` errors supported: `PortError.Unwrap()` method added, `PortError.Cause()` method marked as deprecated.

## 2.4.0

* `GOOS=darwin GOARCH=arm64` build supported [#25](https://github.com/albenik/go-serial/pull/25).
* Fixed regression in `GOOS=darwin` build was introduced in `v2.3.0`

## v2.3.0

* Some fixes backported from https://github.com/bugst/go-serial [#22](https://github.com/albenik/go-serial/pull/22).

## v2.2.0

* `PortError.Cause()` method added

## v2.1.0

* MacOS extended baudrate support added [#14](https://github.com/albenik/go-serial/pull/14).
* MacOS wrong generated syscall fixed [#15](https://github.com/albenik/go-serial/issues/15).

## v2.0.0

* New Go Module import path `github.com/albenik/go-serial/v2`
* `serial.Port` interface discarded in favor of `serial.Port` structure (similar to `os.File`)
* `serial.Mode` discared and replaced with `serial.Option`
* `serial.Open()` method changed to use `serila.Option`)
* `port.SetMode(mode *Mode)` replaced with `port.Reconfigure(opts ...Option)`
* `Disable HUPCL by default` [#7](https://github.com/albenik/go-serial/pull/7)
* `WithHUPCL(bool)` option introduced
* Minor bugfix & refactoring

## v1.x.x

* Forked from https://github.com/bugst/go-serial
* Minor but incompatible interface & logic changes implemented
* Import path altered
