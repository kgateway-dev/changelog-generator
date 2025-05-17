package generator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v68/github"
)

// Regex to extract a fenced release-note block
var releaseNoteRE = regexp.MustCompile("(?s)```release-note\\s*(.*?)\\s*```")

// Supported kinds for bucket headers
var kindHeaders = map[string]string{
	"new_feature":     "🚀 Features",
	"bug_fix":         "🐛 Bug Fixes",
	"breaking_change": "💥 Breaking Changes",
	"documentation":   "📝 Documentation",
	"performance":     "⚡ Performance Improvements",
}

// Generator handles changelog generation for a repository
type Generator struct {
	client *github.Client
	owner  string
	repo   string
}

// New creates a new changelog generator
func New(client *github.Client, owner, repo string) *Generator {
	return &Generator{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// Generate creates a changelog between two commits
func (g *Generator) Generate(ctx context.Context, startSHA, endSHA string) (string, error) {
	// Get the start and end commits
	startCommit, _, err := g.client.Git.GetCommit(ctx, g.owner, g.repo, startSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get start commit: %v", err)
	}

	endCommit, _, err := g.client.Git.GetCommit(ctx, g.owner, g.repo, endSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get end commit: %v", err)
	}

	// Get all commits between start and end
	commits, err := g.getCommitsBetween(ctx, startCommit, endCommit)
	if err != nil {
		return "", fmt.Errorf("failed to get commits: %v", err)
	}

	// Get all PRs referenced in the commits
	prs, err := g.getReferencedPRs(ctx, commits)
	if err != nil {
		return "", fmt.Errorf("failed to get PRs: %v", err)
	}

	// Generate the changelog
	return g.generateChangelog(prs), nil
}

// getCommitsBetween returns all commits between start and end (inclusive)
func (g *Generator) getCommitsBetween(_ context.Context, start, end *github.Commit) ([]*github.Commit, error) {
	// For now, just return the start and end commits
	return []*github.Commit{start, end}, nil
}

// getReferencedPRs returns all PRs referenced in the given commits
func (g *Generator) getReferencedPRs(ctx context.Context, _ []*github.Commit) ([]*github.PullRequest, error) {
	var prs []*github.PullRequest

	// Search for PRs referenced in commit messages
	query := fmt.Sprintf(`repo:%s/%s is:pr is:merged label:"release-note","release-note-needed"`, g.owner, g.repo)
	result, _, err := g.client.Search.Issues(ctx, query, &github.SearchOptions{
		TextMatch: true,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search for PRs: %v", err)
	}

	// Get full PR details for each PR number
	for _, issue := range result.Issues {
		pr, _, err := g.client.PullRequests.Get(ctx, g.owner, g.repo, *issue.Number)
		if err != nil {
			return nil, fmt.Errorf("failed to get PR %d: %v", *issue.Number, err)
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// generateChangelog generates a changelog from the given PRs
func (g *Generator) generateChangelog(prs []*github.PullRequest) string {
	var changelog strings.Builder

	// Group PRs by kind
	buckets := make(map[string][]*github.PullRequest)
	for _, pr := range prs {
		kind := g.getPRKind(pr)
		buckets[kind] = append(buckets[kind], pr)
	}

	// Print each bucket
	for kind, header := range kindHeaders {
		prs := buckets[kind]
		if len(prs) == 0 {
			continue
		}
		changelog.WriteString(fmt.Sprintf("\n## %s\n\n", header))
		for _, pr := range prs {
			body := pr.GetBody()
			if body == "" {
				continue
			}
			// attempt to extract a release-note from the PR body
			match := releaseNoteRE.FindStringSubmatch(body)
			if len(match) < 2 {
				// skip PRs that have the release-note label but no release-note body.
				// TODO(tim): this shouldn't be possible with the labeler being a required
				// check, but we'll check for it anyway in case users have manually added
				// the label to a PR.
				continue
			}
			note := match[1]
			if note == "" {
				// TODO(tim): we should probably log this as an error as this is unexpected
				continue
			}
			changelog.WriteString(fmt.Sprintf("- %s (#%d)\n", note, pr.GetNumber()))
		}
	}

	return changelog.String()
}

// getPRKind determines the kind of a PR based on its labels and title
func (g *Generator) getPRKind(pr *github.PullRequest) string {
	// Check labels first
	for _, label := range pr.Labels {
		switch *label.Name {
		case "kind/feature", "kind/new_feature":
			return "new_feature"
		case "kind/fix", "kind/bug_fix":
			return "bug_fix"
		case "kind/breaking_change":
			return "breaking_change"
		case "kind/documentation":
			return "documentation"
		case "kind/performance":
			return "performance"
		}
	}

	return "other"
}
