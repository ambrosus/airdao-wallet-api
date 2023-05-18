package watcher

import "encoding/hex"

func HexToBytes(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return bytes
}
