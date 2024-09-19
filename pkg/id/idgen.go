package id

import (
	"fmt"
	"hash/crc32"

	"github.com/api7/api7-ingress-controller/pkg/utils"
)

// GenID generates an ID according to the raw material.
func GenID(raw string) string {
	if raw == "" {
		return ""
	}
	p := utils.String2Byte(raw)

	res := crc32.ChecksumIEEE(p)
	return fmt.Sprintf("%x", res)
}
