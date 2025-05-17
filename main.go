package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/cobra"

	"github.com/kgateway-dev/changelog-generator/internal/generator"
)

func main() {
	var token, owner, repo, startSHA, endSHA, outputPath string
	cmd := &cobra.Command{
		Use:   "changelog-generator <token> <owner> <repo> <start-sha> <end-sha> <output-path>",
		Short: "Generate a changelog between two commits",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client := github.NewClient(nil).WithAuthToken(token)

			g := generator.New(client, owner, repo)
			changelog, err := g.Generate(ctx, startSHA, endSHA)
			if err != nil {
				return fmt.Errorf("failed to generate changelog: %v", err)
			}
			if err := printChangelog(changelog, outputPath); err != nil {
				return fmt.Errorf("failed to print changelog: %v", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&token, "token", "t", "", "GitHub token")
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "GitHub owner")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "GitHub repository")
	cmd.Flags().StringVarP(&startSHA, "start-sha", "s", "", "Start commit SHA")
	cmd.Flags().StringVarP(&endSHA, "end-sha", "e", "", "End commit SHA")
	cmd.Flags().StringVarP(&outputPath, "output-path", "p", "", "Output path")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printChangelog(changelog string, outputPath string) error {
	if outputPath == "" {
		fmt.Println(changelog)
		return nil
	}
	if err := os.WriteFile(outputPath, []byte(changelog), 0644); err != nil {
		return fmt.Errorf("failed to write changelog to %s: %v", outputPath, err)
	}
	return nil
}
