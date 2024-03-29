package repoowners

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nasa9084/go-repoowners/internal/pkg/git"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultOwnersFilename  = "OWNERS"
	DefaultAliasesFilename = "OWNERS_ALIASES"
)

var fs = &afero.Afero{Fs: afero.NewOsFs()}
var gc *git.Client

func init() {
	c, err := git.NewClient()
	if err != nil {
		panic(err)
	}
	gc = c
}

// Owners holds Owners configuration for one repository.
type Owners struct {
	// these are path: UsernameSet mapping
	approvers         map[string]UsernameSet
	reviewers         map[string]UsernameSet
	requiredReviewers map[string]UsernameSet

	// path: options mapping
	options map[string]options

	// aliasname: []username mapping
	aliases map[string]UsernameSet

	memoizedApprovers         memo
	memoizedReviewers         memo
	memoizedRequiredReviewers memo

	// base is a base path of repository.
	base string
}

func newOwners() Owners {
	return Owners{
		approvers:         map[string]UsernameSet{},
		reviewers:         map[string]UsernameSet{},
		requiredReviewers: map[string]UsernameSet{},
		options:           map[string]options{},
		aliases:           map[string]UsernameSet{},
	}
}

func LoadRemote(domain, org, repo, branch string) (Owners, error) {
	r, err := gc.Clone(fmt.Sprintf("%s/%s/%s:%s", domain, org, repo, branch))
	if err != nil {
		return Owners{}, err
	}
	return LoadLocal(r.Dir)
}

func LoadLocal(basePath string) (Owners, error) {
	o := newOwners()
	o.base = basePath

	if _, err := fs.Stat(filepath.Join(basePath, DefaultAliasesFilename)); err == nil {
		f, err := fs.Open(filepath.Join(basePath, DefaultAliasesFilename))
		if err != nil {
			return Owners{}, err
		}
		defer f.Close()

		ac, err := parseAliases(f)
		if err != nil {
			return Owners{}, err
		}
		for alias, list := range ac.Aliases {
			o.aliases[alias] = newUsernameSet(list...)
		}
	}
	if err := fs.Walk(o.base, o.walkFunc); err != nil {
		return Owners{}, err
	}

	return o, nil
}

type memo struct {
	sync.Map
}

func (m *memo) store(path string, set UsernameSet) {
	m.Map.Store(path, set)
}

func (m memo) load(path string) UsernameSet {
	val, ok := m.Map.Load(path)
	if ok {
		return val.(UsernameSet)
	}
	return nil
}

func (o *Owners) walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return nil
	}
	fn := filepath.Base(path)
	relPath, err := filepath.Rel(o.base, path)
	if err != nil {
		return err
	}
	relPathDir := filepath.Dir(relPath)
	if info.Mode().IsDir() || !info.Mode().IsRegular() {
		return nil
	}
	if fn != DefaultOwnersFilename {
		return nil
	}

	f, err := fs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	oc, err := parseOwners(f)
	if err != nil {
		return err
	}
	o.applyOwnersConfig(relPathDir, oc)
	return nil
}

func (o *Owners) applyOwnersConfig(path string, oc ownersConfig) {
	if len(oc.Approvers) > 0 {
		o.approvers[path] = newUsernameSet(oc.Approvers...)
	}
	if len(oc.Reviewers) > 0 {
		o.reviewers[path] = newUsernameSet(oc.Reviewers...)
	}
	if len(oc.RequiredReviewers) > 0 {
		o.requiredReviewers[path] = newUsernameSet(oc.RequiredReviewers...)
	}
	o.options[path] = oc.Options
}

func (o Owners) entries(path string, mp map[string]UsernameSet, opts map[string]options) UsernameSet {
	ret := UsernameSet{}
	for {
		us, ok := mp[path]
		if ok {
			ret = ret.Union(us)
		}
		if opts[path].NoInheritance {
			break
		}
		if path == "" {
			break
		}
		path = filepath.Dir(path)
		if path == "." {
			path = ""
		}
		path = strings.TrimSuffix(path, "/")
	}
	ret = o.expandAliases(ret)
	return ret
}

func (o *Owners) expandAliases(usernames UsernameSet) UsernameSet {
	usernames = usernames.Copy()
	for _, username := range usernames.List() {
		if expanded, ok := o.aliases[username]; ok {
			usernames.Delete(username)
			usernames = usernames.Union(expanded)
		}
	}
	return usernames
}

// Approvers returns a set of approvers for given file path.
func (o *Owners) Approvers(path string) UsernameSet {
	if approvers := o.memoizedApprovers.load(path); approvers != nil {
		return approvers
	}
	approvers := o.entries(path, o.approvers, o.options)
	o.memoizedApprovers.store(path, approvers)
	return approvers
}

// IsApprover returns true if given user is an approver for given file path.
func (o *Owners) IsApprover(user, path string) bool {
	approvers := o.Approvers(path)
	return approvers.Has(user)
}

// Reviewers returns a set of reviewers for given file path.
func (o *Owners) Reviewers(path string) UsernameSet {
	if reviewers := o.memoizedReviewers.load(path); reviewers != nil {
		return reviewers
	}
	reviewers := o.entries(path, o.reviewers, o.options)
	o.memoizedReviewers.store(path, reviewers)
	return reviewers
}

// IsReviewer returns true if given user is a reviewer for given file path.
func (o *Owners) IsReviewer(user, path string) bool {
	reviewers := o.Reviewers(path)
	return reviewers.Has(user)
}

// RequiredReviewers returns a set of required reviewers for given file path.
func (o *Owners) RequiredReviewers(path string) UsernameSet {
	if requiredReviewers := o.memoizedRequiredReviewers.load(path); requiredReviewers != nil {
		return requiredReviewers
	}
	requiredReviewers := o.entries(path, o.requiredReviewers, o.options)
	o.memoizedRequiredReviewers.store(path, requiredReviewers)
	return requiredReviewers
}

// IsRequiredReviewer returns true if given user is a required reviewer for given path.
func (o *Owners) IsRequiredReviewer(user, path string) bool {
	requiredReviewers := o.RequiredReviewers(path)
	return requiredReviewers.Has(user)
}

type options struct {
	NoInheritance bool `yaml:"no_inherit,omitempty"`
}

type ownersConfig struct {
	Options           options  `yaml:",inline"`
	Approvers         []string `yaml:"approvers,omitempty"`
	Reviewers         []string `yaml:"reviewers,omitempty"`
	RequiredReviewers []string `yaml:"required_reviewers,omitempty"`
}

type aliasesConfig struct {
	Aliases map[string][]string `yaml:"aliases"`
}

func parseOwners(r io.Reader) (ownersConfig, error) {
	var o ownersConfig
	if err := yaml.NewDecoder(r).Decode(&o); err != nil {
		if err == io.EOF {
			return ownersConfig{}, nil
		}
		return ownersConfig{}, err
	}
	return o, nil
}

func parseAliases(r io.Reader) (aliasesConfig, error) {
	var a aliasesConfig
	if err := yaml.NewDecoder(r).Decode(&a); err != nil {
		if err == io.EOF {
			return aliasesConfig{}, nil
		}
		return aliasesConfig{}, err
	}
	return a, nil
}
