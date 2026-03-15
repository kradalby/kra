package util

import (
	"os"
	"strconv"
)

func GetEnvString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvBool(key string, fallback bool) bool {
	if val, err := strconv.ParseBool(GetEnvString(key, "")); err == nil {
		return val
	}

	return fallback
}
