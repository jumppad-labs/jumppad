package copy

import (
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	cp "github.com/otiai10/copy"
	"golang.org/x/xerrors"
)

type Provider struct {
	log    clients.Logger
	config *Copy
}

func NewProvider(co *Copy, l clients.Logger) *Provider {
	return &Provider{l, co}
}

func (c *Copy) Create() error {
	c.log.Info("Creating Copy", "ref", c.config.Name, "source", c.config.Source, "destination", c.config.Destination, "perms", c.config.Permissions)

	// Check source exists
	_, err := os.Stat(c.config.Source)
	if err != nil {
		c.log.Debug("Error discovering source directory", "ref", c.config.Name, "source", c.config.Source, "error", err)
		return xerrors.Errorf("unable to find source directory for copy resource, ref=%s: %w", c.config.Name, err)
	}

	// Check the dest exists, if so grab the existing perms
	// so we can set them back after copy
	// copy changes the permissions of the destination for some reason
	originalPerms := os.FileMode(0)
	d, err := os.Stat(c.config.Destination)
	if err == nil && d.IsDir() {
		originalPerms = d.Mode()
	}

	opts := cp.Options{}
	opts.Sync = true

	// keep track of
	files := []string{}
	opts.Skip = func(srcinfo fs.FileInfo, src, dest string) (bool, error) {
		c.log.Debug("Copy file", "ref", c.config.Name, "file", src)

		files = append(files, src)
		return false, nil
	}

	err = cp.Copy(c.config.Source, c.config.Destination, opts)
	if err != nil {
		c.log.Debug("Error copying source directory", "ref", c.config.Name, "source", c.config.Source, "error", err)

		return xerrors.Errorf("unable to copy files, ref=%s: %w", c.config.Name, err)
	}

	c.config.CopiedFiles = files

	// set the permissions
	if c.config.Permissions != "" {
		perms, err := strconv.ParseInt(c.config.Permissions, 8, 64)
		if err != nil {
			c.log.Debug("Invalid destination permissions", "ref", c.config.Name, "permissions", c.config.Permissions, "error", err)
			return xerrors.Errorf("Invalid destination permissions for copy resource, ref=%s %s: %w", c.config.Name, c.config.Permissions, err)
		}

		for _, f := range c.config.CopiedFiles {
			fn := strings.Replace(f, c.config.Source, c.config.Destination, -1)
			c.log.Debug("Setting permissions for file", "ref", c.config.Name, "file", fn, "permissions", c.config.Permissions)

			os.Chmod(fn, os.FileMode(perms))
		}
	}

	if originalPerms != os.FileMode(0) {
		c.log.Debug("Restore original permissions", "ref", c.config.Name, "perms", originalPerms.String())
		os.Chmod(c.config.Destination, originalPerms)
	}

	return nil
}

func (c *Copy) Destroy() error {
	c.log.Info("Destroy Copy", "ref", c.config.Name)

	for _, f := range c.config.CopiedFiles {
		fn := strings.Replace(f, c.config.Source, c.config.Destination, -1)
		c.log.Debug("Remove file", "ref", c.config.Name, "file", fn, "source", c.config.Source, "destination", c.config.Destination)

		// double check that the replacement has worked, we do not want to remove the original
		if fn != f {
			err := os.RemoveAll(fn)
			if err != nil {
				c.log.Debug("Unable to remove file", "ref", c.config.Name, "file", fn)
			}
		}
	}

	return nil
}

func (c *Copy) Lookup() ([]string, error) {
	return nil, nil
}

func (c *Copy) Refresh() error {
	c.log.Debug("Refresh Copied files", "ref", c.config.Name)

	return nil
}

func (c *Copy) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}
