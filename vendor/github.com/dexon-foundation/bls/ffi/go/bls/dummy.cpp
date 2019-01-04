// This is a dummy source file which forces cgo to use the C++ linker instead
// of the default C linker. We can therefore eliminate non-portable linker
// flags such as -lstdc++, which is likely to break on FreeBSD and OpenBSD.
