package tools

import (
	"log"
	"strings"
)

func LogOnError(err error, format string, v ...interface{}) {
	format = format + ": %s"
	if err != nil {
		log.Printf(format, v, err)
	}
}

func After(value string, a string) string {
    // Get substring before a string.
    pos := strings.LastIndex(value, a)
    if pos == -1 {
        return ""
    }
    return value[(pos + 1):len(value)]
}