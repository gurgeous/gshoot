package cli

import (
	"fmt"
	"io"
	"strconv"

	"google.golang.org/api/drive/v3"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
)

func printFiles(w io.Writer, files []*drive.File) {
	for i, file := range files {
		const width = 30
		num := ux.Dim.Render(fmt.Sprintf("%2d.", i+1))
		name := fmt.Sprintf("%-"+strconv.Itoa(width)+"s", util.Truncate(file.Name, width))
		date := ux.Dim.Render(util.DateAndTimeStr(file.ModifiedByMeTime))
		fmt.Fprintf(
			w,
			" %s %s   %s\n",
			num,
			util.Hyperlink(w, util.SpreadsheetURL(file.Id), name),
			date,
		)
	}
}
