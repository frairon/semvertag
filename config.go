package semvertag

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"

	format "gopkg.in/src-d/go-git.v4/plumbing/format/config"
)

type AuthorInfo struct {
	Username string
	Email    string
}

func findInRaw(raw *format.Config, section, key string) string {
	for _, s := range raw.Sections {
		if !s.IsName(section) {
			continue
		}
		return s.Option(key)
	}
	return ""
}

func ReadConfig(r *git.Repository) (*AuthorInfo, error) {
	var ai AuthorInfo
	cfg, _ := r.Config()
	ai.Username = findInRaw(cfg.Raw, "user", "name")
	ai.Email = findInRaw(cfg.Raw, "user", "email")

	if ai.Username != "" || ai.Email != "" {
		return &ai, nil
	}

	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("Cannot find current user: %v", err)
	}
	globalCfgFile, err := os.Open(path.Join(usr.HomeDir, ".gitconfig"))
	if err != nil {
		return nil, fmt.Errorf("Cannot open .gitconfig in the user's home: %v", err)
	}
	b, err := ioutil.ReadAll(globalCfgFile)
	if err != nil {
		return nil, fmt.Errorf("Cannot read the git config file: %v", err)
	}
	globalCfg := config.NewConfig()
	if err = globalCfg.Unmarshal(b); err != nil {
		return nil, fmt.Errorf("Cannot unmarshal into a config file: %v", err)
	}
	ai.Username = findInRaw(globalCfg.Raw, "user", "name")
	ai.Email = findInRaw(globalCfg.Raw, "user", "email")
	if ai.Username != "" || ai.Email != "" {
		return &ai, nil
	}

	return nil, fmt.Errorf("Invalid/Incomplete author info")
}
