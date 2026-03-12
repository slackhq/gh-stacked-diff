package interactive

import (
	"github.com/charmbracelet/x/term"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"
)

func InteractiveEnabled(appConfig util.AppConfig) bool {
	inFile, isInFile := appConfig.Io.In.(term.File)
	var isInTerminal bool
	if !isInFile {
		isInTerminal = false
	} else {
		isInTerminal = term.IsTerminal(inFile.Fd())
	}
	outFile, isOutFile := appConfig.Io.Out.(term.File)
	var isOutTerminal bool
	if !isOutFile {
		isOutTerminal = false
	} else {
		isOutTerminal = term.IsTerminal(outFile.Fd())
	}
	return (isInTerminal && isOutTerminal) || (!isInTerminal && !isOutTerminal)
}
