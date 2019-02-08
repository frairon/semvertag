package semvertag

import (
	"fmt"
	"io"
	"log"

	"github.com/manifoldco/promptui"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func Prompt(msg string) {

	prompt := promptui.Prompt{
		Label:     msg,
		IsConfirm: true,
		Default:   "y",
	}

	_, err := prompt.Run()
	if err != nil {
		log.Fatalf("Operation aborted.")
	}
}

func PromptString(msg string) (string, error) {
	prompt := promptui.Prompt{
		Label: msg,
	}
	msg, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return msg, nil
}

func PromptForBranch(r *git.Repository) (*plumbing.Reference, error) {
	branchIter, err := r.Branches()
	if err != nil {
		return nil, fmt.Errorf("Error listing branches for repo: %v", err)
	}
	defer branchIter.Close()
	var branches []*plumbing.Reference
	for {
		branch, err := branchIter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Error iterating branches: %v", err)
		}
		branches = append(branches, branch)
	}

	prompt := promptui.Select{
		Label: "Select Branch",
		Items: branches,
		Size:  15,
		Templates: &promptui.SelectTemplates{
			Active:   "\U00002022 {{ .Name.Short | cyan }}",
			Inactive: "  {{ .Name.Short | cyan }}",
			Selected: "\U00002022 {{ .Name.Short | red | cyan }}",
		},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return branches[i], nil
}
