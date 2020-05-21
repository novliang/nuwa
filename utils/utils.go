package utils

import (
	"os"
	"path"
)

// file exist
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}


// Get App Path
func GetAppPath() string {
	p, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return path.Dir(p)
}

