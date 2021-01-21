// +build linux
package userdir

import "os"

func GetDataHome() string {
	if userSet != "" {
		return userSet
	}
	return os.Getenv("HOME") + "/.local"
}
