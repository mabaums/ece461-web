package git

import (
	"app/lg"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

func Clone(url string) bool {
	localPath := strings.Split(url, "/")[len(strings.Split(url, "/"))-1]
	_, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		panic(err)
	}
	// fmt.Printf("cloned repo: %s\n", r.Worktree())
	cmpLicenses := []string{"Public Domain", "MIT", "X11", "BSD-new", "Apache 2.0", "LGPLv2.1", "LGPLv2.1+", "LGPLv3", "LGPLv3+"}

	for _, s := range cmpLicenses {
		rt := findLicense(s, localPath)
		if rt {
			defer os.RemoveAll(localPath)
			return true
		}
	}

	return false
}

func findLicense(license string, folder string) bool {
	root := folder
	word := license
	var isFound bool
	// isfound = false
	err := filepath.Walk(root, func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileContents, err := ioutil.ReadFile(root)
			if err != nil {
				return err
			}
			isFound = strings.Contains(string(fileContents), word)
			if isFound {
				lg.InfoLogger.Println(word, " found in ", root)
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		lg.WarningLogger.Println("Error walking directory:", err)
	}

	return isFound
}
