//go:build linux
// +build linux

package userdir

import "os"

func LibmanDBDir() string {
	if dir := os.Getenv("LIBMAN_DB_PATH"); dir != "" {
		if dir[len(dir)-1] == '/' {
			return dir[len(dir)-1]
		}
		return dir
	}
	return os.Getenv("HOME") + "/.local/libman"
}

func LibmanConfigDir() string {
	if conf := os.Getenv("LIBMAN_CONFIG_PATH"); conf != "" {
		if conf[len(conf)-1] == '/' {
			return conf[:len(conf)-1]
		}
		return conf
	}
	return os.Getenv("HOME") + "/.config/libman"
}
