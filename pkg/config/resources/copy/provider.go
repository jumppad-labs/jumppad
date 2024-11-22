package copy

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/getter"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	cp "github.com/otiai10/copy"
	"golang.org/x/xerrors"
)

type Provider struct {
	log    sdk.Logger
	config *Copy
	getter getter.Getter
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Copy)
	if !ok {
		return fmt.Errorf("unable to initialize Copy provider, resource is not an instance of Copy")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.getter = cli.Getter
	p.config = c
	p.log = l

	return nil
}

func (p *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cacncelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Copy", "ref", p.config.Meta.Name, "source", p.config.Source, "destination", p.config.Destination, "perms", p.config.Permissions)

	srcPath := p.config.Source

	// are we copying an existing directory or downloading?
	_, err := os.Stat(srcPath)

	if err != nil {
		tempPath := filepath.Join(utils.JumppadTemp(), "copy", p.config.Meta.ID)

		defer func() {
			// clean up temporary files
			err := os.RemoveAll(tempPath)
			if err != nil {
				p.log.Warn("Error removing temporary files", "ref", p.config.Meta.Name, "path", tempPath, "error", err)
			}
		}()

		err := p.getter.Get(p.config.Source, tempPath)
		if err != nil {
			return fmt.Errorf("error getting source from %s: %v", p.config.Source, err)
		}

		// Check source exists
		_, err = os.Stat(tempPath)
		if err != nil {
			p.log.Debug("Error fetching source directory", "ref", p.config.Meta.ID, "source", tempPath, "error", err)
			return xerrors.Errorf("unable to find source directory for copy resource, ref=%s: %w", p.config.Meta.ID, err)
		}

		srcPath = tempPath
	}

	// Check the dest exists, if so grab the existing perms
	// so we can set them back after copy
	// copy changes the permissions of the destination for some reason
	originalPerms := os.FileMode(0)
	d, err := os.Stat(p.config.Destination)
	if err == nil && d.IsDir() {
		originalPerms = d.Mode()
	}

	opts := cp.Options{}
	opts.Sync = true

	// keep track of
	files := []string{}
	opts.Skip = func(srcinfo fs.FileInfo, src, dest string) (bool, error) {
		p.log.Debug("Copy file", "ref", p.config.Meta.Name, "file", src)

		files = append(files, src)
		return false, nil
	}

	err = cp.Copy(srcPath, p.config.Destination, opts)
	if err != nil {
		p.log.Debug("Error copying source directory", "ref", p.config.Meta.Name, "source", srcPath, "error", err)

		return xerrors.Errorf("unable to copy files, ref=%s: %w", p.config.Meta.Name, err)
	}

	p.config.CopiedFiles = files

	// set the permissions
	if p.config.Permissions != "" {
		perms, err := strconv.ParseInt(p.config.Permissions, 8, 64)
		if err != nil {
			p.log.Debug("Invalid destination permissions", "ref", p.config.Meta.Name, "permissions", p.config.Permissions, "error", err)
			return xerrors.Errorf("invalid destination permissions for copy resource, ref=%s %s: %w", p.config.Meta.Name, p.config.Permissions, err)
		}

		for _, f := range p.config.CopiedFiles {
			fn := strings.Replace(f, srcPath, p.config.Destination, -1)
			p.log.Debug("Setting permissions for file", "ref", p.config.Meta.Name, "file", fn, "permissions", p.config.Permissions)

			os.Chmod(fn, os.FileMode(perms))
		}
	}

	if originalPerms != os.FileMode(0) {
		p.log.Debug("Restore original permissions", "ref", p.config.Meta.Name, "perms", originalPerms.String())
		os.Chmod(p.config.Destination, originalPerms)
	}

	return nil
}

func (p *Provider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cacncelled, skipping destroy", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Copy", "ref", p.config.Meta.Name)

	for _, f := range p.config.CopiedFiles {
		fn := strings.Replace(f, p.config.Source, p.config.Destination, -1)
		p.log.Debug("Remove file", "ref", p.config.Meta.Name, "file", fn, "source", p.config.Source, "destination", p.config.Destination)

		// double check that the replacement has worked, we do not want to remove the original
		if fn != f {
			err := os.RemoveAll(fn)
			if err != nil {
				p.log.Debug("Unable to remove file", "ref", p.config.Meta.Name, "file", fn)
			}
		}
	}

	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	p.log.Debug("Refresh Copied files", "ref", p.config.Meta.Name)
	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.Name)
	return false, nil
}
