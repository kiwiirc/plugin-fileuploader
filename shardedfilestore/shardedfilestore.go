// Package shardedfilestore is a modified version of the tusd/filestore implementation.
// Splits file storage into subdirectories based on the hash prefix.
// based on https://github.com/tus/tusd/blob/e138fc3e9e01ab8294a393ec0407eff38a708ddb/pkg/filestore/filestore.go
package shardedfilestore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	_ "github.com/mattn/go-sqlite3"    // register SQL driver

	"github.com/kiwiirc/plugin-fileuploader/db"
	"github.com/rs/zerolog"
	"github.com/tus/tusd/pkg/handler"
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
func (store ShardedFileStore) UseIn(composer *handler.StoreComposer) {
	composer.UseCore(store)
	composer.UseTerminater(store)
	composer.UseConcater(store)
	composer.UseLengthDeferrer(store)
}

func (store ShardedFileStore) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	var err error

	id := Uid()
	binPath := store.binPath(id)
	info.ID = id
	info.Storage = map[string]string{
		"Type": "filestore",
		"Path": binPath,
	}

	// Create the directory stucture if needed
	err = os.MkdirAll(store.metaDir(id), defaultDirectoryPerm)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(store.incompleteBinDir(), defaultDirectoryPerm)
	if err != nil {
		return nil, err
	}

	// Metadata is exposed to users via headers, so remove RemoteIP
	remoteIP := info.MetaData["RemoteIP"]
	delete(info.MetaData, "RemoteIP")

	// create record in uploads table
	if info.MetaData["account"] == "" {
		err = db.UpdateRow(store.DBConn.DB, `INSERT INTO uploads(id, created_at, uploader_ip) VALUES (?, ?, ?)`, id, time.Now().Unix(), remoteIP)
	} else {
		err = db.UpdateRow(store.DBConn.DB,
			`INSERT INTO uploads(id, created_at, uploader_ip, jwt_account, jwt_issuer) VALUES (?, ?, ?, ?, ?)`,
			id, time.Now().Unix(), remoteIP, info.MetaData["account"], info.MetaData["issuer"],
		)
	}
	if err != nil {
		return nil, err
	}

	// Create .bin file with no content
	file, err := os.OpenFile(store.binPath(id), os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return nil, err
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}

	upload := &fileUpload{
		info:     info,
		store:    store,
		infoPath: store.infoPath(id),
		binPath:  store.binPath(id),
	}

	// writeInfo creates the file by itself if necessary
	err = upload.writeInfo()
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (store ShardedFileStore) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	info := handler.FileInfo{}
	data, err := ioutil.ReadFile(store.infoPath(id))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	binPath := store.binPath(id)
	infoPath := store.infoPath(id)
	stat, err := os.Stat(binPath)
	if err != nil {
		return nil, err
	}

	info.Offset = stat.Size()

	return &fileUpload{
		info:     info,
		binPath:  binPath,
		store:    store,
		infoPath: infoPath,
	}, nil
}

func (store ShardedFileStore) AsTerminatableUpload(upload handler.Upload) handler.TerminatableUpload {
	return upload.(*fileUpload)
}

func (store ShardedFileStore) AsLengthDeclarableUpload(upload handler.Upload) handler.LengthDeclarableUpload {
	return upload.(*fileUpload)
}

func (store ShardedFileStore) AsConcatableUpload(upload handler.Upload) handler.ConcatableUpload {
	return upload.(*fileUpload)
}

// binPath returns the path to the file storing the binary data.
func (store ShardedFileStore) binPath(id string) string {
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

type fileUpload struct {
	// info stores the current information about the upload
	info handler.FileInfo
	// store the fileupload's store
	store ShardedFileStore
	// infoPath is the path to the .info file
	infoPath string
	// binPath is the path to the binary file (which has no extension)
	binPath string
}

func (upload *fileUpload) GetInfo(ctx context.Context) (handler.FileInfo, error) {
	return upload.info, nil
}

func (upload *fileUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, src)

	// If the HTTP PATCH request gets interrupted in the middle (e.g. because
	// the user wants to pause the upload), Go's net/http returns an io.ErrUnexpectedEOF.
	// However, for ShardedFileStore it's not important whether the stream has ended
	// on purpose or accidentally.
	if err == io.ErrUnexpectedEOF {
		err = nil
	}

	upload.info.Offset += n

	return n, err
}

func (upload *fileUpload) GetReader(ctx context.Context) (io.Reader, error) {
	return os.Open(upload.binPath)
}

func (upload *fileUpload) Terminate(ctx context.Context) error {
	return upload.store.Terminate(upload.info.ID)
}

func (upload *fileUpload) ConcatUploads(ctx context.Context, uploads []handler.Upload) (err error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, partialUpload := range uploads {
		fileUpload := partialUpload.(*fileUpload)

		src, err := os.Open(fileUpload.binPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, src); err != nil {
			return err
		}
	}

	return
}

func (upload *fileUpload) DeclareLength(ctx context.Context, length int64) error {
	upload.info.Size = length
	upload.info.SizeIsDeferred = false
	return upload.writeInfo()
}

// writeInfo updates the entire information. Everything will be overwritten.
func (upload *fileUpload) writeInfo() error {
	data, err := json.Marshal(upload.info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(upload.infoPath, data, defaultFilePerm)
}

func (upload *fileUpload) FinishUpload(ctx context.Context) error {
	upload.store.log.Debug().
		Str("event", "upload_finished").
		Str("id", upload.info.ID).Msg("Finishing upload")

	// calculate hash
	hash, err := upload.store.hashFile(upload.info.ID)
	if err != nil {
		return err
	}

	// update hash in uploads table
	err = db.UpdateRow(upload.store.DBConn.DB, `
		UPDATE uploads
		SET sha256sum = ?
		WHERE id = ?
	`, hash, upload.info.ID)
	if err != nil {
		return err
	}

	// relocate file
	newPath := upload.store.completeBinPath(hash)
	os.MkdirAll(filepath.Dir(newPath), defaultDirectoryPerm)
	oldPath := upload.store.incompleteBinPath(upload.info.ID)

	err = os.Rename(oldPath, newPath)
	if err != nil {
		upload.store.log.Error().
			Err(err).
			Str("oldPath", oldPath).
			Str("newPath", newPath).
			Msg("Failed to rename")
	}

	if err == nil {
		upload.info.Storage["Path"] = newPath
		err = upload.writeInfo()
	}

	return err
}

// ADDED FUNCTIONS

// taken from https://github.com/tus/tusd/blob/42bfe35457f8bfc79a0af40a9f51c8112903737e/internal/uid/uid.go
func Uid() string {
	id := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		// This is probably an appropriate way to handle errors from our source
		// for random bits.
		panic(err)
	}
	return hex.EncodeToString(id)
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
