package repoowners

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"
)

func TestApprovers(t *testing.T) {
	owners := Owners{
		approvers: map[string][]string{
			"foo/bar":     []string{"bob"},
			"foo/bar/baz": []string{"charlie", "dave"},
			"foo/bar/qux": []string{"ellen"},
		},
	}
	tests := []struct {
		got  []string
		want []string
	}{
		{
			got:  owners.Approvers("foo/bar/baz"),
			want: []string{"bob", "charlie", "dave"},
		},
		{
			got:  owners.Approvers("foo/bar/qux"),
			want: []string{"bob", "ellen"},
		},
		{
			got:  owners.Approvers("foo/bar"),
			want: []string{"bob"},
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
