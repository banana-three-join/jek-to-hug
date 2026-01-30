/*
	Migration tool to migrate from Jekyll to Hugo

	TODO:
		port global variables to hugo.toml
		if an image is in any of the files, get them from static
		check all the different types of images and move them to static
		convert liquid .html to go template .html
		add execution steps
		add tests
		dry-run
		separation of concerns
		rollback
		deal with path as filepath instead of string

		.png, .svg
			it should have the same name but it should now be referenced from static instead
		.md
			copy file to migrated path or move to content

*/

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	HugoConfigFilename   = "hugo.toml"
	JekyllConfigFilename = "_config.yml"
)

var fileSeparator = string(filepath.Separator)

var specialDirs = map[string]string{
	"_sass":     "assets" + fileSeparator + "scss",
	"_data":     "data",
	"_layouts":  "layouts" + fileSeparator + "_default",
	"_includes": "layouts" + fileSeparator + "partials",
	"pages":     "content",
	"assets":    "static",
	"js":        "assets" + fileSeparator + "js",
	"css":       "assets" + fileSeparator + "css",
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Error: cli format: jektohug [ROOT_PATH]")
		os.Exit(1)
	}
	root := os.Args[1]
	if err := filepath.WalkDir(root, walk); err != nil {
		log.Fatal(err)
	}
}

func walk(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	migratedPath, err := migratePath(path)

	if err != nil {
		return err
	}

	if d.IsDir() {
		os.MkdirAll(migratedPath, 0777)
	} else {
		fileExtension := filepath.Ext(path)

		switch fileExtension {
		case ".yml":
			if err := migrateConfigFile(); err != nil {
				return err
			}

		case ".scss":
			if err := moveFile(path, migratedPath); err != nil {
				return err
			}
		case ".md":
			if err := moveFile(path, migratedPath); err != nil {
				return err
			}
		case ".png", ".svg":
			if err := moveFile(path, migratedPath); err != nil {
				return err
			}
		case ".yaml", ".json", ".toml", ".html", ".xml":
			//just copy the file (as long as it's parent dir is data)
			if err := moveFile(path, migratedPath); err != nil {
				return err
			}
		}
	}

	return nil
}

//if not in this map then it must be added to content/_dir
//get path and rewrite the path onto the edge cases and write either the dir or the file
//join with file path

// if the dir has _Xxx structure change it to Xxx?
// if the dir is wherever then /content should be added
func migratePath(path string) (string, error) {
	var migratedPath strings.Builder
	splitPath := strings.Split(path, fileSeparator)

	for i, p := range splitPath {
		if v, ok := specialDirs[p]; ok {
			_, err := migratedPath.WriteString(v)
			if err != nil {
				return "", err
			}
		} else {
			_, err := migratedPath.WriteString(p)
			if err != nil {
				return "", err
			}
		}

		if i < len(splitPath)-1 {
			_, err := migratedPath.WriteString(fileSeparator)
			if err != nil {
				return "", err
			}
		}
	}

	return migratedPath.String(), nil
}

func migrateConfigFile() error {
	/*if
	"description: ""Service mesh...""","[params]  description = ""Service mesh..."""
	*/

	hugoConfigFile, err := os.Create(HugoConfigFilename)
	if err != nil {
		return err
	}
	defer hugoConfigFile.Close()

	jekyllConfigFile, err := os.Open(JekyllConfigFilename)
	if err != nil {
		return err
	}
	defer jekyllConfigFile.Close()

	scanner := bufio.NewScanner(jekyllConfigFile)
	for scanner.Scan() {
		//fix the color #edgecase
		migratedConfig, omit := migrateConfig(scanner.Text())

		if omit {
			continue
		}

		n, err := jekyllConfigFile.Write([]byte(migratedConfig))
		if len(migratedConfig) != n {
			return errors.New("textline wasn't fully written")
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateConfig(c string) (string, bool) {
	omitConfig := func(c string) bool {
		switch c {
		case "markdown", "baseurl", "sass", "plugins", "defaults":
			return true
		default:
			return false
		}
	}

	withoutComments, _, _ := strings.Cut(c, "#")
	//check for edgecases
	config := strings.Split(withoutComments, ":")
	omit := omitConfig(config[0])

	if omit || withoutComments == "" || len(config) < 2 {
		return "", true
	}

	switch config[0] {
	case "exclude":
		config[0] = "ignoreFiles"
		escaped := strings.ReplaceAll(config[1], ".", `\\.`)
		config[1] = escaped + "$"
	case "url":
		config[0] = "BaseURL"
	}

	//any weird edgecases?
	migratedConfig := strings.Join(config, "=")
	return migratedConfig, omit
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}

	return nil
}
