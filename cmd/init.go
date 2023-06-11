package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	headersJsonFile = "headers.json"
	postSubDir      = "_posts"
)

type fileInfo struct {
	FileName string            `json:"filename"`
	Header   map[string]string `json:"header"`
}

type headersInfo struct {
	Posts []fileInfo `json:"posts"`
	Pages []fileInfo `json:"pages"`
}

func copyDir(srcDir, dstDir string) error {
	items, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	if err = os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	for _, item := range items {
		srcPath := path.Join(srcDir, item.Name())
		dstPath := path.Join(dstDir, item.Name())
		if item.IsDir() {
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err = io.Copy(dstFile, srcFile); err != nil {
			return err
		}
	}
	return nil
}

func strTrim(str string) string {
	return strings.Trim(str, " \t\n\r\"")
}

func copyArticleAndGetHeader(srcPostPath, dstPostPath string) (map[string]string, error) {
	srcPostFile, err := os.Open(srcPostPath)
	if err != nil {
		return nil, err
	}
	defer srcPostFile.Close()

	reader := bufio.NewReader(srcPostFile)
	flag := false
	attributes := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err.Error() == "EOF" {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(line, "---") && !flag {
			flag = true
		} else if strings.HasPrefix(line, "---") && flag {
			break
		} else if flag {
			splits := strings.SplitN(line, ":", 2)
			if len(splits) != 2 {
				return nil, fmt.Errorf("parse header failed")
			}
			attributes[strTrim(splits[0])] = strTrim(splits[1])
		}
	}

	dstPostFile, err := os.OpenFile(dstPostPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer dstPostFile.Close()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				dstPostFile.WriteString(line)
				break
			} else {
				return nil, err
			}
		}
		dstPostFile.WriteString(line)
	}
	return attributes, nil
}

func checkDestinationDir(dstDir string, force bool) error {
	items, err := os.ReadDir(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(dstDir, 0755); err != nil {
				return err
			}
			fmt.Printf("warning - destination directory %s is not exist "+
				"and it will be created\n", dstDir)
		}
	}
	if len(items) > 0 {
		if !force {
			return fmt.Errorf("destination directory %s is not empty, "+
				"if you want to cover it, with --force or -f", dstDir)
		} else {
			for _, item := range items {
				if err = os.RemoveAll(
					path.Join(dstDir, item.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkSourceDir(srcDir string) error {
	_, err := os.ReadDir(srcDir)
	return err
}

func handlePosts(srcDir, dstDir string) ([]fileInfo, error) {
	srcPostDir := path.Join(srcDir, postSubDir)
	dstPostDir := path.Join(dstDir, postSubDir)

	postItems, err := os.ReadDir(srcPostDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if err = os.MkdirAll(dstPostDir, 0755); err != nil {
		return nil, err
	}

	filesInfo := make([]fileInfo, 0)

	for _, postItem := range postItems {
		srcPostPath := path.Join(srcPostDir, postItem.Name())
		dstPostPath := path.Join(dstPostDir, postItem.Name())

		if s, err := os.Stat(srcPostPath); err == nil && s.IsDir() {
			continue
		}

		header, err := copyArticleAndGetHeader(srcPostPath, dstPostPath)
		if err != nil {
			fmt.Printf("warning - init post \"%v\" failed, "+
				"you may need handle it manual\n", srcPostPath)
			continue
		}

		filenameWithoutSuffix := strings.Split(postItem.Name(), ".")[0]
		srcPostAssetDir := path.Join(srcPostDir, filenameWithoutSuffix)
		dstPostAssetDir := path.Join(dstPostDir, filenameWithoutSuffix)

		if err = copyDir(srcPostAssetDir, dstPostAssetDir); err != nil {
			fmt.Printf("warning -  copy post \"%v\" assset dir failed, "+
				"you may need handle it manual\n", srcPostPath)
			continue
		}
		filesInfo = append(filesInfo, fileInfo{postItem.Name(), header})
	}
	return filesInfo, nil
}

func handlePages(srcDir, dstDir string) ([]fileInfo, error) {
	filesInfo := make([]fileInfo, 0)
	items, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if !item.IsDir() || item.Name() == postSubDir {
			continue
		}
		srcIndexDir := path.Join(srcDir, item.Name())
		srcIndexPath := path.Join(srcIndexDir, "index.md")
		if _, err := os.Stat(srcIndexPath); err != nil {
			fmt.Printf("warning - \"%s\" is not a vaild page dir, "+
				"beacuse of %v\n", srcIndexDir, err)
			continue
		}
		dstIndexDir := path.Join(dstDir, item.Name())
		dstIndexPath := path.Join(dstIndexDir, "index.md")
		if err = os.MkdirAll(dstIndexDir, 0755); err != nil {
			return nil, err
		}
		if header, err := copyArticleAndGetHeader(srcIndexPath, dstIndexPath); err != nil {
			fmt.Printf("warning - init page \"%v\" failed, because of %v, "+
				"you may need handle it manual\n", srcIndexDir, err)
			continue
		} else {
			filesInfo = append(filesInfo, fileInfo{item.Name(), header})
		}
	}
	return filesInfo, nil
}

func initSource(ctx *cli.Context) error {
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

	if headers.Posts, err = handlePosts(srcDir, dstDir); err != nil {
		return err
	}
	if headers.Pages, err = handlePages(srcDir, dstDir); err != nil {
		return err
	}

	if jsonBytes, err := json.MarshalIndent(headers, "", "    "); err != nil {
		return nil
	} else {
		return os.WriteFile(
			path.Join(dstDir, headersJsonFile), jsonBytes, 0644)
	}
}

func cmdInit() *cli.Command {
	return &cli.Command{
		Name:   "init",
		Action: initSource,
		Usage:  "initialize original hexo/source to the custom architecture",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "source",
				Aliases:  []string{"s"},
				Usage:    "the original hexo/source dir for initializing",
				Required: true,
			},
			&cli.PathFlag{
				Name:     "destination",
				Aliases:  []string{"d"},
				Usage:    "the destination custom/source dir for initializing",
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
