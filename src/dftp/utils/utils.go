package utils

import (
	"runtime"
	"strings"
)

func GetTraceback() string {
	tb := make([]byte, 4096)
	stb := string(tb[:runtime.Stack(tb, false)])
	lines := strings.Split(stb, "\n")
	for i := range lines {
		if strings.Contains(lines[i], "ServeHTTP") {
			// first lines contain boring traceback of this very function
			return strings.Join(lines[4:i], "\n")
		}
	}
	return stb
}
