# Changelog

## v2.0.0

* New Go Module import path `github.com/albenik/go-serial/v2`
* `serial.Port` interface discared in favor of `serial.Port` structure (similar to `os.File`)
* `serial.Mode` discared and replaced with `serial.Option`
* `serial.Open()` method changed to use `serila.Option`)
* `port.SetMode(mode *Mode)` replaced with `port.Reconfigure(opts ...Option)`
* `Disable HUPCL by default` (see https://github.com/albenik/go-serial/pull/7)
* `WithHUPCL(bool)` option introduced
* Minor bugfix & refactoring

## v1.x.x

* Forked from https://github.com/bugst/go-serial
* Minor but incompatible interface & logic changes implemented
* Import path altered
