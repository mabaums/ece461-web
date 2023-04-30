package github

import (
	"context"
	"fmt"

	"strconv"
	"strings"
	"time"

	"github.com/machinebox/graphql"

	git "app/git"
	lg "app/lg"
	nd "app/output"
	"math"
	"os"
)

var GITHUB_TOKEN = os.Getenv("GITHUB_TOKEN")

// Fuction to round off numbers
func roundFloat(val float64, prec uint) float64 {
	ratio := math.Pow(10, float64(prec))
	return math.Round(val*ratio) / ratio
}

// Function to get total number of commits in a repository
func Get_com(owner string, name string) int {
	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
		query Get_commits($own: String!, $repo: String!) {
			repository(owner:$own, name:$repo) {
				defaultBranchRef {
					target {
						... on Commit {
							history {
								totalCount
							}
						}
					}
				}
			}
			
		}
    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to get commits through GrpahQL API in github.go")
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	strs := strings.SplitAfter(str, "totalCount:")
	strss := fmt.Sprint(strs[1])
	strsss := strings.Split(strss, "]")

	commits, err := strconv.Atoi(strsss[0])

	if err != nil {
		lg.ErrorLogger.Println("Unable to get round number of commits")
		return 0
	}

	lg.InfoLogger.Println("Setting number of commits : ", commits)
	return commits

}

func Get_releases(owner string, name string) int {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
		repository(name:$repo, owner: $own) {
			releases {
			totalCount
			}
		}
		}

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to reach releases through GrpahQL API in github.go")
		return 0
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "totalCount:")
	//fmt.Println(strs[1])
	parse_2 := fmt.Sprint(parse_1[1])
	parse_3 := strings.Split(parse_2, "]")

	rels, err := strconv.Atoi(parse_3[0])

	if err != nil {
		lg.ErrorLogger.Println("Unable to get round number of releases")
		os.Exit(1)
	}

	lg.InfoLogger.Println("Setting number of releases : ", rels)

	return rels

}

// score responsiveness based on time to close last 50 issues
// Avg time 0 days -> score = 1, avg time > 50 days -> score = 0
func scoreResponsiveness(owner string, repo string) float64 {

	type Data struct {
		Repository struct {
			Issues struct {
				Edges []struct {
					Node struct {
						ID        string    `json:"id"`
						CreatedAt time.Time `json:"createdAt"`
						ClosedAt  time.Time `json:"closedAt"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"issues"`
		} `json:"repository"`
	}

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
		repository(name:$repo, owner: $own) {
			issues(last:50, states:CLOSED) {
				edges {
					node {
						createdAt
						closedAt
					}
				}
			}
		}
		}

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", repo)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)

	var res Data

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &res); err != nil {
		lg.ErrorLogger.Println("Unable to get resposiveness data through GrpahQL API in github.go")
		fmt.Println((err))
		os.Exit(1)
	}

	numIssues := len(res.Repository.Issues.Edges)
	if numIssues == 0 {
		return 1.0
	}
	totHours := 0.0
	for i := 0; i < numIssues; i++ {
		currIssue := res.Repository.Issues.Edges[i].Node
		createTime := currIssue.CreatedAt
		closeTime := currIssue.ClosedAt
		totHours += (closeTime.Sub(createTime)).Hours()

	}
	avgDaysToClose := (totHours / float64(numIssues)) / 24.0
	score := 1.0 - avgDaysToClose/50.0
	return math.Max(score, 0.0)

}

func Get_assignees(owner string, name string) int {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
		repository(name:$repo, owner: $own) {
			assignableUsers {
			totalCount
			}
		}
		}

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to reach assignees through GrpahQL API in github.go")
		return 0
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "totalCount:")
	//fmt.Println(strs[1])
	parse_2 := fmt.Sprint(parse_1[1])
	parse_3 := strings.Split(parse_2, "]")

	assign, err := strconv.Atoi(parse_3[0])

	if err != nil {
		lg.ErrorLogger.Println("Unable to get round number of assignees")
		os.Exit(1)
	}

	lg.InfoLogger.Println("Setting number of assignees : ", assign)

	return assign

}

