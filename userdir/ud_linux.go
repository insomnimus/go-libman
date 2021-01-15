// +build linux
package userdir

func GetDataHome() string{
	if userSet != ""{
		return userSet
	}
	return os.Getenv("HOME") + "/.local"
}