package npm

import (
	git "app/git"
	lg "app/lg"
	nd "app/output"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/machinebox/graphql"

	maps "golang.org/x/exp/maps"
)

type Connect_npm struct {
	Package            string
	Version            string
	Maintainers        int
	Contributors       int
	License            string
	Dependencies       int
	DependencyVersions []string
	DevDeps            int
	Releases           int
	TestScript         bool
	Commits            int
	Downloads          int
	URL                string
	Homepage           string
	CommitFreq         float64
	ReleaseFreq        float64
	RepoLink           string
}

type Package struct {
	AnalyzedAt string `json:"analyzedAt"`
	Collected  struct {
		Metadata struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
			Maintainers []struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"maintainers"`
			Contributors []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"contributors"`
			Repository struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"repository"`
			Links struct {
				Npm        string `json:"npm"`
				Homepage   string `json:"homepage"`
				Repository string `json:"repository"`
				Bugs       string `json:"bugs"`
			} `json:"links"`
			License         string            `json:"license"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
			Releases        []struct {
				From  string `json:"from"`
				To    string `json:"to"`
				Count int    `json:"count"`
			} `json:"releases"`
			HasTestScript bool `json:"hasTestScript"`
		} `json:"metadata"`
		NPM struct {
			Downloads []struct {
				From  string `json:"from"`
				To    string `json:"to"`
				Count int    `json:"count"`
			} `json:"downloads"`
			StarsCount int `json:"starsCount"`
		} `json:"npm"`
		Github struct {
			Homepage         string `json:"homepage"`
			StarsCount       int    `json:"starsCount"`
			ForksCount       int    `json:"forksCount"`
			SubscribersCount int    `json:"subscribersCount"`
			Contributors     []struct {
				Username     string `json:"username"`
				CommitsCount int    `json:"commitsCount"`
			} `json:"contributors"`
		} `json:"github"`
	} `json:"collected"`
	Evaluation struct {
		Quality     map[string]string `json:"quality"`
		Popularity  map[string]string `json:"popularity"`
		Maintenance struct {
			ReleaseFreq float64 `json:"releasesFrequency"`
			CommitFreq  float64 `json:"commitsFrequency"`
		} `json:"maintenance"`
	} `json:"evaluation"`
}

