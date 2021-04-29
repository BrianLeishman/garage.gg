package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
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

func clone() (fs billy.Filesystem, r *git.Repository, w *git.Worktree, err error) {
	fs = memfs.New()

	r, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:   os.Getenv("REPO"),
		Depth: 1,
	})
	if err != nil {
		err = errors.Wrapf(err, "failed to clone repo to memory")
		return
	}

	w, err = r.Worktree()
	if err != nil {
		err = errors.Wrapf(err, "failed to get worktree")
		return
	}

	return
}

func commit(c committer) error {
	fs, r, w, err := clone()
	if err != nil {
		return err
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

	return push(filename, r, w)
}

func delete(c committer) error {
	fs, r, w, err := clone()
	if err != nil {
		return err
	}

	dir := c.filepath()

	filename := filepath.Join(dir, c.filename())
	fs.Remove(filename)

	return push(filename, r, w)
}

func push(filename string, r *git.Repository, w *git.Worktree) error {
	s, _ := w.Status()
	if s.IsClean() {
		return nil
	}

	_, err := w.Add(filename)
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
