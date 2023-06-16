package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
			strings.TrimPrefix(srcFilePath, srcDir+"/"))
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
		var err error
		if k == "title" {
			_, err = file.WriteString(fmt.Sprintf("%v: %v\n", k, strconv.Quote(v)))
		} else {
			_, err = file.WriteString(fmt.Sprintf("%v: %v\n", k, v))
		}
		if err != nil {
			return err
		}
	}

	file.WriteString("---\n")

	return nil
}

func replaceNonPrintableChars(b []byte) []byte {
	// reg := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	reg := regexp.MustCompile(`[\x10|\xE5|\x86|\x99]`)
	return reg.ReplaceAll(b, []byte{})
}

func readContent(srcFilePath string) ([]byte, error) {
	b, err := os.ReadFile(srcFilePath)
	return replaceNonPrintableChars(b), err
}

func writeContent(dstFile *os.File, srcContent []byte) error {
	if _, err := dstFile.Write(srcContent); err != nil {
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

		srcContent, err := readContent(srcPostPath)
		if err != nil {
			return err
		}

		if autofill {
			if _, ok := finfo.Header["description"]; !ok {
				dsplen := int(math.Min(float64(len(srcContent)), 350))
				description := srcContent[:dsplen]
				noSpaceDsp := bytes.ReplaceAll(
					bytes.ReplaceAll(
						bytes.ReplaceAll(
							bytes.ReplaceAll(
								description, []byte("*"), []byte("")),
							[]byte("#"), []byte("")),
						[]byte("\r\n"), []byte(" ")),
					[]byte("\n"), []byte(" "))
				finfo.Header["description"] =
					strconv.Quote(
						strings.TrimSpace(string(noSpaceDsp)) + " ......")
			}
		}

		if err = writeHeader(srcDir, srcPostPath, dstPostFile, finfo.Header,
			autofill); err != nil {
			return err
		}
		if err = writeContent(dstPostFile, srcContent); err != nil {
			return err
		}

		filenameWithoutSuffix := strings.Split(finfo.FileName, ".")[0]
		srcPostAssetDir := path.Join(srcPostsDir, filenameWithoutSuffix)
		dstPostAssetDir := path.Join(dstPostsDir, filenameWithoutSuffix)

		if err = copyDir(srcPostAssetDir, dstPostAssetDir); err != nil {
			fmt.Printf("warning -  copy post \"%v\" assset dir failed, "+
				"you may need handle it manual\n", srcPostPath)
			continue
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

		srcContent, err := readContent(srcPagePath)
		if err != nil {
			return err
		}

		if err := writeHeader(srcDir, srcPagePath, dstPageFile, finfo.Header,
			autofill); err != nil {
			return err
		}
		if err := writeContent(dstPageFile, srcContent); err != nil {
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
