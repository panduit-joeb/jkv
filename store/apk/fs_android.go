package apk

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/storage/repository"
)

type FileMode uint32

type DirEntry fyne.URI

type FileInfo interface {
	Name() string
	Size() int64
	Mode() FileMode
	ModTime() time.Time
	IsDir() bool
	Sys() any
}

type FileInfoData struct {
	name    string
	size    int64
	mode    FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

type FSOp interface {
	MkdirAll(string, FileMode) error
	Mkdir(string, FileMode) error
	RemoveAll(string) error
	Rmdir(string) error
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte, FileMode) error
	Remove(string) error
	ReadDir(string) ([]DirEntry, error)
	Stat(string) (FileInfo, error)
}

func Mkdir(name string, mode FileMode) (err error) {
	err = storage.CreateListable(storage.NewFileURI(name))
	if err == repository.ErrOperationNotSupported {
		fmt.Println("mkdir", name, "not supported")
		return err
	} else if err != nil {
		fmt.Println("mkdir", name, "failed with err", err.Error())
		return err
	} else {
		fmt.Println("mkdir", name, "succeeded")
		return nil
	}
}

func RemoveAll(name string) (err error) {
	if _, err = Stat(name); err == nil {
		//implies file exists, abort
		return os.ErrExist
	}
	// this is tricky because the top level directory is unknown, for now this method is always called with 2 levels, i.e. jkv_db/hashes or jkv_db/scalars, so we just need to make the jkv_db directory, then the full directory
	u := storage.NewFileURI(name)
	if err := storage.Delete(u); err == nil {
		if baseDir, err := storage.Parent(u); err == nil {
			return storage.Delete(baseDir)
		}
	}
	return err
}

func ReadFile(name string) (data []byte, err error) {
	fURI := storage.NewFileURI(name)
	fmt.Println("trying to read", fURI)
	r, err := storage.Reader(fURI)
	if err == nil {
		n, err := r.Read(data)
		if n == 0 || err != nil {
			fmt.Printf("Reading %s failed, err: %#v\n", name, err.Error())
			return []byte{}, err
		}
		fmt.Printf("Reading %s succeeded, value: %s\n", name, string(data))
		return data, err
	}
	fmt.Printf("getting Reader for %s failed, err: %#v\n", name, err.Error())
	return []byte{}, err
}

func WriteFile(name string, data []byte, mode FileMode) (err error) {
	f := storage.NewFileURI(name)
	w, err := storage.Writer(f)
	if err == nil {
		return func() (err error) { _, err = w.Write(data); return err }()
	}
	fmt.Printf("Writing %s (%s) succeeded, value: %s\n", name, f, string(data))
	return nil
}

func Remove(name string) error { return storage.Delete(storage.NewFileURI(name)) }

func ReadDir(name string) (entries []fyne.URI, err error) {
	return storage.List(storage.NewFileURI(name))
}

func Stat(name string) (f FileInfo, err error) {
	var (
		can, exists bool
		u           = storage.NewFileURI(name)
		fd          FileInfoData
	)
	if can, err = storage.CanList(u); err == nil {
		fd.name = name
		fd.size = 0
		fd.mode = 0755
		fd.modTime = time.Now()
		fd.isDir = can
		fd.sys = nil
		if !can {
			if exists, err = storage.Exists(u); err == nil {
				if exists {
					return fd, err
				}
				return fd, os.ErrNotExist
			}
			return fd, err
		}
		return fd, nil
	}
	return fd, err
}

func (d FileInfoData) Name() string       { return d.name }
func (d FileInfoData) Size() int64        { return d.size }
func (d FileInfoData) Mode() FileMode     { return d.mode }
func (d FileInfoData) ModTime() time.Time { return d.modTime }
func (d FileInfoData) IsDir() bool        { return d.isDir }
func (d FileInfoData) Sys() any           { return d.sys }
