package semvertag

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/blang/semver"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func GetLastCommitForBranch(r *git.Repository, branchRef *plumbing.Reference) (plumbing.Hash, error) {
	if !branchRef.Name().IsTag() {
		return branchRef.Hash(), nil
	}

	o, err := r.Object(plumbing.AnyObject, branchRef.Hash())
	if err != nil {
		return plumbing.ZeroHash, err
	}

	switch o := o.(type) {
	case *object.Tag:
		if o.TargetType != plumbing.CommitObject {
			return plumbing.ZeroHash, fmt.Errorf("unsupported tag object target %q", o.TargetType)
		}

		return o.Target, nil
	case *object.Commit:
		return o.Hash, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("unsupported tag target %q", o.Type())
}

type VersionTag struct {
	Tag     *object.Tag
	Version semver.Version
}

func GetSortedMatchingTags(r *git.Repository, prefix string) []*VersionTag {
	tagIter, err := r.Tags()
	if err != nil {
		log.Fatalf("Error iterating tags: %v", err)
	}
	defer tagIter.Close()
	var matchingTags []*VersionTag
	for {
		ref, err := tagIter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading tags: %v", err)
		}
		tag, err := r.TagObject(ref.Hash())
		if err != nil {
			continue
		}

		// if the tag name doesn't have requested prefix, skip
		if !strings.HasPrefix(tag.Name, prefix) {
			continue
		}

		versionName := tag.Name[len(prefix):]
		versionName = strings.TrimPrefix(versionName, "-")

		version, err := semver.ParseTolerant(versionName)
		// invalid semver, so skip it
		if err != nil {
			continue
		}

		matchingTags = append(matchingTags, &VersionTag{
			Tag:     tag,
			Version: version,
		})
	}

	sort.Slice(matchingTags, func(x, y int) bool {
		return matchingTags[y].Tag.Tagger.When.Before(matchingTags[x].Tag.Tagger.When)
	})
	return matchingTags
}
