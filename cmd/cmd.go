package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func Main() {
	app := &cli.App{
		Name:  "hexo-source-tool",
		Usage: "a tool for reorganizing hexo/source/ directory",
		Commands: []*cli.Command{
			cmdInit(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error - %v\n", err)
	} else {
		fmt.Println("OK")
	}
}
