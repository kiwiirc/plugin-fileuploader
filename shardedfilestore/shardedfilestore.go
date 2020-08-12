// Package shardedfilestore is a modified version of the tusd/filestore implementation.
// Splits file storage into subdirectories based on the hash prefix.
// based on https://github.com/tus/tusd/blob/966f1d51639d3405b630e4c94c0b1d76a0f7475c/filestore/filestore.go
package shardedfilestore

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	"github.com/kiwiirc/plugin-fileuploader/db"
	_ "github.com/mattn/go-sqlite3" // register SQL driver
	"github.com/rs/zerolog"
	lockfile "gopkg.in/Acconut/lockfile.v1"

	"github.com/tus/tusd"
	"github.com/tus/tusd/uid"
)

var defaultFilePerm = os.FileMode(0664)
var defaultDirectoryPerm = os.FileMode(0775)

// ShardedFileStore implements various tusd.DataStore-related interfaces.
// See the interfaces for more documentation about the different methods.
type ShardedFileStore struct {
	BasePath          string // Relative or absolute path to store files in.
	PrefixShardLayers int    // Number of extra directory layers to prefix file paths with.
	DBConn            *db.DatabaseConnection
	log               *zerolog.Logger
}

// New creates a new file based storage backend. The directory specified will
// be used as the only storage entry. This method does not check
// whether the path exists, use os.MkdirAll to ensure.
// In addition, a locking mechanism is provided.
func New(basePath string, prefixShardLayers int, dbConnection *db.DatabaseConnection, log *zerolog.Logger) *ShardedFileStore {

	store := &ShardedFileStore{
		BasePath:          basePath,
		PrefixShardLayers: prefixShardLayers,
		DBConn:            dbConnection,
		log:               log,
	}
	store.initDB()
	return store
}

// UseIn sets this store as the core data store in the passed composer and adds
// all possible extension to it.
func (store *ShardedFileStore) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(store)
	composer.UseGetReader(store)
	composer.UseTerminater(store)
	composer.UseLocker(store)
	composer.UseConcater(store)
	composer.UseFinisher(store)
}

