package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TarGzOptions struct {
	// OmitRoot when set to true ignores the top level directory in the tar archive
	// only adding sub directories and files.
	OmitRoot bool
}

type TarGz struct {
}

// /folder/foo/bar/test.txt -> /test.txt
// /folder/foo/bar/baz/* -> /baz

func (tg *TarGz) Compress(buf io.Writer, options *TarGzOptions, src ...string) error {
	if options == nil {
		options = &TarGzOptions{}
	}

	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	for _, path := range src {
		// calculate the root folder
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		// the top level is the name of the folder from the src path
		topLevel := filepath.Dir(path)
		if fi.IsDir() && options.OmitRoot {
			topLevel = path
		}

		// walk through every file in the folder
		filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
			// generate tar header
			header, err := tar.FileInfoHeader(fi, strings.Replace(file, topLevel, "", -1))
			if err != nil {
				return err
			}

			// set the filename as a relative path
			// remove the leading / if it exists
			name := filepath.ToSlash(strings.Replace(file, topLevel, "", -1))
			name = strings.TrimLeft(name, "/")
			header.Name = name

			// if the header has an empty name, skip it
			if header.Name == "" {
				return nil
			}

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, data); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}

func (tg *TarGz) Uncompress(src io.Reader, gziped bool, dst string) error {
	var zr io.Reader = src
	var err error

	if gziped {
		// ungzip
		zr, err = gzip.NewReader(src)
		if err != nil {
			return err
		}
	}

	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := ""

		// validate name against path traversal
		if !tg.validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q\n", target)
		}

		// add dst + re-format slashes according to system
		target = filepath.Join(dst, header.Name)
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		// check the type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(fileToWrite, tr); err != nil {
				return err
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	return nil
}

// check for path traversal and correct forward slashes
func (tg *TarGz) validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
