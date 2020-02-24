package command

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/utils"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoViewCmd)
	repoCmd.AddCommand(repoForkCmd)
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "View repositories",
	Long: `Work with GitHub repositories.

A repository can be supplied as an argument in any of the following formats:
- "OWNER/REPO"
- by URL, e.g. "https://github.com/OWNER/REPO"`,
}

var repoViewCmd = &cobra.Command{
	Use:   "view [<repository>]",
	Short: "View a repository in the browser.",
	Long: `View a GitHub repository in the browser.

With no argument, the repository for the current directory is opened.`,
	RunE: repoView,
}

var repoForkCmd = &cobra.Command{
	Use:   "fork [<repository>]",
	Short: "Create a fork of a repository.",
	Long: `Create a fork of a repository.

With no argument, creates a fork of the current repository and adds a remote. Otherwise, forks the specified repository.`,
	RunE: repoFork,
}

func isURL(arg string) bool {
	return strings.HasPrefix(arg, "http:/") || strings.HasPrefix(arg, "https:/")
}

func repoFork(cmd *cobra.Command, args []string) error {
	ctx := contextForCommand(cmd)

	var toFork ghrepo.Interface
	if len(args) == 0 {
		baseRepo, err := determineBaseRepo(cmd, ctx)
		if err != nil {
			return fmt.Errorf("unable to determine base repository: %w", err)
		}
		// TODO check if user already has a fork
		toFork = baseRepo
	} else {
		repoArg := args[0]

		if isURL(repoArg) {
			parsedURL, err := url.Parse(repoArg)
			if err != nil {
				return fmt.Errorf("did not understand argument: %w", err)
			}

			toFork, err = ghrepo.FromURL(parsedURL)
			if err != nil {
				return fmt.Errorf("did not understand argument: %w", err)
			}

		} else {
			toFork = ghrepo.FromFullName(repoArg)
			if toFork.RepoName() == "" || toFork.RepoOwner() == "" {
				return fmt.Errorf("could not parse owner or repo name from %s", repoArg)
			}
		}
	}

	out := colorableOut(cmd)
	fmt.Fprintf(out, "Forking %s...\n", utils.Cyan(ghrepo.FullName(toFork)))

	return nil
}

func repoView(cmd *cobra.Command, args []string) error {
	ctx := contextForCommand(cmd)

	var openURL string
	if len(args) == 0 {
		baseRepo, err := determineBaseRepo(cmd, ctx)
		if err != nil {
			return err
		}
		openURL = fmt.Sprintf("https://github.com/%s", ghrepo.FullName(baseRepo))
	} else {
		repoArg := args[0]
		if isURL(repoArg) {
			openURL = repoArg
		} else {
			openURL = fmt.Sprintf("https://github.com/%s", repoArg)
		}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Opening %s in your browser.\n", displayURL(openURL))
	return utils.OpenInBrowser(openURL)
}
