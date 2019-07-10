package repoowners

import (
	"sort"
	"strconv"
	"strings"
)

// UsernameSet is a set type for usernames.
type UsernameSet map[string]struct{}

func newUsernameSet(usernames ...string) UsernameSet {
	us := UsernameSet{}
	us.Add(usernames...)
	return us
}

// Add given usernames to the set.
// This function is mutable.
func (us UsernameSet) Add(usernames ...string) {
	for _, username := range usernames {
		us[username] = struct{}{}
	}
}

// Delete given usernames from the set.
// This function is mutable.
func (us UsernameSet) Delete(usernames ...string) {
	for _, username := range usernames {
		delete(us, username)
	}
}

// Union get a new set which contains the members of the set
// and the members of given set.
// This function is immutable.
func (us UsernameSet) Union(us2 UsernameSet) UsernameSet {
	result := UsernameSet{}
	for k := range us {
		result.Add(k)
	}
	for k := range us2 {
		result.Add(k)
	}
	return result
}

// Has returns true if given username is a member of the set.
func (us UsernameSet) Has(username string) bool {
	_, has := us[username]
	return has
}

// List returns a sorted list which contains members of set.
func (us UsernameSet) List() []string {
	ret := make([]string, 0, len(us))
	for k := range us {
		ret = append(ret, k)
	}
	return ret
}

// Copy returns a new copy of UsernameSet.
func (us UsernameSet) Copy() UsernameSet {
	return us.Union(nil)
}

// Pop returns a username from the set and remove it.
// Return true if the returned value at first return,
// is a member of the set, otherwise false as second
// return value..
func (us UsernameSet) Pop() (string, bool) {
	for key := range us {
		us.Delete(key)
		return key, true
	}
	return "", false
}

func (us UsernameSet) String() string {
	var buf strings.Builder
	buf.WriteString("{")
	usernames := us.List()
	for i := range usernames {
		usernames[i] = strconv.Quote(usernames[i])
	}
	sort.Strings(usernames)
	buf.WriteString(strings.Join(usernames, ", "))
	buf.WriteString("}")
	return buf.String()
}
