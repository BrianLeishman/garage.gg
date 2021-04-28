package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	transport "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
)

type committer interface {
	filepath() string
	filename() string
	markdown() []byte
}

func commit(c committer) error {
	fs := memfs.New()

	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:   os.Getenv("REPO"),
		Depth: 1,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to clone repo to memory")
	}

	w, err := r.Worktree()
	if err != nil {
		return errors.Wrapf(err, "failed to get worktree")
	}

	dir := c.filepath()
	err = fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir")
	}

	filename := filepath.Join(dir, c.filename())
	f, err := fs.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to create file in memory")
	}

	_, err = f.Write(c.markdown())
	if err != nil {
		return errors.Wrapf(err, "failed to write file in memory")
	}

	_, err = w.Add(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to git-add new file")
	}

	_, err = w.Commit(filename, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Admin API",
			Email: "brian@garage.gg",
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to commit new file")
	}

	auth := &transport.BasicAuth{
		Username: os.Getenv("GH_USER"),
		Password: os.Getenv("GH_PAT"),
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to push new file")
	}

	return nil
}
