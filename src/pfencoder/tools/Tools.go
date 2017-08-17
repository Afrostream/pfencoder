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

func Before(value string, a string) string {
    // Get substring before a string.
    pos := strings.Index(value, a)
    if pos == -1 {
        return ""
    }
    return value[0:pos]
}