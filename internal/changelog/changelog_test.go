package changelog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v68/github"
	"github.com/gorilla/mux"
	"github.com/kgateway-dev/changelog-generator/internal/changelog"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

// TestGenerateChangelog_Mocked tests the changelog-generator logic
// by mocking GitHub API calls via go-github-mock.
func TestGenerateChangelog_Mocked(t *testing.T) {
	// Prepare mock transport
	mockedHTTPClient := mock.NewMockedHTTPClient(
		// 1. Mock Git.GetCommit for startSHA
		mock.WithRequestMatchHandler(
			mock.GetReposGitCommitsByOwnerByRepoByCommitSha,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				if vars["commit_sha"] == "start-sha" {
					json.NewEncoder(w).Encode(github.Commit{
						Committer: &github.CommitAuthor{
							Date: &github.Timestamp{Time: time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)},
						},
					})
					return
				}
				if vars["commit_sha"] == "end-sha" {
					json.NewEncoder(w).Encode(github.Commit{
						Committer: &github.CommitAuthor{
							Date: &github.Timestamp{Time: time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)},
						},
					})
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}),
		),
		// 3. Mock Search.Issues for merged PRs
		mock.WithRequestMatch(
			mock.GetSearchIssues,
			github.IssuesSearchResult{
				Issues: []*github.Issue{
					{Number: github.Int(42)},
				},
			},
		),
		// 4. Mock PullRequests.Get for PR #42
		mock.WithRequestMatch(
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			github.PullRequest{
				Number: github.Ptr(42),
				Title:  github.Ptr("Add new feature"),
				Body:   github.Ptr("```release-note\nMy note for PR42\n```"),
				Labels: []*github.Label{
					{Name: github.Ptr("kind/new-feature")},
				},
			},
		),
	)

	// Create GitHub client with mock transport
	ghClient := github.NewClient(mockedHTTPClient)

	// Create changelog generator
	generator := changelog.NewGenerator(ghClient, "foo", "bar")

	// Generate changelog
	changelog, err := generator.Generate(context.Background(), "start-sha", "end-sha")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Debug output
	fmt.Printf("Generated changelog:\n%s\n", changelog)

	// Verify changelog content
	if !strings.Contains(changelog, "🚀 Features") {
		t.Error("changelog missing Features section")
	}
	if !strings.Contains(changelog, "#42") {
		t.Error("changelog missing PR #42")
	}
}
