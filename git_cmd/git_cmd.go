package git_cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type GitCmd struct {
	dir string
}

func NewGitCmd(dir string) *GitCmd {
	return &GitCmd{dir}
}

type GitCommit struct {
	Parent  string `json:"parent"`
	Message string `json:"message"`
}

type BacklogPR struct {
	PullRequestID int
	BaseParent    string
	BranchParent  string
	FilePaths     []string
}

type GitPullRequestCommit struct {
	GitCommit
	FilePaths []string
}

func (g *GitCmd) Exec(subcommand string, args ...string) (string, error) {
	// argsWithSub := []string{subcommand}
	// argsWithSub = append(argsWithSub, args...)

	// cmd := exec.Command("git", argsWithSub...)
	// cmd.Dir = g.dir
	out, err := g.Cmd(subcommand, args...).Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (g *GitCmd) Cmd(subcommand string, args ...string) *exec.Cmd {
	argsWithSub := []string{subcommand}
	argsWithSub = append(argsWithSub, args...)

	cmd := exec.Command("git", argsWithSub...)
	cmd.Dir = g.dir

	return cmd
}

func (g *GitCmd) GetPRMergedCommits(lastCommit string) ([]BacklogPR, error) {
	format := `{
	"parent": "%P", 
	"message": "%f"
},`

	out, err := g.Exec("log", "--pretty=format:"+format, fmt.Sprintf("%s..HEAD", lastCommit))

	if err != nil {
		return nil, err
	}
	logOut := string(out)

	if len(logOut) == 0 {
		return nil, err
	}
	logOut = logOut[:len(logOut)-1]
	logOut = fmt.Sprintf("[%s]", logOut)

	var gitCommitList []GitCommit
	err = json.Unmarshal([]byte(logOut), &gitCommitList)
	if err != nil {
		return []BacklogPR{}, err
	}

	baclogPR, err := g.getDiffFiles(filterBacklogPR(gitCommitList))

	if err != nil {
		return nil, err
	}

	return baclogPR, nil
}

var prMergeReg = regexp.MustCompile(`^Merge-pull-request-([0-9]+)-(\S+)-into-\S+`)

func (g *GitCmd) getDiffFiles(prs []BacklogPR) ([]BacklogPR, error) {
	backlogPRs := make([]BacklogPR, len(prs))
	for i, p := range prs {
		//Todo
		mergeBaseCommit, err := g.Exec("merge-base", p.BranchParent, p.BaseParent)
		if err != nil {
			return []BacklogPR{}, err
		}

		paths, err := g.Exec("--no-pager", "diff", "--name-only", p.BranchParent, strings.TrimSpace(mergeBaseCommit))

		if err != nil {
			return []BacklogPR{}, err
		}
		backlogPRs[i] = BacklogPR{
			PullRequestID: p.PullRequestID,
			BaseParent:    p.BaseParent,
			BranchParent:  p.BranchParent,
			FilePaths:     strings.Split(paths, "\n"),
		}
	}

	return backlogPRs, nil
}

func filterBacklogPR(commits []GitCommit) []BacklogPR {
	var filteredCommits []BacklogPR
	prIDSet := make(map[int]*BacklogPR)

	for _, commit := range commits {
		if matches := prMergeReg.FindStringSubmatch(commit.Message); len(matches) >= 2 {
			i, _ := strconv.Atoi(matches[1])
			commits := strings.Split(commit.Parent, " ")

			prIDSet[i] = &BacklogPR{
				PullRequestID: i,
				BaseParent:    strings.TrimSpace(commits[0]),
				BranchParent:  strings.TrimSpace(commits[1]),
			}
		}
	}

	for k := range prIDSet {
		filteredCommits = append(filteredCommits, *prIDSet[k])
	}

	return filteredCommits
}
