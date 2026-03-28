//go:build windows

package secretstore

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	crypt32DLL             = syscall.NewLazyDLL("Crypt32.dll")
	kernel32DLL            = syscall.NewLazyDLL("Kernel32.dll")
	procCryptProtectData   = crypt32DLL.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32DLL.NewProc("CryptUnprotectData")
	procLocalFree          = kernel32DLL.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func encryptSecretPayload(plainData []byte) ([]byte, error) {
	return protectData(plainData)
}

func decryptSecretPayload(data []byte) ([]byte, error) {
	return unprotectData(data)
}

func resolveEncryptionKey() ([]byte, error) {
	return nil, nil
}

func protectData(plainData []byte) ([]byte, error) {
	in := bytesToDataBlob(plainData)
	var out dataBlob
	result, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, fmt.Errorf("failed to encrypt secret payload with DPAPI: %w", err)
	}
	defer freeDataBlob(out)
	return dataBlobToBytes(out), nil
}

func unprotectData(data []byte) ([]byte, error) {
	in := bytesToDataBlob(data)
	var out dataBlob
	result, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, fmt.Errorf("failed to decrypt secret payload with DPAPI: %w", err)
	}
	defer freeDataBlob(out)
	return dataBlobToBytes(out), nil
}

func bytesToDataBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
}

func dataBlobToBytes(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	source := unsafe.Slice(blob.pbData, blob.cbData)
	result := make([]byte, len(source))
	copy(result, source)
	return result
}

func freeDataBlob(blob dataBlob) {
	if blob.pbData == nil {
		return
	}
	_, _, _ = procLocalFree.Call(uintptr(unsafe.Pointer(blob.pbData)))
}
