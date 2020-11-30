package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
	"github.com/segmentio/cli"
	"golang.org/x/oauth2"
)

const (
	githubTokenEnv = "GITHUB_TOKEN"
)

type config struct {
	Domain string `flag:"--domain" help:"Domain where content will be hosted"`
	VCS    string `flag:"--vcs" help:"Base VCS URL where the code lives, eg. github.com/eculver"`
	Match  string `flag:"--match" help:"Regular expression to be used for matching repository names" default:""`
}

func main() {
	cli.Exec(cli.Command(func(conf config, repoDirs []string) {
		// maybe this matcher fu should manifest differently, but it seems like
		// main should be the one throwing panics or errors based on user
		// input and not the internals of the code that is doing I/O (aka RepositoryReader implementations)
		matchAll := func(_ string) bool {
			return true
		}
		matchRegexp := func(pattern string) RepositoryMatcher {
			re := regexp.MustCompile(pattern)
			return func(name string) bool {
				return re.MatchString(name)
			}
		}
		matcherFunc := matchAll
		if conf.Match != "" {
			matcherFunc = matchRegexp(conf.Match)
		}

		var reader RepositoryReader
		if len(repoDirs) > 0 {
			reader = NewLocalReader(repoDirs, conf.VCS, matcherFunc)
		} else if strings.HasPrefix(conf.VCS, "github.com") {
			token := os.Getenv(githubTokenEnv)
			if token == "" {
				log.Fatalf("GitHub token not found, please set %s", githubTokenEnv)
			}
			vcsPts := strings.Split(conf.VCS, "/")
			if len(vcsPts) != 2 {
				log.Fatalf("Invalid VCS: must be in form github.com/$org for GitHub, got %s", conf.VCS)
			}
			org := vcsPts[1]
			reader = NewGitHubReader(token, org, matcherFunc)
		}
		if reader == nil {
			log.Fatal("Could not determine reader, must provide repository directories or --vcs that matches one of the supported prefixes")
		}

		h := host{Domain: conf.Domain, Repositories: &repositories{}}
		if err := reader.Read(h.Repositories); err != nil {
			log.Fatal(err)
		}
		bs, err := json.MarshalIndent(h, "", "\t")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(bs))
	}))
}

// StringReplacer is the signature for a function that replaces parts of the from string to produce a new string.
// this type is used as a generic way to translate repo names without having to specify
// the method (regex, interpolation) by which the replacement actually happens.
// TODO: remove?
type StringReplacer func(from string) string

// RepositoryMatcher takes a repository name and returns whether the repository should be included in results.
type RepositoryMatcher func(string) bool

// RepositoryReader reads repositories from some source.
type RepositoryReader interface {
	// Read marshals repositories into the slice given.
	// An error should be returned if those repositories cannot be populated with indication of the source.
	Read(*repositories) error

	// GetPrefix translates the name of the repository as it exists in the source.
	GetPrefix(string) string
}

type localReader struct {
	dirs        []string
	vcs         string
	matcherFunc RepositoryMatcher
}

// NewLocalReader returns a reader to read from the local file system.
func NewLocalReader(dirs []string, vcs string, matcherFunc RepositoryMatcher) *localReader {
	return &localReader{
		dirs:        dirs,
		vcs:         vcs,
		matcherFunc: matcherFunc,
	}
}

// Read reads repositories from a slice of directory names. If one of the paths
// is not a directory, an error will be returned.
// TODO: discover subpackages
func (r *localReader) Read(repos *repositories) error {
	for _, repoDir := range r.dirs {
		fi, err := os.Stat(repoDir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			fmt.Printf("%s is not a directory, skipping\n", repoDir)
			continue
		}
		if !r.matcherFunc(fi.Name()) {
			continue
		}

		// TODO: this should be based on local VCS config in .git/config
		homeURL := fmt.Sprintf("https://%s/%s", r.vcs, fi.Name())
		dirURL := fmt.Sprintf("%s/tree/master/{/dir}", homeURL)
		fileURL := fmt.Sprintf("%s/blob/master{/dir}/{file}#L{line}", homeURL)

		repo := repository{
			Prefix:  r.GetPrefix(fi.Name()),
			URL:     homeURL,
			Website: website{URL: homeURL},
			SourceURLs: sourceURLs{
				Home: homeURL,
				Dir:  dirURL,
				File: fileURL,
			},
		}
		repos.append(repo)
	}
	return nil
}

// GetPrefix translates the directory name to the top-level package name
func (r *localReader) GetPrefix(from string) string {
	re := regexp.MustCompile(`^go\-`)
	return re.ReplaceAllString(from, "")
}

type githubReader struct {
	org         string
	client      *github.Client
	matcherFunc RepositoryMatcher
}

// NewGitHubReader returns a new RepositoryReader for listing repositories in the GitHub organization matching org and using
// a static Personal Access Token token for auth. For each repository found, matcherFunc will be called to determine whether
// it should be included in the results.
func NewGitHubReader(token, org string, matcherFunc RepositoryMatcher) *githubReader {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &githubReader{
		org:         org,
		client:      github.NewClient(tc),
		matcherFunc: matcherFunc,
	}
}

// Read reads repositories from GitHub. Any errors from the GitHub client
// are forwarded.
// TODO: discover subpackages
func (r *githubReader) Read(repos *repositories) error {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
		Visibility: "public",
	}

	// list all repositories for the authenticated user
	for {
		ghrepos, resp, err := r.client.Repositories.List(context.Background(), r.org, opts)
		if err != nil {
			return err
		}

		for _, ghrepo := range ghrepos {
			log.Printf("found repo: %s", *ghrepo.Name)
			if !r.matcherFunc(*ghrepo.Name) {
				continue
			}
			log.Printf("found match: %s", *ghrepo.Name)
			homeURL := *ghrepo.HTMLURL
			dirURL := fmt.Sprintf("%s/tree/master/{/dir}", homeURL)
			fileURL := fmt.Sprintf("%s/blob/master{/dir}/{file}#L{line}", homeURL)
			repo := repository{
				Prefix:  r.GetPrefix(*ghrepo.Name),
				URL:     homeURL,
				Website: website{URL: homeURL},
				SourceURLs: sourceURLs{
					Home: homeURL,
					Dir:  dirURL,
					File: fileURL,
				},
			}
			repos.append(repo)
		}
		if resp.LastPage == opts.Page {
			break
		}
		opts.Page = resp.NextPage
	}
	return nil
}

// GetPrefix translates the repository name to the top-level package name
func (r *githubReader) GetPrefix(from string) string {
	re := regexp.MustCompile(`^go\-`)
	return re.ReplaceAllString(from, "")
}
