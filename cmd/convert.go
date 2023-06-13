package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

// file should be open and closed externally when this method is called
func writeHeader(file *os.File, header map[string]string) error {
	file.WriteString("---\n")
	for k, v := range header {
		if _, err :=
			file.WriteString(fmt.Sprintf("%v: %v\n", k, v)); err != nil {
			return err
		}
	}
	file.WriteString("---\n")
	return nil
}

func writeContent(dstFile *os.File, srcFilePath string) error {
	srcContent, err := os.ReadFile(srcFilePath)
	if err != nil {
		return err
	}
	if _, err = dstFile.Write(srcContent); err != nil {
		return err
	}
	return nil
}

func convertHandlePosts(srcDir, dstDir string, finfos []fileInfo) error {
	srcPostsDir := path.Join(srcDir, postSubDir)
	dstPostsDir := path.Join(dstDir, postSubDir)
	for _, finfo := range finfos {
		srcPostPath := path.Join(srcPostsDir, finfo.FileName)
		dstPostPath := path.Join(dstPostsDir, finfo.FileName)
		if err := os.MkdirAll(dstPostsDir, 0755); err != nil {
			return err
		}
		dstPostFile, err := os.Create(dstPostPath)
		if err != nil {
			return err
		}
		defer dstPostFile.Close()

		if err = writeHeader(dstPostFile, finfo.Header); err != nil {
			return err
		}
		if err = writeContent(dstPostFile, srcPostPath); err != nil {
			return err
		}
	}
	return nil
}

func convertHandlePages(srcDir, dstDir string, finfos []fileInfo) error {
	for _, finfo := range finfos {
		srcPageDir := path.Join(srcDir, finfo.FileName)
		dstPageDir := path.Join(dstDir, finfo.FileName)
		if err := os.MkdirAll(dstPageDir, 0755); err != nil {
			return err
		}
		srcPagePath := path.Join(srcPageDir, "index.md")
		dstPagePath := path.Join(dstPageDir, "index.md")
		dstPageFile, err := os.Create(dstPagePath)
		if err != nil {
			return err
		}
		defer dstPageFile.Close()

		if err := writeHeader(dstPageFile, finfo.Header); err != nil {
			return err
		}
		if err := writeContent(dstPageFile, srcPagePath); err != nil {
			return err
		}
	}
	return nil
}

func convert(ctx *cli.Context) error {
	var (
		srcDir  = ctx.Path("source")
		dstDir  = ctx.Path("destination")
		force   = ctx.Bool("force")
		headers = headersInfo{}
		err     error
	)

	if err = checkDestinationDir(dstDir, force); err != nil {
		return err
	}
	if err = checkSourceDir(srcDir); err != nil {
		return err
	}

	headersJsonPath := path.Join(srcDir, headersJsonFile)
	if data, err := os.ReadFile(headersJsonPath); err != nil {
		return err
	} else if err = json.Unmarshal(data, &headers); err != nil {
		return err
	}

	if err = convertHandlePosts(srcDir, dstDir, headers.Posts); err != nil {
		return err
	}
	if err = convertHandlePages(srcDir, dstDir, headers.Pages); err != nil {
		return err
	}

	return nil
}

func cmdConvert() *cli.Command {
	return &cli.Command{
		Name:   "convert",
		Action: convert,
		Usage:  "convert to original hexo/source from the custom architecture",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Usage: "the custom source dir " +
					"that convert to original hexo/source from",
				Required: true,
			},
			&cli.PathFlag{
				Name:     "destination",
				Aliases:  []string{"d"},
				Usage:    "the destination original source dir for converting",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "if true, the destination directory will be covered",
			},
		},
	}
}
