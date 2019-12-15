package main

import (
	"archive/tar"
	"io"
	"ioutil"
	"testing"
)

func TestTar(t *testing.T) {
	// create a temp file
	tf, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		panic(err)
	}
	defer tf.Close()

	// write the file from Docker
	tf.Seek(0, 0)

	// create a tar image file
	tif, err := ioutil.TempFile("", "*.tar")
	if err != nil {
		panic(err)
	}
	defer tif.Close()

	// create a new tar writer
	tw := tar.NewWriter(tif)

	// add the stream from docker as a file
	hdr, _ := tar.FileInfoHeader(tf.Stat(), "image.tar")
	tw.WriteHeader(hdr)

	io.Copy(tw, tf)
	tw.Close()
}
