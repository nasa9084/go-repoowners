package repoowners

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultOwnersFilename  = "OWNERS"
	DefaultAliasesFilename = "OWNERS_ALIASES"
)

type Owners struct {
	approvers         map[string][]string
	reviewers         map[string][]string
	requiredReviewers map[string][]string
	options           map[string]options

	memoizedApprovers         memo
	memoizedReviewers         memo
	memoizedRequiredReviewers memo
}

type memo struct {
	sync.Map
}

func (m *memo) store(path string, list []string) {
	m.Map.Store(path, list)
}

func (m memo) load(path string) []string {
	val, ok := m.Map.Load(path)
	if ok {
		return val.([]string)
	}
	return nil
}

func (o Owners) entries(path string, mp map[string][]string, opts options) []string {
	ret := []string{}
	for {
		pp, ok := mp[path]
		if ok {
			ret = append(ret, pp...)
		}
		if opts.NoInheritance {
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
	sort.Strings(ret)
	return ret
}

func (o *Owners) Approvers(path string) []string {
	if approvers := o.memoizedApprovers.load(path); approvers != nil {
		return approvers
	}
	approvers := o.entries(path, o.approvers, o.options[path])
	o.memoizedApprovers.store(path, approvers)
	return approvers
}

func (o *Owners) IsApprover(user, path string) bool {
	approvers := o.Approvers(path)
	return isIn(user, approvers)
}

func (o *Owners) Reviewers(path string) []string {
	if reviewers := o.memoizedReviewers.load(path); reviewers != nil {
		return reviewers
	}
	reviewers := o.entries(path, o.reviewers, o.options[path])
	o.memoizedReviewers.store(path, reviewers)
	return reviewers
}

func (o *Owners) IsReviewer(user, path string) bool {
	reviewers := o.Reviewers(path)
	return isIn(user, reviewers)
}

func (o *Owners) RequiredReviewers(path string) []string {
	if requiredReviewers := o.memoizedRequiredReviewers.load(path); requiredReviewers != nil {
		return requiredReviewers
	}
	requiredReviewers := o.entries(path, o.requiredReviewers, o.options[path])
	o.memoizedRequiredReviewers.store(path, requiredReviewers)
	return requiredReviewers
}

func (o *Owners) IsRequiredReviewer(user, path string) bool {
	requiredReviewers := o.RequiredReviewers(path)
	return isIn(user, requiredReviewers)
}

func isIn(val string, slice []string) bool {
	for _, target := range slice {
		if target == val {
			return true
		}
	}
	return false
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

func (o ownersConfig) repr() string {
	var buf strings.Builder
	fmt.Fprint(&buf, "ownersConfig{")
	fmt.Fprintf(&buf, "\n\tOptions: options{")
	fmt.Fprintf(&buf, "\n\t\tNoInheritance: %t,", o.Options.NoInheritance)
	fmt.Fprint(&buf, "\n\t},")
	fmt.Fprint(&buf, "\n\tApprovers: []string{")
	for _, approver := range o.Approvers {
		fmt.Fprintf(&buf, "\n\t\t%s,", strconv.Quote(approver))
	}
	fmt.Fprint(&buf, "\n\t},")
	fmt.Fprint(&buf, "\n\tReviewers: []string{")
	for _, reviewer := range o.Reviewers {
		fmt.Fprintf(&buf, "\n\t\t%s,", strconv.Quote(reviewer))
	}
	fmt.Fprint(&buf, "\n\t},")
	fmt.Fprint(&buf, "\n}")
	return buf.String()
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
