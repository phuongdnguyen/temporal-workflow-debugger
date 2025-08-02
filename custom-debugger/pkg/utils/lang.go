package utils

import (
	"os"
)

type Lang string

const (
	GoDAP   Lang = "go-dap"
	GoDelve Lang = "go-delve"
	Python  Lang = "python"
)

func GetLang() Lang {
	lang := Lang(os.Getenv("LANG"))
	if lang == "" {
		lang = GoDelve
	}
	return lang
}
