package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blang/semver"
	"github.com/frairon/semvertag"
	"github.com/spf13/pflag"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

var (
	repo   = pflag.String("repo", "", "directory to repo. If empty, the current directory will be used")
	prefix = pflag.String("prefix", "", "Tag prefix")

	patch = pflag.Bool("patch", false, "Upgrade patch version")
	minor = pflag.Bool("minor", false, "Upgrade minor version")
	major = pflag.Bool("major", false, "Upgrade major version")

	noFetch            = pflag.Bool("no-fetch", false, "Do not fetch before creating a new version")
	noPush             = pflag.Bool("no-push", false, "Do not push the new tag")
	quiet              = pflag.Bool("quiet", false, "Do not ask before setting (and pushing) the new tag")
	noSelectBranch     = pflag.Bool("no-select-branch", false, "Do not ask to select a branch to tag if we're NOT on the master.")
	alwaysSelectBranch = pflag.Bool("always-select-branch", false, "Always select the branch to tag")

	username = pflag.String("username", "", "Username to create tag with. Try to guess or prompt if not provided")
	email    = pflag.String("email", "", "Email to create tag with. Try to guess or prompt if not provided")

	tagMessage = pflag.String("message", "", "Optional tag message. Will open a prompt if not set, unless --quiet is set")

	printLast = pflag.Int("print-last", 5, "Print last n tags for information")
)

const (
	dateFormat = "2006-01-02 15:04:05Z07"
)

func checkArgs() {
	var argsSet int
	if *patch {
		argsSet++
	}
	if *minor {
		argsSet++
	}
	if *major {
		argsSet++
	}

	if argsSet != 1 {
		log.Fatalf("Set exactly one of major, minor, or patch to increment")
	}

	if *noSelectBranch && *alwaysSelectBranch {
		log.Fatalf("Can set at most one of --no-branch-select and --always-select-branch")
	}

	if *alwaysSelectBranch && *quiet {
		log.Fatalf("Cannot do --quiet and --always-select-branch at the same time")
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("> ")
	pflag.Parse()

	checkArgs()

	var (
		dir string
		err error
	)
	if *repo != "" {
		dir = *repo
	} else {
		dir, err = os.Getwd()
		if err != nil {
			log.Fatalf("Error getting working directory: %v", err)
		}
	}

	r, err := git.PlainOpen(dir)
	if err != nil {
		log.Fatalf("Error opening directory %s as repository: %v", dir, err)
	}
	if !*noFetch {
		if err = semvertag.Execute(dir, "git", "fetch", "--tags"); err != nil {
			log.Fatalf("Error fetching new version of repo")
		}
	}

	tags := semvertag.GetSortedMatchingTags(r, *prefix)

	if *printLast > 0 {
		log.Printf("Last %d tags with matching pattern:", *printLast)
		for _, tag := range tags {
			log.Printf("  %s   %s (%s by %s)", tag.Tag.Name, tag.Tag.Hash.String()[:6], tag.Tag.Tagger.When.Format(dateFormat), tag.Tag.Tagger.Name)
		}
	}

	var lastVersion semver.Version
	if len(tags) > 0 {
		lastVersion = tags[0].Version
	}

	if *patch {
		lastVersion.Patch++
	}
	if *minor {
		lastVersion.Minor++
	}
	if *major {
		lastVersion.Major++
	}

	currentBranch, err := r.Head()
	if err != nil {
		log.Fatalf("error getting head tag. %v", err)
	}
	var (
		commitToTag plumbing.Hash
		branchName  = currentBranch.Name().Short()
	)

	if (currentBranch.Name().Short() != "master" && !*noSelectBranch && !*quiet) || *alwaysSelectBranch {
		branch, err := semvertag.PromptForBranch(r)
		if err != nil {
			log.Fatalf("Error selecting branch: %v", err)
		}
		commitToTag, err = semvertag.GetLastCommitForBranch(r, branch)
		if err != nil {
			log.Fatalf("Cannot get last commit to current branch: %v", err)
		}
		branchName = branch.Name().Short()
	} else {
		commitToTag, err = semvertag.GetLastCommitForBranch(r, currentBranch)
		if err != nil {
			log.Fatalf("Cannot get last commit to current branch: %v", err)
		}
	}
	var tagPrefix string
	if *prefix != "" {
		tagPrefix = fmt.Sprintf("%s-", *prefix)
	}

	if branchName != "master" {
		tagPrefix = fmt.Sprintf("%s%s", tagPrefix, branchName)
	}

	newTagName := fmt.Sprintf("%sv%s", tagPrefix, lastVersion.String())
	author, cfgReadErr := semvertag.ReadConfig(r)
	if cfgReadErr != nil {
		log.Printf("Error reading config to get author information: %v", err)
	}
	if author == nil && *quiet && (*username == "" || *email == "") {
		log.Fatalf("Cannot read username/email from config, and configured to quiet, and it's not set via command line..")
	}
	if *username != "" && *email != "" {
		author = &semvertag.AuthorInfo{
			Username: *username,
			Email:    *email,
		}
	} else if author == nil || author.Email == "" || author.Username == "" {

		var name, email string
		if name, err = semvertag.PromptString("author name"); err != nil {
			log.Fatalf("Error prompting name: %v", err)
		}
		if email, err = semvertag.PromptString("author email"); err != nil {
			log.Fatalf("Error prompting email: %v", err)
		}

		author = &semvertag.AuthorInfo{
			Username: name,
			Email:    email,
		}
	}

	var msg = *tagMessage
	if !*quiet {
		if msg == "" {
			msg, err = semvertag.PromptString("Enter tag message")
			if err != nil {
				log.Fatalf("Error prompting message: %v", err)
			}
		}
	}

	if !*quiet {
		ok := semvertag.Prompt(fmt.Sprintf("Will create tag %s on %s, %s(%s) with message '%s'", newTagName, branchName, author.Username, author.Email, msg))
		if !ok {
			log.Printf("Aborted.")
			return
		}
	}

	_, err = r.CreateTag(newTagName, commitToTag, &git.CreateTagOptions{
		Message: msg,
		Tagger: &object.Signature{
			When:  time.Now(),
			Name:  author.Username,
			Email: author.Email,
		},
	})

	if err != nil {
		log.Fatalf("Error creating tag: %v", err)
	}

	if !*noPush {
		semvertag.Execute(dir, "git", "push", "origin", newTagName)
	}

	log.Printf("done.")
}
