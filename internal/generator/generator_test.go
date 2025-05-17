package generator_test

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
	"github.com/kgateway-dev/changelog-generator/internal/generator"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

// TestGenerateChangelog tests the changelog-generator logic
// by mocking GitHub API calls via go-github-mock.
func TestGenerateChangelog(t *testing.T) {
	tt := []struct {
		name              string
		owner             string
		repo              string
		startSHA          string
		startSHADate      time.Time
		endSHA            string
		endSHADate        time.Time
		issuesForSearch   []*github.Issue
		pullRequests      []*github.PullRequest
		expectedChangelog string
		expectError       bool
	}{
		{
			name:         "Valid/Feature PR with release-note",
			owner:        "foo",
			repo:         "bar",
			startSHA:     "start-sha",
			startSHADate: time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC),
			endSHA:       "end-sha",
			endSHADate:   time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC),
			issuesForSearch: []*github.Issue{
				{Number: github.Ptr(42)},
			},
			pullRequests: []*github.PullRequest{{
				Number: github.Ptr(42),
				Title:  github.Ptr("Add new feature"),
				Body:   github.Ptr("```release-note\nMy note for PR42\n```"),
				Labels: []*github.Label{{
					Name: github.Ptr("kind/feature"),
				}},
			}},
			expectedChangelog: `
## 🚀 Features

- My note for PR42 (#42)
`,
			expectError: false,
		},
		{
			name:         "Valid/No PRs with release-note content",
			owner:        "foo",
			repo:         "bar",
			startSHA:     "start-sha",
			startSHADate: time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC),
			endSHA:       "end-sha",
			endSHADate:   time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC),
			issuesForSearch: []*github.Issue{
				{Number: github.Ptr(43)},
			},
			pullRequests: []*github.PullRequest{{
				Number: github.Ptr(43),
				Title:  github.Ptr("Refactor code"),
				Body:   github.Ptr("No release note here."),
				Labels: []*github.Label{{
					Name: github.Ptr("irrelevant-label"),
				}},
			}},
			expectedChangelog: "",
			expectError:       false,
		},
		{
			name:         "Valid/Multiple PRs with different kinds",
			owner:        "foo",
			repo:         "bar",
			startSHA:     "start-sha",
			startSHADate: time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC),
			endSHA:       "end-sha",
			endSHADate:   time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC),
			issuesForSearch: []*github.Issue{
				{Number: github.Ptr(42)},
				{Number: github.Ptr(43)},
				{Number: github.Ptr(44)},
			},
			pullRequests: []*github.PullRequest{
				{
					Number: github.Ptr(42),
					Title:  github.Ptr("Add new feature"),
					Body:   github.Ptr("```release-note\nImplement new feature\n```"),
					Labels: []*github.Label{{
						Name: github.Ptr("kind/feature"),
					}},
				},
				{
					Number: github.Ptr(43),
					Title:  github.Ptr("Fix bug"),
					Body:   github.Ptr("```release-note\nFixed a bug\n```"),
					Labels: []*github.Label{{
						Name: github.Ptr("kind/fix"),
					}},
				},
				{
					Number: github.Ptr(44),
					Title:  github.Ptr("Remove old feature"),
					Body:   github.Ptr("```release-note\nRemoved old feature\n```"),
					Labels: []*github.Label{{
						Name: github.Ptr("kind/breaking_change"),
					}},
				},
			},
			expectedChangelog: `
## 🚀 Features

- Implement new feature (#42)

## 🐛 Bug Fixes

- Fixed a bug (#43)

## 💥 Breaking Changes

- Removed old feature (#44)
`,
			expectError: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			currentMockHandlers := []mock.MockBackendOption{
				mockGetCommitHandler(tc.startSHA, tc.startSHADate, tc.endSHA, tc.endSHADate),
				mockSearchIssuesHandler(tc.issuesForSearch),
			}
			if len(tc.pullRequests) > 0 {
				currentMockHandlers = append(currentMockHandlers, mockGetPullRequestsHandler(tc.pullRequests))
			}
			mockedHTTPClient := mock.NewMockedHTTPClient(currentMockHandlers...)

			g := generator.New(github.NewClient(mockedHTTPClient), tc.owner, tc.repo)
			changelog, err := g.Generate(context.Background(), tc.startSHA, tc.endSHA)
			switch {
			case tc.expectError && err == nil:
				t.Fatalf("Expected an error, but got nil")
			case !tc.expectError && err != nil:
				t.Fatalf("Expected no error, but got: %v", err)
			default:
				if strings.TrimSpace(changelog) != strings.TrimSpace(tc.expectedChangelog) {
					t.Fatalf("Generated changelog does not match expected changelog:\nwant: %s\ngot: %s", tc.expectedChangelog, changelog)
				}
			}
		})
	}
}

// mockGetCommitHandler creates a mock handler for the GetCommit API call.
func mockGetCommitHandler(startSHA string, startDate time.Time, endSHA string, endDate time.Time) mock.MockBackendOption {
	return mock.WithRequestMatchHandler(
		mock.GetReposGitCommitsByOwnerByRepoByCommitSha,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			commitSHA := vars["commit_sha"]
			var commitDate time.Time
			switch commitSHA {
			case startSHA:
				commitDate = startDate
			case endSHA:
				commitDate = endDate
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(github.Commit{
				Committer: &github.CommitAuthor{
					Date: &github.Timestamp{Time: commitDate},
				},
			})
		}),
	)
}

// mockSearchIssuesHandler creates a mock handler for the SearchIssues API call.
func mockSearchIssuesHandler(issues []*github.Issue) mock.MockBackendOption {
	return mock.WithRequestMatch(
		mock.GetSearchIssues,
		github.IssuesSearchResult{
			Issues: issues,
		},
	)
}

// mockGetPullRequestsHandler creates a mock handler for GetPullRequests API calls.
// It mocks responses for each PR in the provided slice based on its number.
func mockGetPullRequestsHandler(prs []*github.PullRequest) mock.MockBackendOption {
	return mock.WithRequestMatchHandler(
		mock.GetReposPullsByOwnerByRepoByPullNumber,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			prNumberStr := vars["pull_number"]
			for _, pr := range prs {
				if pr.Number != nil && fmt.Sprintf("%d", *pr.Number) == prNumberStr {
					json.NewEncoder(w).Encode(pr)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}),
	)
}