func (cn Connect_npm) Data(packageName string) string {
	cn.URL = packageName

	// The following makes an API call to NPM site and recieves JSON response.
	res1 := strings.Split(packageName, "/")
	packageName = res1[len(res1)-1]
	resp, err := http.Get(fmt.Sprintf("https://api.npms.io/v2/package/%s", packageName))
	if err != nil {
		lg.ErrorLogger.Println("Unable to reach package through RESTFUL API in npm.go : ", packageName)
		return ""
	}
	defer resp.Body.Close()

	// Marshallig JSON response onto the required struct
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		lg.ErrorLogger.Println("Unable to marshal JSON response to struct in npm.go : ", body)
		panic(err)
		return ""
	}

	// Package type variable that hold unmarshalled JSON response
	var pkg Package

	// Unmarshal or Decode the JSON to the interface.
	json.Unmarshal([]byte(body), &pkg)

	cn.Package = pkg.Collected.Metadata.Name
	lg.InfoLogger.Println("Setting package name : ", cn.Package)
	cn.Version = pkg.Collected.Metadata.Version
	lg.InfoLogger.Println("Setting Version : ", cn.Version)
	cn.Maintainers = len(pkg.Collected.Metadata.Maintainers)
	lg.InfoLogger.Println("Setting Maintainer: ", cn.Maintainers)
	if len(pkg.Collected.Metadata.Contributors) > 0 {
		cn.Contributors = len(pkg.Collected.Metadata.Contributors)
	} else {
		cn.Contributors = len(pkg.Collected.Github.Contributors)
	}
	lg.InfoLogger.Println("Setting Contributors: ", cn.Contributors)
	cn.License = pkg.Collected.Metadata.License
	lg.InfoLogger.Println("Setting License: ", cn.License)
	cn.Dependencies = len(pkg.Collected.Metadata.Dependencies)
	lg.InfoLogger.Println("Setting Dependenices: ", cn.Dependencies)
	cn.DependencyVersions = maps.Values(pkg.Collected.Metadata.Dependencies)
	lg.InfoLogger.Println("Setting Dependency Versions: ", cn.DependencyVersions)
	cn.DevDeps = len(pkg.Collected.Metadata.DevDependencies)
	lg.InfoLogger.Println("Setting DevDeps: ", cn.DevDeps)
	cn.Releases = len(pkg.Collected.Metadata.Releases)
	lg.InfoLogger.Println("Setting Releases: ", cn.Releases)
	cn.TestScript = pkg.Collected.Metadata.HasTestScript
	lg.InfoLogger.Println("Setting TestScript: ", cn.TestScript)
	cn.Commits = 0
	for _, s := range pkg.Collected.Github.Contributors {
		cn.Commits += s.CommitsCount
	}
	lg.InfoLogger.Println("Setting Commits: ", cn.Commits)
	cn.Downloads = 0
	for _, s := range pkg.Collected.NPM.Downloads {
		cn.Downloads += s.Count
	}
	lg.InfoLogger.Println("Setting Downloads: ", cn.Downloads)
	cn.Homepage = pkg.Collected.Metadata.Links.Repository

	cn.ReleaseFreq = pkg.Evaluation.Maintenance.ReleaseFreq
	cn.CommitFreq = pkg.Evaluation.Maintenance.CommitFreq

	cn.RepoLink = pkg.Collected.Metadata.Repository.URL

	return cn.RepoLink
}

func (cn Connect_npm) Score() *nd.NdJson {

	if cn.get_License_score() == 0.0 {
		res := git.Clone(cn.Homepage)
		if res {
			cn.License = "MIT"
		}

	}
	responsiveness := cn.get_responsivnesss()
	busFactor := cn.get_bus_factor()
	license := cn.get_License_score()
	rampup := cn.get_rampup_score()
	correctness := cn.get_correctness()
	dependencyVersions := cn.getDependencyVersionsScore()
	engineeeringProcess := cn.getEngineeringProcessScore()
	overallScore := 0.05*responsiveness + 0.1*busFactor + 0.25*license + 0.1*rampup + 0.05*correctness + 0.25*dependencyVersions + 0.2*engineeeringProcess
	overallScore = math.Min(1.0, overallScore)
	nd := new(nd.NdJson)
	nd = nd.DataToNd(cn.URL, overallScore, rampup, busFactor, responsiveness, correctness, license, dependencyVersions, engineeeringProcess)
	return nd
}

func Contains(sl []string, name string) bool {
	for _, value := range sl {
		if value == name {
			return true
		}
	}
	return false
}

func (cn Connect_npm) get_License_score() float64 {
	cmpLicenses := []string{"Public Domain", "MIT", "X11", "BSD-new", "Apache 2.0", "LGPLv2.1", "LGPLv2.1+", "LGPLv3", "LGPLv3+"}

	if Contains(cmpLicenses, cn.License) {
		return 1.0
	}

	return 0.0

}

func (cn Connect_npm) get_rampup_score() float64 {
	score := float64(cn.DevDeps) / float64(cn.Dependencies)
	return math.Min(1.0, score)
}

func (cn Connect_npm) get_bus_factor() float64 {
	if cn.Contributors == 0 {
		return 0.0
	}
	score := 1.0 - 1.0/float64(cn.Contributors)
	return math.Min(1.0, score)
}

func (cn Connect_npm) get_correctness() float64 {
	str1 := cn.Version
	res1 := strings.Split(str1, ".")
	major, _ := strconv.ParseFloat(res1[0], 64)
	minor, _ := strconv.ParseFloat(res1[1], 64)
	patch, _ := strconv.ParseFloat(res1[2], 64)

	denominator := 1.0
	if major != 0 {
		denominator = major
	}
	if minor != 0 {
		denominator *= minor
	}
	if patch != 0 {
		denominator *= patch
	}

	score := 1 - ((major + minor + patch) / (denominator + 1))
	return float64(score)
}

