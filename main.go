package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v68/github"
	"github.com/kgateway-dev/changelog-generator/internal/generator"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "changelog-generator <token> <owner> <repo> <start-sha> <end-sha> <output-path>",
		Short: "Generate a changelog between two commits",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, owner, repo, startSHA, endSHA, outputPath := args[0], args[1], args[2], args[3], args[4], args[5]

			ctx := context.Background()
			client := github.NewClient(nil).WithAuthToken(token)

			g := generator.New(client, owner, repo)
			changelog, err := g.Generate(ctx, startSHA, endSHA)
			if err != nil {
				return fmt.Errorf("failed to generate changelog: %v", err)
			}
			if err := os.WriteFile(outputPath, []byte(changelog), 0644); err != nil {
				return fmt.Errorf("failed to write changelog to %s: %v", outputPath, err)
			}
			fmt.Printf("::set-output name=changelog-path::%s\n", outputPath)

			return nil
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
