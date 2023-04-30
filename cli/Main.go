package main

import (
	gh "app/github"
	log "app/lg"
	np "app/npm"
	nd "app/output"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func seperateLinks(links []string) []*nd.NdJson {
	var re = regexp.MustCompile(`(?m)github`)

	var scores []*nd.NdJson
	for _, url := range links {
		if re.MatchString(url) {
			//log.InfoLogger.Println("Github Condition in Seperate Links , Current URL: ",url)
			//fmt.Println(url)
			scores = append(scores, gh.Score(url))
		} else if strings.Contains(url, "npm") {
			//log.InfoLogger.Println("NPM Condition in Seperate Links , Current URL: ",url)

			cn := new(np.Connect_npm)

			GithubLink := cn.Data(url)
			if GithubLink == "" {
				continue
			}
			parts := strings.Split(GithubLink, "/")

			packageName := parts[len(parts)-1]
			packageName = packageName[:len(packageName)-4]

			owner := parts[len(parts)-2]
			githubURL := "https://github.com/" + owner + "/" + packageName
			if GithubLink != "" {
				scores = append(scores, gh.Score(githubURL))
			}
		}
	}
	return scores
}

func readInput(inputFile string) []string {
	readfile, err := os.Open(inputFile)
	log.ErrorLogger.Println("error in opeing file: ", inputFile)

	if err != nil {
		log.ErrorLogger.Println("error in opeing file")
		return nil
	}

	fileScanner := bufio.NewScanner(readfile)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string

	//The following read the file and adds to an array
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}
	readfile.Close()

	return fileLines
}

func main() {
	log.Init(os.Getenv("LOG_FILE"))
	inputFile := os.Args[1]

	links := readInput(inputFile)
	if links == nil {
		return
	}
	score := seperateLinks(links)
	output := nd.FormattedOutput(score)
	fmt.Println(output)

	os.Exit(0)

}
