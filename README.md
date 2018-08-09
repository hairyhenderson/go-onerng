⚠️ **Warning:** _This code is grossly incomplete and under-tested! Don't use it yet, except to hack on it._

# go-onerng

This is a Go port of the OneRNG tools distributed at https://onerng.info/. Much credit is due to the OneRNG creators - all I'm doing here is porting a bunch of Bash and Python code to Go.

## Roadmap

This is still fairly immature. Here's what I want to be able to do with it:

- [x] print the version (`cmdv`)
- [x] print the ID (`cmdI`)
- [x] verify the image (`cmdX` & verify PGP signature)
- [ ] generate some amount of entropy
- [ ] add extra AES128-whitening
- [ ] run as a daemon and integrate with `rngd`
