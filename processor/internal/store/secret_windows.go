//go:build windows

package store

// Windows: the pairing token is protected with DPAPI (CryptProtectData,
// current-user scope). Only this Windows user on this computer can unprotect
// it; a copied file or another Windows account can't read it. Programs running
// as the same user still can — an accepted limit for this alpha.
//
// Compiled and cross-built here; NOT yet observed on a real Windows machine.

import (
	"errors"
	"syscall"
	"unsafe"
)

const SecretScheme = "dpapi-current-user"

var (
	crypt32          = syscall.NewLazyDLL("crypt32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procProtect      = crypt32.NewProc("CryptProtectData")
	procUnprotect    = crypt32.NewProc("CryptUnprotectData")
	procLocalFree    = kernel32.NewProc("LocalFree")
	entropy          = []byte("Arameair/Thought-Processor pairing v1")
	errDPAPIFailed   = errors.New("Windows could not protect the pairing (DPAPI)")
	errDPAPIUnreadab = errors.New("Windows could not unprotect the pairing (made by another Windows user or computer?)")
)

const cryptprotectUIForbidden = 0x1

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func blob(b []byte) *dataBlob {
	if len(b) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{cbData: uint32(len(b)), pbData: &b[0]}
}

func (d *dataBlob) bytes() []byte {
	out := make([]byte, d.cbData)
	copy(out, unsafe.Slice(d.pbData, d.cbData))
	return out
}

func Protect(plain []byte) ([]byte, error) {
	var out dataBlob
	r, _, _ := procProtect.Call(uintptr(unsafe.Pointer(blob(plain))), 0, uintptr(unsafe.Pointer(blob(entropy))), 0, 0,
		cryptprotectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, errDPAPIFailed
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return out.bytes(), nil
}

func Unprotect(sealed []byte) ([]byte, error) {
	var out dataBlob
	r, _, _ := procUnprotect.Call(uintptr(unsafe.Pointer(blob(sealed))), 0, uintptr(unsafe.Pointer(blob(entropy))), 0, 0,
		cryptprotectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, errDPAPIUnreadab
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return out.bytes(), nil
}