func Get_contributors(owner string, name string) int {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`

	query Get_commits($own: String!, $repo: String!){
		repository(owner:$own, name: $repo) {
			id
			name
			mentionableUsers {
			totalCount
			}
		}
		}

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to reach assignees through GrpahQL API in github.go")
		os.Exit(1)
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "totalCount:")
	//fmt.Println(strs[1])
	parse_2 := fmt.Sprint(parse_1[1])
	parse_3 := strings.Split(parse_2, "]")

	contributors, err := strconv.Atoi(parse_3[0])

	if err != nil {
		lg.ErrorLogger.Println("Unable to get round number of contributors")
		os.Exit(1)
	}

	lg.InfoLogger.Println("Setting number of contributors : ", contributors)

	return contributors

}

func scoreBusFactor(owner string, repo string) float64 {

	// assign := Get_assignees(owner, repo)
	contributors := Get_contributors("nullivex", "nodist")
	if contributors == 0 {
		return 0.0
	}
	score := 1.0 - 1.0/float64(contributors)

	if score > 1 {
		score = 1
	} else if score < 0 {
		score = 0
	}

	lg.InfoLogger.Println("Finding responsiveness score : ", score)
	return roundFloat(score, 2)

}

func get_dependencies(owner string, name string) int {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
	repository(owner:$own, name:$repo) {
		dependencyGraphManifests {
		edges {
			node {
				dependenciesCount
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
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to get depandancies through GrpahQL API in github.go")
		os.Exit(1)
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "dependenciesCount:")
	if len(parse_1) <= 1 {
		return 0
	}
	//fmt.Println(parse_1[1])
	parse_2 := fmt.Sprint(parse_1[1])
	parse_3 := strings.Split(parse_2, "]")

	dependencies, err := strconv.Atoi(parse_3[0])

	if err != nil {
		lg.ErrorLogger.Println("Unable to get round number of depandancies")
		os.Exit(1)
	}

	lg.InfoLogger.Println("Setting number of depandencies : ", dependencies, "for :", name)

	return dependencies

}

func get_devDep(owner string, name string) int {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
		repository(owner:$own, name:$repo) {
		  dependencyGraphManifests {
			edges {
			  node {
				dependencies {
				  totalCount
				  nodes {
					hasDependencies
				  }
				}
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
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to get Dev depandancies through GrpahQL API in github.go")
		os.Exit(1)
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "totalCount:")
	//fmt.Println(parse_1[0])

	parse_2 := fmt.Sprint(parse_1[0])
	devDep := strings.Count(parse_2, "true")

	lg.InfoLogger.Println("Setting number of Dev depandancies : ", devDep)

	if devDep > 0 {
		return devDep
	} else {
		return 0
	}

}

func scoreRampUp(owner string, repo string) float64 {

	dependencies := get_dependencies(owner, repo)
	devDep := get_devDep(owner, repo)

	score := float64(devDep) / float64((dependencies + 1))

	if score > 1 {
		score = 1
	} else if score < 0 {
		score = 0

	}

	lg.InfoLogger.Println("Finding RampUp score : ", score)

	return roundFloat(score, 2)

}
func get_License(owner string, name string) string {

	url := fmt.Sprintf("https://github.com/%s/%s", owner, name)
	defer func() {
		if err := recover(); err != nil {

		}
	}()
	if git.Clone(url) {
		return "present"
	}

	// graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	// graphqlRequest := graphql.NewRequest(`
	// query Get_commits($own: String!, $repo: String!){
	// 		repository(name: $repo, owner: $own) {
	// 		  licenseInfo {
	// 			name
	// 		  }

	// 	}
	// }

	// `)

	// graphqlRequest.Var("own",owner)
	// graphqlRequest.Var("repo",name)

	// graphqlRequest.Header.Set("Authorization", "Bearer " + GITHUB_TOKEN)
	// var graphqlResponse interface{}
	// if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
	// 	lg.ErrorLogger.Println("Unable to get License through GrpahQL API in github.go")
	//     os.Exit(1)
	// }

	// str := fmt.Sprint(graphqlResponse)
	// fmt.Println(str)

	// found_license := strings.Count(str, "name:")
	// fmt.Println(found_license)
	// if found_license == 0{
	// 	lg.WarningLogger.Println("No license Found")
	// 	return "No license found"
	// }

	// parse_1 := strings.SplitAfter(str, "name:")
	// //fmt.Println(parse_1[0])
	// parse_2 := fmt.Sprint(parse_1[1])
	// parse_3 := strings.Split(parse_2,"]")
	// parse_4 := fmt.Sprint(parse_3[0])
	// parse_5 := strings.Split(parse_4,"License")
	// parse_6 := fmt.Sprint(parse_5[0])
	// parse_7 := strings.Split(parse_6," ")

	// //fmt.Println(parse_5[0])

	//lg.InfoLogger.Println("License is  : ",parse_7[0])

	return "not_present"

}

