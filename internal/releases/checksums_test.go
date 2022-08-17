package releases

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bep/workers"
	qt "github.com/frankban/quicktest"
)

func TestCreateChecksumLines(t *testing.T) {
	c := qt.New(t)

	w := workers.New(runtime.NumCPU())

	tempDir := t.TempDir()

	subDir := filepath.Join(tempDir, "sub")
	err := os.Mkdir(subDir, 0755)
	c.Assert(err, qt.IsNil)

	var filenames []string

	for i := 0; i < 10; i++ {
		filename := filepath.Join(subDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(filename, []byte(fmt.Sprintf("hello%d", i)), 0644)
		c.Assert(err, qt.IsNil)
		filenames = append(filenames, filename)
	}

	checksums, err := CreateChecksumLines(w, filenames...)
	c.Assert(err, qt.IsNil)
	c.Assert(checksums, qt.DeepEquals, []string{
		"196373310827669cb58f4c688eb27aabc40e600dc98615bd329f410ab7430cff  file6.txt",
		"47ea70cf08872bdb4afad3432b01d963ac7d165f6b575cd72ef47498f4459a90  file3.txt",
		"4e74512f1d8e5016f7a9d9eaebbeedb1549fed5b63428b736eecfea98292d75f  file9.txt",
		"5a936ee19a0cf3c70d8cb0006111b7a52f45ec01703e0af8cdc8c6d81ac5850c  file0.txt",
		"5d9dad16709372200908eecb6a67541ba4013bf7490ccb40d8b75832a1b4aca0  file7.txt",
		"87298cc2f31fba73181ea2a9e6ef10dce21ed95e98bdac9c4e1504ea16f486e4  file2.txt",
		"8dfe82d9a72ad831e48e524a38ad111f206ef08c39aa5847db26df034ee3b57d  file5.txt",
		"91e9240f415223982edc345532630710e94a7f52cd5f48f5ee1afc555078f0ab  file1.txt",
		"bd4c6c665a1b8b4745bcfd3d744ea37488237108681a8ba4486a76126327d3f2  file8.txt",
		"e361a57a7406adee653f1dcff660d84f0ca302907747af2a387f67821acfce33  file4.txt"})

}
