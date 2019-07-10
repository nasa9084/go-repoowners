package git_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nasa9084/go-repoowners/internal/pkg/git"
	gogit "gopkg.in/src-d/go-git.v4"
	object "gopkg.in/src-d/go-git.v4/plumbing/object"
)

type fakeRemote struct {
	dir string
}

func newFakeRemote() (*fakeRemote, error) {
	tmp, err := ioutil.TempDir("", "fake")
	if err != nil {
		return nil, err
	}
	return &fakeRemote{
		dir: tmp,
	}, nil
}

func (fr *fakeRemote) clean() {
	os.RemoveAll(fr.dir)
}

func (fr *fakeRemote) mkRepo(domain, org, repo string) error {
	repoDir := filepath.Join(fr.dir, domain, org, repo)
	if err := os.MkdirAll(repoDir, os.ModePerm); err != nil {
		return err
	}
	if _, err := gogit.PlainInit(repoDir, false); err != nil {
		return err
	}
	if err := fr.commit(domain, org, repo, "first"); err != nil {
		return err
	}
	return nil
}

func (fr *fakeRemote) commit(domain, org, repo, filename string) error {
	repoDir := filepath.Join(fr.dir, domain, org, repo)
	r, err := gogit.PlainOpen(repoDir)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(repoDir, filename), []byte(""), os.ModePerm); err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	if _, err := wt.Add(filename); err != nil {
		return err
	}
	usig := &object.Signature{
		Name:  "alice",
		Email: "alice@example.com",
		When:  time.Now(),
	}
	if _, err := wt.Commit(filename+" commit", &gogit.CommitOptions{Author: usig, Committer: usig}); err != nil {
		return err
	}
	return nil
}

func TestClone(t *testing.T) {
	fake, err := newFakeRemote()
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.mkRepo("foo", "bar", "baz"); err != nil {
		t.Fatal(err)
	}
	if err := fake.mkRepo("foo", "bar", "qux"); err != nil {
		t.Fatal(err)
	}

	c, err := git.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Clean()

	repo1, err := c.Clone(filepath.Join(fake.dir, "foo", "bar", "baz"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo1.Clean()

	if err := fake.commit("foo", "bar", "baz", "second"); err != nil {
		t.Fatal(err)
	}
	repo2, err := c.Clone(filepath.Join(fake.dir, "foo", "bar", "baz"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo2.Clean()

	logs, err := repo2.Log()
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Errorf("unexpected number of logs: %d != 2", len(logs))
		t.Log("log:")
		for _, log := range logs {
			t.Logf("\t%s", log)
		}
		return
	}
}
