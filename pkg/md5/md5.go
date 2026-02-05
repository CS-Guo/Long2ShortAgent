package md5

import (
	"crypto/md5"
	"encoding/hex"
)

func Sum(data []byte) string {
	md5 := md5.New()
	md5.Write(data)
	return hex.EncodeToString(md5.Sum(nil))
}
