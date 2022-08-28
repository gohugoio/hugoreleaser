package changelog

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CollectChanges collects changes according to the given options.
// If opts.ResolveUserName is set, it will be used to resolve Change.Username (e.g. GitHub login).
func CollectChanges(opts Options) (Changes, error) {
	c := &collector{opts: opts}
	return c.collect()
}

// GroupByTitleFunc groups g by title according to the grouping function f.
// If f returns false, that change item is not included in the result.
func GroupByTitleFunc(g Changes, f func(Change) (string, int, bool)) ([]TitleChanges, error) {
	var ngi []TitleChanges
	for _, gi := range g {
		title, i, ok := f(gi)
		if !ok {
			continue
		}
		idx := -1
		for j, ngi := range ngi {
			if ngi.Title == title {
				idx = j
				break
			}
		}
		if idx == -1 {
			ngi = append(ngi, TitleChanges{Title: title, ordinal: i + 1})
			idx = len(ngi) - 1
		}
		ngi[idx].Changes = append(ngi[idx].Changes, gi)
	}

	sort.Slice(ngi, func(i, j int) bool {
		return ngi[i].ordinal < ngi[j].ordinal
	})

	return ngi, nil

}

// Change represents a git commit.
type Change struct {
	// Fetched from git log.
	Hash    string
	Author  string
	Subject string
	Body    string

	Issues []int

	// Resolved from GitHub.
	Username string
}

// Changes represents a list of git commits.
type Changes []Change

// Options for collecting changes.
type Options struct {
	// Can be nil.
	// ResolveUserName returns the username for the given author (email address) and commit sha.
	// This is the GitHub login (e.g. bep) in its first iteration.
	ResolveUserName func(commit, author string) (string, error)

	// All of these can be empty.
	PrevTag   string
	Tag       string
	Commitish string
	RepoPath  string
}

// TitleChanges represents a list of changes grouped by title.
type TitleChanges struct {
	Title   string
	Changes Changes

	ordinal int
}

type collector struct {
	opts Options
}

func (c *collector) collect() (Changes, error) {
	log, err := gitLog(c.opts.RepoPath, c.opts.PrevTag, c.opts.Tag, c.opts.Commitish)
	if err != nil {
		return nil, err
	}
	g, err := gitLogToGitInfos(log)
	if err != nil {
		return nil, err
	}

	if c.opts.ResolveUserName != nil {
		for i, gi := range g {
			username, err := c.opts.ResolveUserName(gi.Hash, gi.Author)
			if err != nil {
				return nil, err
			}
			g[i].Username = username
		}
	}

	return g, nil
}

func git(repo string, args ...string) (string, error) {
	if repo != "" {
		args = append([]string{"-C", repo}, args...)
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git failed: %q: %q (%q)", err, out, args)
	}
	return string(out), nil
}

func gitLog(repo, prevTag, tag, commitish string) (string, error) {
	var err error
	if prevTag != "" {
		exists, err := gitTagExists(repo, prevTag)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("prevTag %q does not exist", prevTag)
		}
	}

	if tag != "" {
		exists, err := gitTagExists(repo, tag)
		if err != nil {
			return "", err
		}
		if !exists {
			// Assume it hasn't been created yet,
			tag = ""
		}
	}

	var from, to string
	if prevTag != "" {
		from = prevTag
	}
	if tag != "" {
		to = tag
	}

	if to == "" {
		to = commitish
	}

	if from == "" {
		from, err = gitVersionTagBefore(repo, to)
		if err != nil {
			return "", err
		}
	}

	args := []string{"log", "--pretty=format:%x1e%h%x1f%aE%x1f%s%x1f%b", "--abbrev-commit", from + ".." + to}

	log, err := git(repo, args...)
	if err != nil {
		return ",", err
	}

	return log, err
}

func gitLogToGitInfos(log string) (Changes, error) {
	var g Changes
	log = strings.Trim(log, "\n\x1e'")
	entries := strings.Split(log, "\x1e")

	for _, entry := range entries {
		items := strings.Split(entry, "\x1f")
		var gi Change

		if len(items) > 0 {
			gi.Hash = items[0]
		}
		if len(items) > 1 {
			gi.Author = items[1]
		}
		if len(items) > 2 {
			gi.Subject = items[2]
		}
		if len(items) > 3 {
			gi.Body = items[3]

			// Parse issues.
			gi.Issues = parseIssues(gi.Body)
		}

		g = append(g, gi)
	}

	return g, nil
}

func gitShort(repo string, args ...string) (output string, err error) {
	output, err = git(repo, args...)
	return strings.Replace(strings.Split(output, "\n")[0], "'", "", -1), err
}

func gitTagExists(repo, tag string) (bool, error) {
	out, err := git(repo, "tag", "-l", tag)
	if err != nil {
		return false, err
	}

	if strings.Contains(out, tag) {
		return true, nil
	}

	return false, nil
}

func gitVersionTagBefore(repo, ref string) (string, error) {
	return gitShort(repo, "describe", "--tags", "--abbrev=0", "--always", "--match", "v[0-9]*", ref+"^")
}

var issueRe = regexp.MustCompile(`(?i)(?:Updates?|Closes?|Fix.*|See) #(\d+)`)

func parseIssues(body string) []int {
	var i []int
	m := issueRe.FindAllStringSubmatch(body, -1)
	for _, mm := range m {
		issueID, err := strconv.Atoi(mm[1])
		if err != nil {
			continue
		}
		i = append(i, issueID)
	}
	return i
}
