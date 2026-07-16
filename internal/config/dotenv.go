package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv membaca file .env (jika ada) di working directory dan meng-set
// setiap key sebagai environment variable proses ini - TAPI hanya jika
// variable tersebut belum pernah diset di environment OS.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env opsional - tidak error kalau tidak ada
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = unquote(value)

		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, value)
	}
}

func unquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}