package backlog_pr

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/trknhr/backlog-git-pr-diff-checker/git_cmd"
	backlog "github.com/vvatanabe/go-backlog/backlog/v2"
)

type BacklogGit struct {
	GitCmd      *git_cmd.GitCmd
	LastCommit  string
	ApiKey      string
	TargetPaths []string
}

type BacklogInfo struct {
	PullRequests []*backlog.PullRequest
	RepoURL      string
}

func GetPullRequest(bg *BacklogGit) (*BacklogInfo, error) {
	prMergeCommits, err := bg.GitCmd.GetPRMergedCommits(bg.LastCommit)

	if err != nil {
		return nil, err
	}

	remoteURL, err := bg.getRemoteURL()

	if err != nil {
		return nil, err
	}

	spaceDomain := bg.spaceDomain(remoteURL)
	projectKey, repoName := bg.projectKeyAndRepo(remoteURL)

	client := backlog.NewClient(spaceDomain, nil)

	client.SetAPIKey(bg.ApiKey)

	targetPRs := []*backlog.PullRequest{}
	combineRegexStr := fmt.Sprintf("(%s)", strings.Join(bg.TargetPaths[:], "|"))
	for _, p := range prMergeCommits {
		if existsTargetPaths(combineRegexStr, p) {
			pr, _, err := client.PullRequests.GetPullRequest(context.Background(), projectKey, repoName, p.PullRequestID)
			if err != nil {
				return nil, err
			}

			targetPRs = append(targetPRs, pr)
		}

	}

	return &BacklogInfo{
		PullRequests: targetPRs,
		RepoURL:      fmt.Sprintf("%s/git/%s/%s", fmt.Sprintf("https://%s", spaceDomain), projectKey, repoName),
	}, nil
}

func existsTargetPaths(combineRegexStr string, pr git_cmd.BacklogPR) bool {
	for _, filePath := range pr.FilePaths {
		match, _ := regexp.MatchString(combineRegexStr, filePath)

		if match {
			return true
		}
	}

	return false
}

var serviceDomains = []string{"backlog.jp", "backlog.com", "backlogtool.com"}

type remoteURL struct {
	Protocol string
	Host     string
	Port     string
	Path     string
}

func (b *BacklogGit) getRemoteURL() (*remoteURL, error) {
	out, err := b.GitCmd.Exec("remote", "-v")

	if err != nil {
		return nil, err
	}

	remotes := strings.Split(out, "\n")

	remote := b.getRemote()
	var u string
	for _, r := range remotes {
		fields := strings.Fields(r)
		if len(fields) > 1 && fields[0] == remote {
			u = fields[1]
			break
		}
	}

	var remoteURL *remoteURL
	if isHTTP(u) {
		remoteURL = toRemoteURLFromHTTP(u)
	} else if isSSH(u) {
		remoteURL = toRemoteURLFromSSH(u)
	} else {
		return remoteURL, errors.New("could not be used protocol except http and ssh")
	}

	return remoteURL, nil
}

func (b *BacklogGit) getRemote() string {
	return "origin"
}

func isHTTP(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func toRemoteURLFromHTTP(str string) *remoteURL {
	u, _ := url.Parse(str)
	return &remoteURL{
		Protocol: u.Scheme,
		Host:     u.Host,
		Port:     u.Port(),
		Path:     u.Path,
	}
}

var sshURLReg = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:\s]+):(?:(?P<port>[0-9]{1,5})/)?(?P<path>[^\\].*)$`)

func isSSH(str string) bool {
	return sshURLReg.MatchString(str)
}

func toRemoteURLFromSSH(str string) *remoteURL {
	m := sshURLReg.FindStringSubmatch(str)
	return &remoteURL{
		Protocol: "ssh",
		Host:     m[2],
		Port:     m[3],
		Path:     m[4],
	}
}

var repoURLReg = regexp.MustCompile(`([^/:]+)/([^/]+?)(?:\.git)?$`)

func (b *BacklogGit) projectKeyAndRepo(remoteURL *remoteURL) (projectKey, repo string) {
	if matches := repoURLReg.FindStringSubmatch(remoteURL.Path); len(matches) > 2 {
		return matches[1], matches[2]
	}
	return
}

func (b *BacklogGit) spaceDomain(remoteURL *remoteURL) string {
	var isBacklogDomain bool
	for _, d := range serviceDomains {
		if strings.HasSuffix(remoteURL.Host, "."+d) {
			isBacklogDomain = true
			break
		}

	}

	if !isBacklogDomain {
		return remoteURL.Host
	}

	if strings.HasPrefix(remoteURL.Protocol, "http") {
		return remoteURL.Host
	}

	delimitedHost := strings.Split(remoteURL.Host, ".")
	spaceKey := delimitedHost[0]
	domain := strings.Join(delimitedHost[len(delimitedHost)-2:], ".")
	return fmt.Sprintf("%s.%s", spaceKey, domain)
}
