package utils

import "unsafe"

// s2b converts string to a byte slice without memory allocation.
func String2Byte(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