func (cn Connect_npm) get_responsivnesss() float64 {
	// rf:=roundFloat(cn.ReleaseFreq,2)
	// cf:=roundFloat(cn.CommitFreq,2)
	score := float64(cn.Releases) / float64(cn.Commits)
	return math.Min(1.0, score)
}

func (cn Connect_npm) getDependencyVersionsScore() float64 {
	// x := cn.Dependencies
	dependencyVersionList := cn.DependencyVersions

	numDependencies := len(dependencyVersionList)
	numPinned := 0.0
	for i := 0; i < numDependencies; i++ {
		currDependency := dependencyVersionList[i]
		//If no ~ or ^, then exact version is pinned
		if (!strings.Contains(currDependency, "~")) && (!strings.Contains(currDependency, "^")) {
			numPinned += 1.0
			//If ~ and atleast one . (major and minor specified)
		} else if strings.Contains(currDependency, "~") && strings.Contains(currDependency, ".") {
			numPinned += 1.0
			//if ^ and major version is 0 (we find 0 before . and no other num before 0)
		} else if strings.Contains(currDependency, "^") {
			zeroInd := strings.Index(currDependency, "0")
			dotInd := strings.Index(currDependency, ".")
			if (zeroInd != -1) && (zeroInd < dotInd) {
				numPinned += 1.0
			}
		}
	}
	score := float64(numPinned) / float64(numDependencies)

	return score
}

func (cn Connect_npm) getEngineeringProcessScore() float64 {

	GithubLink := cn.RepoLink
	parts := strings.Split(GithubLink, "/")

	packageName := parts[len(parts)-1]
	packageName = packageName[:len(packageName)-4]

	owner := parts[len(parts)-2]

	totAdditionsWithReview, totAdditions := getEngineeringProcessData(owner, packageName)

	return scoreEngineeringProcess(totAdditionsWithReview, totAdditions)
}

// Returns number of additions through code reviewed prs and number of additions through all prs
func getEngineeringProcessData(owner string, name string) (int, int) {
	var GITHUB_TOKEN = os.Getenv("GITHUB_TOKEN")
	type Data struct {
		Repository struct {
			PullRequests struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					ID        string `json:"id"`
					Additions int    `json:"additions"`
					Reviews   struct {
						TotalCount int `json:"totalCount"`
					} `json:"reviews"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
      query ($own: String!, $repo: String!)  {
        repository(owner: $own, name: $repo) {
          pullRequests(last: 100) {
			totalCount
            nodes {
              id
			  additions
			  reviews(last: 100) {
				totalCount
			  }
            }
		  }
        }
      }
    

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	graphqlRequest.Header.Set("Accept", "application/vnd.github.hawkgirl-preview+json")

	var res Data

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &res); err != nil {
		lg.ErrorLogger.Println("Unable to get engineering process data through GrpahQL API in github.go")
		fmt.Println((err))
		os.Exit(1)
	}

	totAdditions := 0
	totAdditionsWithReview := 0
	numPulls := len(res.Repository.PullRequests.Nodes)
	for i := 0; i < numPulls; i++ {
		currPullReq := res.Repository.PullRequests.Nodes[i]
		reqAddtions := currPullReq.Additions
		totAdditions += reqAddtions
		if currPullReq.Reviews.TotalCount >= 1 {
			totAdditionsWithReview += reqAddtions
		}

	}

	return totAdditionsWithReview, totAdditions

}

func scoreEngineeringProcess(totAdditionsWithReview int, totAdditions int) float64 {
	if totAdditions == 0 {
		return 0.0
	}
	return float64(totAdditionsWithReview) / float64(totAdditions)
}
