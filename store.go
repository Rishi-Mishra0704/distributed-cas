package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

const DefaultRootFolderName = "GGNetwork"

func CasPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize

		paths[i] = hashStr[from:to]

	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		FileName: hashStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	FileName string
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

type StoreOpts struct {
	// Root is the root directory where the files will be stored
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = DefaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}

var DefaultTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key}
}

func (p PathKey) firstPathName() string {
	paths := strings.Split(p.PathName, "/")

	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)

	_, err := os.Stat(pathKey.FullPath())
	if err == fs.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.firstPathName())
	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.FileName)
	}()
	return os.RemoveAll(firstPathNameWithRoot)
}
func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	PathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, PathKey.FullPath())
	return os.Open(fullPathWithRoot)
}

func (s *Store) WriteStream(key string, r io.Reader) error {
	PathKey := s.PathTransformFunc(key)
	pathKeyWithRoot := fmt.Sprintf("%s/%s", s.Root, PathKey.PathName)
	if err := os.MkdirAll(pathKeyWithRoot, os.ModePerm); err != nil {
		return err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, PathKey.FullPath())

	f, err := os.Create(fullPathWithRoot)
	if err != nil {
		return err
	}

	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written %d bytes to disk: %s", n, fullPathWithRoot)

	return nil
}