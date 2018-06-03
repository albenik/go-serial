# go-serial

A cross-platform serial library for go-lang based on [github.com/bugst/go-serial](https://github.com/bugst/go-serial) and  edited for use under own import path.

## Documentation and examples

See the godoc here: https://godoc.org/github.com/albenik/go-serial

## What's new in v1

There are some API improvements, in particular object naming is now more idiomatic, class names are less redundant (for example `serial.SerialPort` is now called `serial.Port`), some internal class fields, constants or enumerations are now private and some methods have been moved into the proper interface.

If you come from the version v0 and want to see the full list of API changes, please check this pull request:

https://github.com/bugst/go-serial/pull/5/files

## License

The software is release under a BSD 3-clause license

https://github.com/bugst/go-serial/blob/v1/LICENSE

