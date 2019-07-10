package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Client struct {
	cacheDir string

	rlm       sync.Mutex
	repoLocks map[string]*sync.Mutex
}

func NewClient() (*Client, error) {
	cacheDir, err := ioutil.TempDir("", "git-cache")
	if err != nil {
		return nil, err
	}
	c := &Client{
		cacheDir:  cacheDir,
		repoLocks: map[string]*sync.Mutex{},
	}
	return c, nil
}

// Clean removes the local repository caches.
func (c Client) Clean() error {
	return os.RemoveAll(c.cacheDir)
}

type Repository struct {
	Dir string

	repo    string
	gitRepo *git.Repository
}

func (repo *Repository) Clean() error {
	return os.RemoveAll(repo.Dir)
}

func (repo *Repository) Log() ([]string, error) {
	iter, err := repo.gitRepo.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}
	var ret []string
	iter.ForEach(
		func(commit *object.Commit) error {
			ret = append(ret, commit.Message)
			return nil
		},
	)
	return ret, nil
}

func (c *Client) Clone(repo string) (*Repository, error) {
	c.lockRepo(repo)
	defer c.unlockRepo(repo)

	cache := filepath.Join(c.cacheDir, repo) + ".git"

	// caching bare repository
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		// no cache
		_, err := git.PlainClone(cache, true, &git.CloneOptions{
			URL: repo,
		})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// there is cache
		r, err := git.PlainOpen(cache)
		if err != nil {
			return nil, err
		}
		refSpecs := []config.RefSpec{"refs/heads/*:refs/heads/*"}
		if err := r.Fetch(&git.FetchOptions{RefSpecs: refSpecs}); err != nil {
			if err != git.NoErrAlreadyUpToDate {
				return nil, err
			}
		}
	}

	t, err := ioutil.TempDir("", "git")
	if err != nil {
		return nil, err
	}
	gr, err := git.PlainClone(t, false, &git.CloneOptions{
		URL: cache,
	})
	if err != nil {
		return nil, err
	}
	return &Repository{
		Dir:     t,
		repo:    repo,
		gitRepo: gr,
	}, nil
}

func (c *Client) lockRepo(repo string) {
	c.rlm.Lock()
	defer c.rlm.Unlock()
	if _, ok := c.repoLocks[repo]; !ok {
		c.repoLocks[repo] = &sync.Mutex{}
	}
	c.repoLocks[repo].Lock()
}

func (c *Client) unlockRepo(repo string) {
	c.rlm.Lock()
	defer c.rlm.Unlock()
	c.repoLocks[repo].Unlock()
}
