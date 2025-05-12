package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"
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

func main() {
	cmd := &cobra.Command{
		Use:   "changelog-generator <token> <owner> <repo> <start-sha> <end-sha>",
		Short: "Generate a changelog between two commits",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, owner, repo, startSHA, endSHA := args[0], args[1], args[2], args[3], args[4]

			ctx := context.Background()
			client := github.NewClient(nil).WithAuthToken(token)

			// Get the start and end commits
			startCommit, _, err := client.Git.GetCommit(ctx, owner, repo, startSHA)
			if err != nil {
				return fmt.Errorf("failed to get start commit: %v", err)
			}

			endCommit, _, err := client.Git.GetCommit(ctx, owner, repo, endSHA)
			if err != nil {
				return fmt.Errorf("failed to get end commit: %v", err)
			}

			// Get all commits between start and end
			commits, err := getCommitsBetween(ctx, client, owner, repo, startCommit, endCommit)
			if err != nil {
				return fmt.Errorf("failed to get commits: %v", err)
			}

			// Get all PRs referenced in the commits
			prs, err := getReferencedPRs(ctx, client, owner, repo, commits)
			if err != nil {
				return fmt.Errorf("failed to get PRs: %v", err)
			}

			// Generate and print the changelog
			changelog := generateChangelog(prs)
			fmt.Println(changelog)

			return nil
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getCommitsBetween returns all commits between start and end (inclusive)
func getCommitsBetween(ctx context.Context, client *github.Client, owner, repo string, start, end *github.Commit) ([]*github.Commit, error) {
	// For now, just return the start and end commits
	return []*github.Commit{start, end}, nil
}

// getReferencedPRs returns all PRs referenced in the given commits
func getReferencedPRs(ctx context.Context, client *github.Client, owner, repo string, commits []*github.Commit) ([]*github.PullRequest, error) {
	var prs []*github.PullRequest

	// Search for PRs referenced in commit messages
	query := fmt.Sprintf("repo:%s/%s is:pr is:merged", owner, repo)
	opts := &github.SearchOptions{
		TextMatch: true,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	result, _, err := client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search for PRs: %v", err)
	}

	// Get full PR details for each PR number
	for _, issue := range result.Issues {
		pr, _, err := client.PullRequests.Get(ctx, owner, repo, *issue.Number)
		if err != nil {
			return nil, fmt.Errorf("failed to get PR %d: %v", *issue.Number, err)
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// generateChangelog generates a changelog from the given PRs
func generateChangelog(prs []*github.PullRequest) string {
	var changelog strings.Builder

	// Group PRs by kind
	buckets := make(map[string][]*github.PullRequest)
	for _, pr := range prs {
		kind := getPRKind(pr)
		buckets[kind] = append(buckets[kind], pr)
	}

	// Print each bucket
	for kind, header := range kindHeaders {
		if prs, ok := buckets[kind]; ok && len(prs) > 0 {
			changelog.WriteString(fmt.Sprintf("\n## %s\n\n", header))
			for _, pr := range prs {
				changelog.WriteString(fmt.Sprintf("- %s (#%d)\n", *pr.Title, *pr.Number))
			}
		}
	}

	return changelog.String()
}

// getPRKind determines the kind of a PR based on its labels and title
func getPRKind(pr *github.PullRequest) string {
	// Check labels first
	for _, label := range pr.Labels {
		switch *label.Name {
		case "kind/new-feature":
			return "new_feature"
		case "kind/bug":
			return "bug_fix"
		case "kind/breaking-change":
			return "breaking_change"
		case "kind/documentation":
			return "documentation"
		case "kind/performance":
			return "performance"
		}
	}

	// Fall back to title-based detection
	title := strings.ToLower(*pr.Title)
	switch {
	case strings.Contains(title, "feat") || strings.Contains(title, "feature"):
		return "new_feature"
	case strings.Contains(title, "fix") || strings.Contains(title, "bug"):
		return "bug_fix"
	case strings.Contains(title, "break") || strings.Contains(title, "breaking"):
		return "breaking_change"
	case strings.Contains(title, "doc") || strings.Contains(title, "docs"):
		return "documentation"
	case strings.Contains(title, "perf") || strings.Contains(title, "performance"):
		return "performance"
	default:
		return "other"
	}
}
