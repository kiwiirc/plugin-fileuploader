// Slightly modified version of the tusd/filestore implementation.
// Splits file storage into subdirectories based on the hash prefix.
// based on https://github.com/tus/tusd/blob/966f1d51639d3405b630e4c94c0b1d76a0f7475c/filestore/filestore.go
package shardedfilestore

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tus/tusd"
	"github.com/tus/tusd/uid"

	"gopkg.in/Acconut/lockfile.v1"
)

var defaultFilePerm = os.FileMode(0664)
var defaultDirectoryPerm = os.FileMode(0775)

// See the tusd.DataStore interface for documentation about the different
// methods.
type ShardedFileStore struct {
	// Relative or absolute path to store files in.
	BasePath string
	// Number of extra directory layers to prefix file paths with.
	PrefixShardLayers int
}

// New creates a new file based storage backend. The directory specified will
// be used as the only storage entry. This method does not check
// whether the path exists, use os.MkdirAll to ensure.
// In addition, a locking mechanism is provided.
func New(basePath string, prefixShardLayers int) ShardedFileStore {
	return ShardedFileStore{basePath, prefixShardLayers}
}

// UseIn sets this store as the core data store in the passed composer and adds
// all possible extension to it.
func (store ShardedFileStore) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(store)
	composer.UseGetReader(store)
	composer.UseTerminater(store)
	composer.UseLocker(store)
	composer.UseConcater(store)
}

func (store ShardedFileStore) NewUpload(info tusd.FileInfo) (id string, err error) {
	id = uid.Uid()
	info.ID = id

	// Create the prefix directory stucture if needed
	err = os.MkdirAll(store.path(id), defaultDirectoryPerm)
	if err != nil {
		return "", err
	}

	// Create .bin file with no content
	file, err := os.OpenFile(store.binPath(id), os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// writeInfo creates the file by itself if necessary
	err = store.writeInfo(id, info)
	return
}

func (store ShardedFileStore) WriteChunk(id string, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(store.binPath(id), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, src)
	return n, err
}

func (store ShardedFileStore) GetInfo(id string) (tusd.FileInfo, error) {
	info := tusd.FileInfo{}
	data, err := ioutil.ReadFile(store.infoPath(id))
	if err != nil {
		return info, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return info, err
	}

	stat, err := os.Stat(store.binPath(id))
	if err != nil {
		return info, err
	}

	info.Offset = stat.Size()

	return info, nil
}

func (store ShardedFileStore) GetReader(id string) (io.Reader, error) {
	return os.Open(store.binPath(id))
}

func (store ShardedFileStore) Terminate(id string) error {
	if err := os.Remove(store.infoPath(id)); err != nil {
		return err
	}
	if err := os.Remove(store.binPath(id)); err != nil {
		return err
	}
	return nil
}

func (store ShardedFileStore) ConcatUploads(dest string, uploads []string) (err error) {
	file, err := os.OpenFile(store.binPath(dest), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, id := range uploads {
		src, err := store.GetReader(id)
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, src); err != nil {
			return err
		}
	}

	return
}

func (store ShardedFileStore) LockUpload(id string) error {
	lock, err := store.newLock(id)
	if err != nil {
		return err
	}

	err = lock.TryLock()
	if err == lockfile.ErrBusy {
		return tusd.ErrFileLocked
	}

	return err
}

func (store ShardedFileStore) UnlockUpload(id string) error {
	lock, err := store.newLock(id)
	if err != nil {
		return err
	}

	err = lock.Unlock()

	// A "no such file or directory" will be returned if no lockfile was found.
	// Since this means that the file has never been locked, we drop the error
	// and continue as if nothing happened.
	if os.IsNotExist(err) {
		err = nil
	}

	return err
}

// newLock contructs a new Lockfile instance.
func (store ShardedFileStore) newLock(id string) (lockfile.Lockfile, error) {
	path, err := filepath.Abs(filepath.Join(store.path(id), id+".lock"))
	if err != nil {
		return lockfile.Lockfile(""), err
	}

	// We use Lockfile directly instead of lockfile.New to bypass the unnecessary
	// check whether the provided path is absolute since we just resolved it
	// on our own.
	return lockfile.Lockfile(path), nil
}

func (store ShardedFileStore) path(id string) string {
	if len(id) < store.PrefixShardLayers {
		panic("id is too short for requested number of shard layers")
	}
	shards := make([]string, store.PrefixShardLayers+1)
	shards[0] = store.BasePath
	for n, char := range id[:store.PrefixShardLayers] {
		shards[n+1] = string(char)
	}
	return filepath.Join(shards...)
}

// binPath returns the path to the .bin storing the binary data.
func (store ShardedFileStore) binPath(id string) string {
	return filepath.Join(store.path(id), id+".bin")
}

// infoPath returns the path to the .info file storing the file's info.
func (store ShardedFileStore) infoPath(id string) string {
	return filepath.Join(store.path(id), id+".info")
}

// writeInfo updates the entire information. Everything will be overwritten.
func (store ShardedFileStore) writeInfo(id string, info tusd.FileInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(store.infoPath(id), data, defaultFilePerm)
}
