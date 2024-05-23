package dotenv

import (
	"os"
	"strconv"
)

// GetString read env variable or use a fallback string value
func GetString(variable string, fallback string) string {
	res := os.Getenv(variable)
	if res != "" {
		return res
	}

	return fallback
}

// GetInt read env variable or use a fallback integer value
func GetInt(variable string, fallback int) int {
	res := os.Getenv(variable)
	if res != "" {
		resInt, err := strconv.Atoi(res)
		if err == nil {
			return resInt
		}
	}

	return fallback
}

// GetBool read env variable or use a fallback boolean value
func GetBool(variable string, fallback bool) bool {
	res := os.Getenv(variable)
	if res != "" {
		resBool, err := strconv.ParseBool(res)
		if err == nil {
			return resBool
		}
	}

	return fallback
}