func scoreLicense(owner string, repo string) float64 {

	license := get_License(owner, repo)

	if "present" == license {
		lg.InfoLogger.Println("LicenseScore  is  : ", 1)
		return 1.0
	}
	lg.InfoLogger.Println("LicenseScore is  : ", 0)
	return 0.0

}

func get_tag(owner string, name string) string {

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
	query Get_commits($own: String!, $repo: String!){
			repository(name: $repo, owner: $own) {
			  releases(last: 1) {
				nodes{
					tagName
				}
			  }

		}
	}

    `)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		lg.ErrorLogger.Println("Unable to get Repository version through GrpahQL API in github.go")
		os.Exit(1)
	}

	str := fmt.Sprint(graphqlResponse)
	//fmt.Println(str)

	parse_1 := strings.SplitAfter(str, "tagName:")
	if len(parse_1) == 1 {
		return "No version"
	}
	//fmt.Println(parse_1[1])
	parse_2 := fmt.Sprint(parse_1[1])
	parse_3 := strings.Split(parse_2, "]")
	//fmt.Println(parse_3[0])
	parse_4 := fmt.Sprint(parse_3[0])
	// parse_5 := strings.Split(parse_4,"License")
	// parse_6 := fmt.Sprint(parse_5[0])
	parse_7 := strings.Split(parse_4, " ")

	lg.InfoLogger.Println("Setting the repository version : ", parse_7[0])

	return parse_7[0]
	//fmt.Println(parse_7[0])

}

// Returns number of open issues, number of closed issues
func getIssuesData(owner string, name string) (int, int) {

	type IssuesData struct {
		Repository struct {
			Issues struct {
				TotalCount int `json:"totalCount"`
			} `json:"issues"`
		} `json:"repository"`
	}

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
      query ($own: String!, $repo: String!)  {
        repository(owner: $own, name: $repo) {
          issues(last:1, states:OPEN) {
			totalCount
		  }
        }
      }
	`)

	graphqlRequest.Var("own", owner)
	graphqlRequest.Var("repo", name)

	graphqlRequest.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	graphqlRequest.Header.Set("Accept", "application/vnd.github.hawkgirl-preview+json")

	var res IssuesData

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &res); err != nil {
		lg.ErrorLogger.Println("Unable to get open issues count through GrpahQL API in github.go")
		fmt.Println((err))
		os.Exit(1)
	}
	openIssues := res.Repository.Issues.TotalCount

	graphqlRequest2 := graphql.NewRequest(`
      query ($own: String!, $repo: String!)  {
        repository(owner: $own, name: $repo) {
          issues(last:1, states:CLOSED) {
			totalCount
		  }
        }
      }
	`)

	graphqlRequest2.Var("own", owner)
	graphqlRequest2.Var("repo", name)

	graphqlRequest2.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	graphqlRequest2.Header.Set("Accept", "application/vnd.github.hawkgirl-preview+json")

	var res2 IssuesData

	if err := graphqlClient.Run(context.Background(), graphqlRequest2, &res2); err != nil {
		lg.ErrorLogger.Println("Unable to get closed issues count through GrpahQL API in github.go")
		fmt.Println((err))
		os.Exit(1)
	}
	closedIssues := res2.Repository.Issues.TotalCount
	return openIssues, closedIssues

}

// score correctness based on number of open issues versus closed issues
func scoreCorrectness(owner string, repo string) float64 {

	openIssues, closedIssues := getIssuesData(owner, repo)
	if (openIssues + closedIssues) == 0 {
		return 1.0
	}
	score := 1 - float64(openIssues)/(float64(openIssues+closedIssues))
	lg.InfoLogger.Println("Finding Correctness score : ", float64(score))
	return float64(score)
}

