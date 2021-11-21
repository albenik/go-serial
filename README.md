# github.com/albenik/go-serial/v2

![Go](https://github.com/albenik/go-serial/workflows/Go/badge.svg)

## MacOS Note

* Since version **v2.1.0** `GOOS=darwin` build requires `IOKit` as dependency and is only possible on Mac with cgo enabled.
* Apple M1 (darwin/arm64) is supported. (Thank to [martinhpedersen's](https://github.com/martinhpedersen) [PR](https://github.com/albenik/go-serial/pull/25))

## Package updated to v2 version

A cross-platform serial library for Go.

Forked from [github.com/bugst/go-serial](https://github.com/bugst/go-serial) and now developing independently.

Many ideas are bein taken from [github.com/bugst/go-serial](https://github.com/bugst/go-serial)
and [github.com/pyserial/pyserial](https://github.com/pyserial/pyserial).

Any PR-s are welcome.

## INSTALL

**Not work in GOPATH mode**

```
go get -u github.com/albenik/go-serial/v2
```

**`CGO_ENABLED=1` required for MacOS build**

## Documentation and examples

See the godoc here: https://godoc.org/github.com/albenik/go-serial

## License

The software is release under a BSD 3-clause license
