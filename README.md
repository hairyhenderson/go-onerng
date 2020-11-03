⚠️ **Warning:** _This code is grossly incomplete and under-tested! Don't use it yet, except to hack on it._

# go-onerng

This is an unofficial Go version of the OneRNG tools distributed at https://onerng.info/. Much credit is due to the OneRNG creators - this all started as a port of a bunch of Bash and Python code to Go.

The different commands available were discovered by reading [the firmware source code](https://github.com/OneRNG/firmware/blob/master/cdc_app.c#L346).

## Roadmap

This is still fairly immature. Here's what I want to be able to do with it:

- [x] print the version (`cmdv`)
- [x] print the ID (`cmdI`)
- [x] verify the image (`cmdX` & verify PGP signature)
- [x] generate some amount of entropy (`onerng read` command)
- [x] add extra AES128-whitening
- [ ] run as a daemon and integrate with `rngd`