func (store *ShardedFileStore) NewUpload(info tusd.FileInfo) (id string, err error) {
	id = uid.Uid()
	info.ID = id

	// Create the directory stucture if needed
	err = os.MkdirAll(store.metaDir(id), defaultDirectoryPerm)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(store.incompleteBinDir(), defaultDirectoryPerm)
	if err != nil {
		return "", err
	}

	// create record in uploads table
	if info.MetaData["account"] == "" {
		err = db.UpdateRow(store.DBConn.DB, `INSERT INTO uploads(id, created_at) VALUES (?, ?)`, id, time.Now().Unix())
	} else {
		err = db.UpdateRow(store.DBConn.DB,
			`INSERT INTO uploads(id, created_at, jwt_account, jwt_issuer) VALUES (?, ?, ?, ?)`,
			id, time.Now().Unix(), info.MetaData["account"], info.MetaData["issuer"],
		)
	}
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

func (store *ShardedFileStore) WriteChunk(id string, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(store.binPath(id), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, src)
	return n, err
}

func (store *ShardedFileStore) GetInfo(id string) (tusd.FileInfo, error) {
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

func (store *ShardedFileStore) GetReader(id string) (io.Reader, error) {
	return os.Open(store.binPath(id))
}

func (store *ShardedFileStore) Terminate(id string) error {
	duplicates, err := store.getDuplicateCount(id)
	if err != nil {
		return err
	}

	binPath := store.binPath(id)

	// remove upload .info file
	if err := RemoveWithDirs(store.infoPath(id), store.BasePath); err != nil {
		return err
	}

	// delete .bin if there are no other upload records using it
	if duplicates == 0 {
		if err := RemoveWithDirs(binPath, store.BasePath); err != nil {
			return err
		}
		store.log.Info().
			Str("event", "blob_deleted").
			Str("binPath", binPath).
			Msg("Removed upload bin")
	}

	// mark upload db record as deleted
	err = db.UpdateRow(store.DBConn.DB, `
		UPDATE uploads
		SET deleted = 1
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	return nil
}

func (store *ShardedFileStore) ConcatUploads(dest string, uploads []string) (err error) {
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

func (store *ShardedFileStore) LockUpload(id string) error {
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

func (store *ShardedFileStore) UnlockUpload(id string) error {
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
func (store *ShardedFileStore) newLock(id string) (lockfile.Lockfile, error) {
	path, err := filepath.Abs(store.lockPath(id))
	if err != nil {
		return lockfile.Lockfile(""), err
	}

	// We use Lockfile directly instead of lockfile.New to bypass the unnecessary
	// check whether the provided path is absolute since we just resolved it
	// on our own.
	return lockfile.Lockfile(path), nil
}

// binPath returns the path to the .bin storing the binary data.
func (store *ShardedFileStore) binPath(id string) string {
	hashBytes, isFinal, err := store.lookupHash(id)
	if err != nil {
		store.log.Fatal().Err(err).Msg("Could not look up hash")
	}

	if !isFinal {
		return store.incompleteBinPath(id)
	}

	return store.completeBinPath(hashBytes)
}

// infoPath returns the path to the .info file storing the upload's metadata.
func (store *ShardedFileStore) infoPath(id string) string {
	// <base-path>/meta/<id-shards>/<id>.info
	return filepath.Join(store.metaDir(id), id+".info")
}

// writeInfo updates the entire information. Everything will be overwritten.
func (store *ShardedFileStore) writeInfo(id string, info tusd.FileInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(store.infoPath(id), data, defaultFilePerm)
}

//
// ADDED FUNCTIONS
//

// Close frees the database connection pool held within ShardedFileStore
func (store *ShardedFileStore) Close() error {
	return store.DBConn.DB.Close()
}

// FinishUpload deduplicates the upload by its cryptographic hash
func (store *ShardedFileStore) FinishUpload(id string) error {
	store.log.Debug().
		Str("event", "upload_finished").
		Str("id", id).Msg("Finishing upload")

	// calculate hash
	hash, err := store.hashFile(id)
	if err != nil {
		return err
	}

	// update hash in uploads table
	err = db.UpdateRow(store.DBConn.DB, `
		UPDATE uploads
		SET sha256sum = ?
		WHERE id = ?
	`, hash, id)
	if err != nil {
		return err
	}

	// relocate file
	newPath := store.completeBinPath(hash)
	os.MkdirAll(filepath.Dir(newPath), defaultDirectoryPerm)
	oldPath := store.incompleteBinPath(id)
	err = os.Rename(oldPath, newPath)
	if err != nil {
		store.log.Error().
			Err(err).
			Str("oldPath", oldPath).
			Str("newPath", newPath).
			Msg("Failed to rename")
	}
	return err
}

func (store *ShardedFileStore) hashFile(id string) ([]byte, error) {
	f, err := os.Open(store.binPath(id))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func (store *ShardedFileStore) getDuplicateCount(id string) (duplicates int, err error) {
	// fetch hash
	hash, _, err := store.lookupHash(id)
	if err != nil {
		return
	}

	// check if there are any other uploads pointing to this file
	err = store.DBConn.DB.QueryRow(`
		SELECT count(id)
		FROM uploads
		WHERE
			sha256sum = ? AND
			id != ? AND
			deleted = 0
	`, hash, id).Scan(&duplicates)

	return
}

// RemoveWithDirs deletes the given path and its empty parent directories
// up to the given basePath
func RemoveWithDirs(path string, basePath string) (err error) {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	if !strings.HasPrefix(absPath, absBase) {
		return fmt.Errorf("Path %#v is not prefixed by basepath %#v", path, basePath)
	}

	if _, err := os.Stat(path); err == nil {
		err = os.Remove(path)
	} else if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	parent := path
	for {
		parent = filepath.Dir(parent)
		parentAbs, err := filepath.Abs(parent)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(parentAbs, absBase) || parentAbs == absBase {
			return err
		}

		empty, err := isDirEmpty(parent)
		if empty {
			err = os.Remove(parent)
		}
		if err != nil {
			return err
		}
	}
}

// lookupHash translates a randomly generated upload id into its cryptographic
// hash by querying the upload database.
func (store *ShardedFileStore) lookupHash(id string) (hash []byte, isFinal bool, err error) {
	row := store.DBConn.DB.QueryRow(`SELECT sha256sum FROM uploads WHERE id = ?`, id)
	err = row.Scan(&hash)

	// no finalized upload exists
	if err == sql.ErrNoRows {
		isFinal = false
		err = nil
		return
	}

	// something went wrong!
	if err != nil {
		return
	}

	isFinal = hash != nil
	return
}

// metaDir returns the directory that the info and lock files reside in for a given id
func (store *ShardedFileStore) metaDir(id string) string {
	// <base-path>/meta/<id-shards>
	shards := store.shards(id)
	return filepath.Join(store.BasePath, "meta", shards)
}

// lockPath returns the path to the .lock file for an upload id
func (store *ShardedFileStore) lockPath(id string) string {
	// <base-path>/meta/<id-shards>/<id>.lock
	return filepath.Join(store.metaDir(id), id+".lock")
}

// generates a directory hierarchy
func (store *ShardedFileStore) shards(id string) string {
	if len(id) < store.PrefixShardLayers {
		panic("id is too short for requested number of shard layers")
	}
	shards := make([]string, store.PrefixShardLayers)
	for n, char := range id[:store.PrefixShardLayers] {
		shards[n] = string(char)
	}
	return filepath.Join(shards...)
}

func (store *ShardedFileStore) incompleteBinDir() string {
	return filepath.Join(store.BasePath, "incomplete")
}

func (store *ShardedFileStore) incompleteBinPath(id string) string {
	// during upload: <base-path>/incomplete/<id>.bin
	return filepath.Join(store.incompleteBinDir(), id+".bin")
}

func (store ShardedFileStore) completeBinPath(hashBytes []byte) string {
	// finished: <base-path>/complete/<hash-shards>/<hash>.bin
	hash := fmt.Sprintf("%x", hashBytes)
	shards := store.shards(hash)
	return filepath.Join(store.BasePath, "complete", shards, hash+".bin")
}
