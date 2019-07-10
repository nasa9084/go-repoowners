package repoowners

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/spf13/afero"
)

func newMemFS() *afero.Afero {
	return &afero.Afero{Fs: afero.NewMemMapFs()}
}

func TestLoadLocalOnlyOneOwners(t *testing.T) {
	const basePath = "src/github.com/nasa9084/test_repository"
	fs = newMemFS()
	if err := fs.MkdirAll(basePath, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := fs.OpenFile(filepath.Join(basePath, DefaultOwnersFilename), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprint(f, `---
approvers:
- alice
- bob
reviewers:
- charlie
- dave
- ellen`)

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	o, err := LoadLocal(basePath)
	if err != nil {
		t.Fatal(err)
	}
	wantApprovers := map[string]UsernameSet{
		".": newUsernameSet("alice", "bob"),
	}
	wantReviewers := map[string]UsernameSet{
		".": newUsernameSet("charlie", "dave", "ellen"),
	}
	if !reflect.DeepEqual(o.approvers, wantApprovers) {
		t.Errorf("unexpected approvers:\n  got:  %+v\n  want: %+v", o.approvers, wantApprovers)
		return
	}
	if !reflect.DeepEqual(o.reviewers, wantReviewers) {
		t.Errorf("unexpected reviewers:\n  got:  %+v\n  want: %+v", o.reviewers, wantReviewers)
		return
	}
}

func TestLoadLocalOnlyAliases(t *testing.T) {
	const basePath = "src/github.com/nasa9084/test_repository"
	fs = newMemFS()
	if err := fs.MkdirAll(basePath, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := fs.OpenFile(filepath.Join(basePath, DefaultAliasesFilename), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(f, `---
aliases:
  managers:
    - alice
  members:
    - bob
    - charlie
    - dave`)

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	o, err := LoadLocal(basePath)
	if err != nil {
		t.Fatal(err)
	}
	wantAliases := map[string]UsernameSet{
		"managers": newUsernameSet("alice"),
		"members":  newUsernameSet("bob", "charlie", "dave"),
	}
	if !reflect.DeepEqual(o.aliases, wantAliases) {
		t.Errorf("unexpected aliases:\n  got:  %+v\n  want: %+v", o.aliases, wantAliases)
		return
	}
}

func TestApprovers(t *testing.T) {
	owners := Owners{
		approvers: map[string]UsernameSet{
			"foo/bar":     newUsernameSet("bob"),
			"foo/bar/baz": newUsernameSet("charlie", "dave"),
			"foo/bar/qux": newUsernameSet("ellen"),
		},
	}
	tests := []struct {
		got  UsernameSet
		want UsernameSet
	}{
		{
			got:  owners.Approvers("foo/bar/baz"),
			want: newUsernameSet("bob", "charlie", "dave"),
		},
		{
			got:  owners.Approvers("foo/bar/qux"),
			want: newUsernameSet("bob", "ellen"),
		},
		{
			got:  owners.Approvers("foo/bar"),
			want: newUsernameSet("bob"),
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("unexpected approvers:\n  got:  %+v\n  want: %+v", tt.got, tt.want)
				return
			}
		})
	}
}

func TestApproversWithNoInheritance(t *testing.T) {
	owners := Owners{
		options: map[string]options{
			"foo/bar/baz": options{
				NoInheritance: true,
			},
		},
		approvers: map[string]UsernameSet{
			"foo/bar":          newUsernameSet("alice"),
			"foo/bar/baz":      newUsernameSet("bob"),
			"foo/bar/qux":      newUsernameSet("charlie"),
			"foo/bar/baz/quux": newUsernameSet("dave"),
		},
	}
	tests := []struct {
		got  UsernameSet
		want UsernameSet
	}{
		{
			got:  owners.Approvers("foo"),
			want: newUsernameSet(),
		},
		{
			got:  owners.Approvers("foo/bar"),
			want: newUsernameSet("alice"),
		},
		{
			got:  owners.Approvers("foo/bar/baz"),
			want: newUsernameSet("bob"),
		},
		{
			got:  owners.Approvers("foo/bar/baz/qux"),
			want: newUsernameSet("bob"),
		},
		{
			got:  owners.Approvers("foo/bar/qux"),
			want: newUsernameSet("alice", "charlie"),
		},
		{
			got:  owners.Approvers("foo/bar/baz/quux"),
			want: newUsernameSet("bob", "dave"),
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("unexpected approvers:\n  got:  %+v\n  want: %+v", tt.got, tt.want)
				return
			}
		})
	}
}

