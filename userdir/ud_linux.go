// +build linux
package userdir

import "os"

func GetDataHome() string {
	if userSet != "" {
		return userSet
	}
	return os.Getenv("HOME") + "/.local"
}

func GetConfigHome() string{
	if userConfig != ""{
		return userConfig
	}
	return os.Getenv("HOME") + "/.config"
}