package command

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/cli/api"
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
	// TODO enable this working outside of repo
	ctx := contextForCommand(cmd)
	apiClient, err := apiClientForContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	var toFork ghrepo.Interface
	if len(args) == 0 {
		baseRepo, err := determineBaseRepo(cmd, ctx)
		if err != nil {
			return fmt.Errorf("unable to determine base repository: %w", err)
		}
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

	authLogin, err := ctx.AuthLogin()
	if err != nil {
		return fmt.Errorf("could not determine current username: %w", err)
	}

	possibleFork := ghrepo.New(authLogin, toFork.RepoName())
	exists, err := api.RepoExistsOnGitHub(apiClient, possibleFork)
	if err != nil {
		return fmt.Errorf("problem with API request: %w", err)
	}

	if exists {
		return fmt.Errorf("%s %s", utils.Cyan(ghrepo.FullName(possibleFork)), utils.Red("already exists!"))
	}

	forkedRepo, err := api.ForkRepo(apiClient, toFork)
	if err != nil {
		return fmt.Errorf("failed to fork: %w", err)
	}

	fmt.Fprintf(out, "%s %s %s!\n",
		utils.Cyan(ghrepo.FullName(toFork)),
		utils.Green("successfully forked to"),
		utils.Cyan(ghrepo.FullName(forkedRepo)))

	fmt.Fprintf(out, "Add new fork as a remote: git remote add fork %s\n", forkedRepo.CloneURL)
	// TODO soon gh repo clone
	fmt.Fprintf(out, "Clone the new fork: git clone %s\n", forkedRepo.CloneURL)

	// TODO: should we do more, here?

	// - if no repo was specified, we're "in" a repo and creating a fork of it. probably they want a
	// remote called "fork" added (we could prompt about this or just tell them how to do it).
	// - if this was called with an arg, we're going to need to actually clone the fork so they can
	// use it (right?)

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
