package consts

import "os"

func getEnvVarOrDefault(key, fallback string) string {
	if value, exist := os.LookupEnv(key); exist {
		return value
	}
	return fallback
}
