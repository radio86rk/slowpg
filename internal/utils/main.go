package utils

import (
	"os"
)

func GetEnv(key, defl string)string{
	val, exists := os.LookupEnv(key)
	if !exists {
		val = defl
	}
	return val
}