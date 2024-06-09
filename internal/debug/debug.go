package debug

import (
	"fmt"
)

var debugMode = false

func SetDebugMode(enabled bool) {
	debugMode = enabled
	if debugMode {
		fmt.Println("SYS:", "Debug mode enabled; debug statements will print to terminal")
	}
}

func IsDebugMode() bool {
	return debugMode
}

func Println(a ...any) {
	if debugMode {
		fmt.Print("DEBUG: ")
		fmt.Println(a...)
	}
}

func Printf(format string, a ...any) {
	if debugMode {
		fmt.Print("DEBUG: ")
		fmt.Printf(format, a...)
	}
}
