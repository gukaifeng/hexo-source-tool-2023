package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

func removeExt(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func ExecCommand(srcDir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var err error

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = srcDir

	if err = cmd.Run(); err != nil || stderr.String() != "" {
		return "", fmt.Errorf("%s %s", err.Error(), stderr.String())
	}

	return stdout.String(), nil
}

// file should be open and closed externally when this method is called
func writeHeader(srcDir, srcFilePath string, file *os.File,
	header map[string]string, autofill bool) error {

	file.WriteString("---\n")

	if autofill {
		if _, ok := header["title"]; !ok {
			fileStat, err := file.Stat()
			if err != nil {
				return err
			}
			header["title"] = strconv.Quote(removeExt(fileStat.Name()))
		}

		commitTime, err := ExecCommand(srcDir, "git", "log",
			"--pretty=format:%ad", "--date=format:\"%Y-%m-%d %H:%M:%S\"",
			srcFilePath)
		if err != nil {
			return err
		}
		commitTimeList := strings.Split(commitTime, "\n")

		if _, ok := header["date"]; !ok {
			header["data"] = commitTimeList[len(commitTimeList)-1]
		}
		if _, ok := header["updated"]; !ok {
			header["updated"] = commitTimeList[0]
		}

	}

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

func convertHandlePosts(
	srcDir, dstDir string, finfos []fileInfo, autofill bool) error {
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

		if err = writeHeader(srcDir, srcPostPath, dstPostFile, finfo.Header,
			autofill); err != nil {
			return err
		}
		if err = writeContent(dstPostFile, srcPostPath); err != nil {
			return err
		}
	}
	return nil
}

func convertHandlePages(
	srcDir, dstDir string, finfos []fileInfo, autofill bool) error {
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

		if err := writeHeader(srcDir, srcPagePath, dstPageFile, finfo.Header,
			autofill); err != nil {
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
		srcDir   = ctx.Path("source")
		dstDir   = ctx.Path("destination")
		force    = ctx.Bool("force")
		autofill = ctx.Bool("autofill")
		headers  = headersInfo{}
		err      error
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

	if err = convertHandlePosts(
		srcDir, dstDir, headers.Posts, autofill); err != nil {
		return err
	}
	if err = convertHandlePages(
		srcDir, dstDir, headers.Pages, autofill); err != nil {
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
			&cli.BoolFlag{
				Name: "autofill",
				Usage: "if true, missing fields in the article's header are " +
					"automatically fetched from its git information and " +
					"currently include `title`, `date`, and `updated` fields, " +
					" use --autofill=false to disable this",
				Value: true,
			},
		},
	}
}
