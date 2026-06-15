package main

import (
	"os"
	"path/filepath"

	"github.com/forbearing/gst/internal/clioutput"
)

func checkErr(err error) {
	if err == nil {
		return
	}
	panic(err)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)

	var err error
	if _, err = os.Stat(dir); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	return err
}

func writeFileWithLog(filename string, content string) {
	if fileExists(filename) {
		oldData, err := os.ReadFile(filename)
		checkErr(err)
		if string(oldData) == content {
			clioutput.Item("SKIP", "%s", filename)
		} else {
			clioutput.Status(clioutput.StyleWarn, clioutput.SymbolSuccess, "UPDATE", "%s", filename)
			checkErr(os.WriteFile(filename, []byte(content), 0o600))
		}
	} else {
		clioutput.Success("CREATE", "%s", filename)
		checkErr(ensureParentDir(filename))
		checkErr(os.WriteFile(filename, []byte(content), 0o600))
	}
}