// Returns a list of direct dependency versions for repo according to github.
func getDependencyVersions(owner string, name string) []string {

	type Data struct {
		Repository struct {
			DependencyGraphManifests struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					Filename string `json:"filename"`
				} `json:"nodes"`
				Edges []struct {
					Node struct {
						BlobPath     string `json:"blobPath"`
						Dependencies struct {
							TotalCount int `json:"totalCount"`
							Nodes      []struct {
								PackageName     string `json:"packageName"`
								Requirements    string `json:"requirements"`
								HasDependencies bool   `json:"hasDependencies"`
								PackageManager  string `json:"packageManager"`
							} `json:"nodes"`
						} `json:"dependencies"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"dependencyGraphManifests"`
		} `json:"repository"`
	}

	graphqlClient := graphql.NewClient("https://api.github.com/graphql")

	graphqlRequest := graphql.NewRequest(`
      query ($own: String!, $repo: String!)  {
        repository(owner: $own, name: $repo) {
          dependencyGraphManifests {
            edges {
              node {
                dependencies {
                  nodes {
                    requirements
                   
                  }
                }
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
		lg.ErrorLogger.Println("Unable to get depandancies through GrpahQL API in github.go")
		fmt.Println((err))
		os.Exit(1)
	}
	var totalDep []string
	numEdges := len(res.Repository.DependencyGraphManifests.Edges)
	for i := 0; i < numEdges; i++ {
		currEdge := res.Repository.DependencyGraphManifests.Edges[i]
		numNodes := len(currEdge.Node.Dependencies.Nodes)
		for j := 0; j < numNodes; j++ {
			totalDep = append(totalDep, currEdge.Node.Dependencies.Nodes[j].Requirements)
		}

	}

	return totalDep
}

// Returns the number of majorMinor pinned dependencies divided by the total number of dependencies.
func scoreVersionPinning(dependencyVersionList []string) float64 {
	// dependencyVersionList := getDependencyVersions(owner, repo)

	numDependencies := len(dependencyVersionList)
	if numDependencies == 0 {
		return 1.0
	}
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
	score := numPinned / float64(numDependencies)

	return score
}

// Returns number of additions through code reviewed prs and number of additions through all prs
func getEngineeringProcessData(owner string, name string) (int, int) {

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
          pullRequests(last: 100, states:MERGED) {
			totalCount
            nodes {
              id
			  additions
			  reviews(last: 1) {
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
	// fmt.Printf("name: %s, numpulls:%d, tot:%d , rev: %d\n", name, numPulls, totAdditions, totAdditionsWithReview)
	return totAdditionsWithReview, totAdditions

}

func scoreEngineeringProcess(totAdditionsWithReview int, totAdditions int) float64 {
	if totAdditions == 0 {
		return 0.0
	}
	return float64(totAdditionsWithReview) / float64(totAdditions)
}

func Score(URL string) *nd.NdJson {

	cuttingByTwo := strings.FieldsFunc(URL, func(r rune) bool {
		if r == '/' {
			return true
		}
		return false
	})

	//fmt.Println(cuttingByTwo)

	var owner string = cuttingByTwo[2]
	var repo string = cuttingByTwo[3]

	overallScore := 0.0
	rampUp := scoreRampUp(owner, repo)
	busFactor := scoreBusFactor(owner, repo)
	responsiveness := scoreResponsiveness(owner, repo)
	correctness := scoreCorrectness(owner, repo)
	license := scoreLicense(owner, repo)
	depVersionList := getDependencyVersions(owner, repo)

	versionPinning := scoreVersionPinning(depVersionList)
	totAdditionsWithReview, totAdditions := getEngineeringProcessData(owner, repo)
	engineeringProcess := scoreEngineeringProcess(totAdditionsWithReview, totAdditions)
	overallScore = 0.1*responsiveness + 0.15*busFactor + 0.20*license + 0.1*rampUp + 0.1*correctness + 0.20*versionPinning + 0.15*engineeringProcess
	lg.InfoLogger.Println("Finding overall score : ", overallScore)

	nd := new(nd.NdJson)
	nd = nd.DataToNd(URL, overallScore, rampUp, busFactor, responsiveness, correctness, license, versionPinning, engineeringProcess)

	return nd
}
