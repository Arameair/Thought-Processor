//go:build !windows

package store

// Non-Windows: the token is kept in a file readable only by this user (0600).

const SecretScheme = "file-permissions-0600"

func Protect(plain []byte) ([]byte, error)    { return append([]byte(nil), plain...), nil }
func Unprotect(sealed []byte) ([]byte, error) { return append([]byte(nil), sealed...), nil }
