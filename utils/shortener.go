package utils

import (
	"encoding/base64"
	"fmt"
	"time"
)

func GetShortCode() string {
	fmt.Println("Shortening URL")
	ts := time.Now().UnixNano()
	fmt.Println("Timestamp: ", ts)

	ts_bytes := []byte(fmt.Sprintf("%d", ts))
	key := base64.StdEncoding.EncodeToString(ts_bytes)
	fmt.Println("Key: ", key)

	key = key[:len(key)-2]

	return key[16:]
}