func TestApproversWithAliases(t *testing.T) {
	owners := Owners{
		approvers: map[string]UsernameSet{
			"foo/bar":      newUsernameSet("alice", "admins"),
			"foo/bar/baz":  newUsernameSet("bob", "members"),
			"foo/qux":      newUsernameSet("alice"),
			"foo/qux/quux": newUsernameSet("bob", "admins"),
		},
		aliases: map[string]UsernameSet{
			"admins":  newUsernameSet("charlie"),
			"members": newUsernameSet("dave", "ellen"),
		},
	}
	tests := []struct {
		got  UsernameSet
		want UsernameSet
	}{
		{
			got:  owners.Approvers("foo/bar"),
			want: newUsernameSet("alice", "charlie"),
		},
		{
			got:  owners.Approvers("foo/bar/baz"),
			want: newUsernameSet("alice", "bob", "charlie", "dave", "ellen"),
		},
		{
			got:  owners.Approvers("foo/qux/quux"),
			want: newUsernameSet("alice", "bob", "charlie"),
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("unexpected approvers:\n  got:  %+v\n  want: %+v", tt.got, tt.want)
				return
			}
		})
	}
}

func TestIsApprover(t *testing.T) {
	owners := Owners{
		approvers: map[string]UsernameSet{
			"foo/bar":     newUsernameSet("alice"),
			"foo/bar/baz": newUsernameSet("bob"),
		},
	}
	tests := []struct {
		got  bool
		want bool
	}{
		{
			got:  owners.IsApprover("alice", "foo/bar"),
			want: true,
		},
		{
			got:  owners.IsApprover("alice", "foo"),
			want: false,
		},
		{
			got:  owners.IsApprover("alice", "foo/bar/qux"),
			want: true,
		},
		{
			got:  owners.IsApprover("bob", "foo/bar"),
			want: false,
		},
		{
			got:  owners.IsApprover("bob", "foo/bar/qux"),
			want: false,
		},
		{
			got:  owners.IsApprover("bob", "foo/bar/baz"),
			want: true,
		},
		{
			got:  owners.IsApprover("bob", "foo/bar/baz/qux"),
			want: true,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%t != %t", tt.got, tt.want)
				return
			}
		})
	}
}

func TestParseOwners(t *testing.T) {
	tests := []struct {
		label string
		in    string
		want  ownersConfig
	}{
		{
			label: "only approvers",
			in: `approvers:
- alice
- bob`,
			want: ownersConfig{
				Approvers: []string{"alice", "bob"},
			},
		},
		{
			label: "approvers and reviewers and required reviewers",
			in: `approvers:
- alice
- bob
reviewers:
- charlie
- dave
required_reviewers:
- ellen`,
			want: ownersConfig{
				Approvers:         []string{"alice", "bob"},
				Reviewers:         []string{"charlie", "dave"},
				RequiredReviewers: []string{"ellen"},
			},
		},
		{
			label: "with no_inherit option",
			in: `no_inherit: true
approvers:
- alice
reviewers:
- bob`,
			want: ownersConfig{
				Options: options{
					NoInheritance: true,
				},
				Approvers: []string{"alice"},
				Reviewers: []string{"bob"},
			},
		},
		{
			label: "with comment",
			in: `---
# This is comment
approvers:
  - alice # foo-team
reviewers:
  - bob
  - charlie # bar-team`,
			want: ownersConfig{
				Approvers: []string{"alice"},
				Reviewers: []string{"bob", "charlie"},
			},
		},
		{
			label: "empty yaml document",
			in:    `---`,
			want:  ownersConfig{},
		},
		{
			label: "empty",
			in:    ``,
			want:  ownersConfig{},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i)+"/"+tt.label, func(t *testing.T) {
			got, err := parseOwners(bytes.NewBufferString(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unexpected ownerConfig:\n  got:  %+v\n  want: %+v", got, tt.want)
				return
			}
		})
	}
}

func TestParseAliases(t *testing.T) {
	tests := []struct {
		label string
		in    string
		want  aliasesConfig
	}{
		{
			label: "aliases",
			in: `aliases:
  managers:
  - alice
  - bob`,
			want: aliasesConfig{
				Aliases: map[string][]string{"managers": []string{"alice", "bob"}},
			},
		},
		{
			label: "empty yaml document",
			in:    `---`,
			want:  aliasesConfig{},
		},
		{
			label: "empty",
			in:    ``,
			want:  aliasesConfig{},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i)+"/"+tt.label, func(t *testing.T) {
			got, err := parseAliases(bytes.NewBufferString(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unexpected aliasesConfig:\n  got:  %+v\n  want: %+v", got, tt.want)
				return
			}
		})
	}
}
