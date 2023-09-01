# github.com/bclswl0827/go-serial/v2

![Go](https://github.com/albenik/go-serial/workflows/Go/badge.svg)

A cross-platform serial library for Go. Forked from [github.com/bugst/go-serial](https://github.com/bugst/go-serial) and
now developing independently.

Many ideas are bein taken from [github.com/bugst/go-serial](https://github.com/bugst/go-serial)
and [github.com/pyserial/pyserial](https://github.com/pyserial/pyserial).

Any PR-s are welcome.

## INSTALL

Not work in GOPATH mode!!!

```
go get -u github.com/bclswl0827/go-serial/v2
```

## MacOS build note

* Since version **v2.1.0** the macos build requires `IOKit` as dependency and is only possible on Mac with cgo enabled.
* Apple M1 (darwin/arm64) is supported. _(Thanks to [martinhpedersen](https://github.com/albenik/go-serial/pull/25))_

## Documentation and examples

See the godoc here: https://pkg.go.dev/github.com/bclswl0827/go-serial/v2

## License

The software is release under a BSD 3-clause license
