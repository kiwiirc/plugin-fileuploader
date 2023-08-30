// Package shardedfilestore is a modified version of the tusd/filestore implementation.
// Splits file storage into subdirectories based on the hash prefix.
// based on https://github.com/tus/tusd/blob/6d987aa226e2e6cefca1df012da5599a57622b17/pkg/filestore/filestore.go
package shardedfilestore

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IGLOU-EU/go-wildcard"
	"github.com/rs/zerolog"
	"github.com/tus/tusd/pkg/handler"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	_ "github.com/mattn/go-sqlite3"    // register SQL driver

	"github.com/kiwiirc/plugin-fileuploader/config"
	"github.com/kiwiirc/plugin-fileuploader/db"
)

var defaultFilePerm = os.FileMode(0664)
var defaultDirectoryPerm = os.FileMode(0775)

// ShardedFileStore implements various tusd.DataStore-related interfaces.
// See the interfaces for more documentation about the different methods.
type ShardedFileStore struct {
	BasePath             string        // Relative or absolute path to store files in.
	PrefixShardLayers    int           // Number of extra directory layers to prefix file paths with.
	ExpireTime           time.Duration // How long before an upload expires (seconds)
	ExpireIdentifiedTime time.Duration // How long before an upload expires with valid account (seconds)
	PreFinishCommands    []config.PreFinishCommand
	DBConn               *db.DatabaseConnection
	log                  *zerolog.Logger
}

// New creates a new file based storage backend. The directory specified will
// be used as the only storage entry. This method does not check
// whether the path exists, use os.MkdirAll to ensure.
// In addition, a locking mechanism is provided.
func New(basePath string, prefixShardLayers int, expireTime, expireIdentifiedTime time.Duration, PreFinishCommands []config.PreFinishCommand, dbConnection *db.DatabaseConnection, log *zerolog.Logger) *ShardedFileStore {
	store := &ShardedFileStore{
		BasePath:             basePath,
		PrefixShardLayers:    prefixShardLayers,
		ExpireTime:           expireTime,
		ExpireIdentifiedTime: expireIdentifiedTime,
		PreFinishCommands:    PreFinishCommands,
		DBConn:               dbConnection,
		log:                  log,
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

	if info.ID == "" {
		info.ID = Uid()
	}
	binPath := store.binPath(info.ID)
	info.Storage = map[string]string{
		"Type": "filestore",
		"Path": binPath,
	}

	// Create the directory stucture if needed
	err = os.MkdirAll(store.metaDir(info.ID), defaultDirectoryPerm)
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
	err = db.UpdateRow(store.DBConn.DB,
		`INSERT INTO uploads(id, created_at, uploader_ip, jwt_account, jwt_issuer) VALUES (?, ?, ?, ?, ?)`,
		info.ID, time.Now().Unix(), remoteIP, info.MetaData["account"], info.MetaData["issuer"],
	)
	if err != nil {
		return nil, err
	}

	// Create .bin file with no content
	file, err := os.OpenFile(binPath, os.O_CREATE|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return nil, err
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}

	upload := &FileUpload{
		info:     info,
		store:    store,
		infoPath: store.infoPath(info.ID),
		binPath:  binPath,
	}

	// writeInfo creates the file by itself if necessary
	err = upload.writeInfo()
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (store ShardedFileStore) GetFileUpload(ctx context.Context, id string) (*FileUpload, error) {
	info := handler.FileInfo{}
	data, err := ioutil.ReadFile(store.infoPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			// Interpret os.ErrNotExist as 404 Not Found
			err = handler.ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	binPath := store.binPath(id)
	infoPath := store.infoPath(id)
	stat, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Interpret os.ErrNotExist as 404 Not Found
			err = handler.ErrNotFound
		}
		return nil, err
	}

	info.Offset = stat.Size()

	return &FileUpload{
		info:     info,
		binPath:  binPath,
		store:    store,
		infoPath: infoPath,
	}, nil
}

func (store ShardedFileStore) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	return store.GetFileUpload(ctx, id)
}

func (store ShardedFileStore) AsTerminatableUpload(upload handler.Upload) handler.TerminatableUpload {
	return upload.(*FileUpload)
}

func (store ShardedFileStore) AsLengthDeclarableUpload(upload handler.Upload) handler.LengthDeclarableUpload {
	return upload.(*FileUpload)
}

func (store ShardedFileStore) AsConcatableUpload(upload handler.Upload) handler.ConcatableUpload {
	return upload.(*FileUpload)
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

type FileUpload struct {
	// info stores the current information about the upload
	info handler.FileInfo
	// store the fileupload's store
	store ShardedFileStore
	// infoPath is the path to the .info file
	infoPath string
	// binPath is the path to the binary file (which has no extension)
	binPath string
}

func (upload *FileUpload) GetInfo(ctx context.Context) (handler.FileInfo, error) {
	return upload.info, nil
}

func (upload *FileUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, src)

	upload.info.Offset += n
	return n, err
}

func (upload *FileUpload) GetReader(ctx context.Context) (io.Reader, error) {
	return os.Open(upload.binPath)
}

func (upload *FileUpload) GetFile(ctx context.Context) (*os.File, error) {
	return os.Open(upload.binPath)
}

func (upload *FileUpload) Terminate(ctx context.Context) error {
	return upload.store.Terminate(upload.info.ID)
}

func (upload *FileUpload) ConcatUploads(ctx context.Context, uploads []handler.Upload) (err error) {
	file, err := os.OpenFile(upload.binPath, os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, partialUpload := range uploads {
		fileUpload := partialUpload.(*FileUpload)

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

func (upload *FileUpload) DeclareLength(ctx context.Context, length int64) error {
	upload.info.Size = length
	upload.info.SizeIsDeferred = false
	return upload.writeInfo()
}

// writeInfo updates the entire information. Everything will be overwritten.
func (upload *FileUpload) writeInfo() error {
	data, err := json.Marshal(upload.info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(upload.infoPath, data, defaultFilePerm)
}

func (upload *FileUpload) FinishUpload(ctx context.Context) error {
	upload.store.log.Debug().
		Str("event", "upload_finished").
		Str("id", upload.info.ID).Msg("Finishing upload")

	oldPath := upload.store.incompleteBinPath(upload.info.ID)

	// execute completion commands
	fileType, ok := upload.info.MetaData["filetype"]
	if !ok {
		upload.store.log.Warn().
			Msg("Failed to get filetype metadata in ExecuteCommands")
	}

	absPath, err := filepath.Abs(oldPath)
	if err != nil {
		upload.store.log.Error().
			Err(err).
			Msg("Failed resolve path in ExecuteCommands")
	}

	for _, preFinish := range upload.store.PreFinishCommands {
		if !wildcard.Match(preFinish.Pattern, fileType) {
			continue
		}

		args := make([]string, 0)
		for _, arg := range preFinish.Args {
			arg = strings.ReplaceAll(arg, "%FILE%", absPath)
			args = append(args, arg)
		}

		cmd := exec.Command(preFinish.Command, args...)
		var stdOut, stdErr bytes.Buffer
		cmd.Stdout = &stdOut
		cmd.Stderr = &stdErr
		if err := cmd.Run(); err != nil && preFinish.RejectOnNoneZeroExit {
			upload.store.log.Warn().
				Err(err).
				Strs("args", args).
				Str("command", preFinish.Command).
				Msg("Error with pre-finish command")

			upload.store.log.Debug().
				Str("stdout", stdOut.String()).
				Str("stderr", stdErr.String()).
				Msg("Error with pre-finish command")

			upload.store.Terminate(upload.info.ID)
			return handler.NewHTTPError(errors.New("Upload has been reject by server"), 406)
		}
	}

	// calculate hash
	hash, err := upload.store.hashFile(upload.info.ID)
	if err != nil {
		upload.store.log.Error().
			Err(err).
			Msg("Failed to hash completed upload")
		return err
	}

	expires := durationToExpire(upload.store.ExpireTime)
	if upload.info.MetaData["account"] != "" {
		expires = durationToExpire(upload.store.ExpireIdentifiedTime)
	}
	upload.info.MetaData["expires"] = strconv.FormatInt(expires, 10)

	// update hash in uploads table
	err = db.UpdateRow(upload.store.DBConn.DB, `
		UPDATE uploads
		SET sha256sum = ?,
		expires_at = ?
		WHERE id = ?
	`, hash, expires, upload.info.ID)
	if err != nil {
		upload.store.log.Error().
			Err(err).
			Msg("Failed to update db")
		return err
	}

	// relocate file
	newPath := upload.store.completeBinPath(hash)
	os.MkdirAll(filepath.Dir(newPath), defaultDirectoryPerm)

	if _, err := os.Stat(newPath); err != nil {
		// file needs moving to the sharded filestore
		err = os.Rename(oldPath, newPath)
		if err != nil {
			upload.store.log.Error().
				Err(err).
				Str("oldPath", oldPath).
				Str("newPath", newPath).
				Msg("Failed to rename")
		}
	} else {
		// file already exists just remove the tempoary upload
		err = os.Remove(oldPath)
		if err != nil {
			upload.store.log.Error().
				Err(err).
				Str("oldPath", oldPath).
				Msg("Failed to remove")
		}
	}

	upload.info.Storage["Path"] = newPath
	err = upload.writeInfo()

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

	// remove upload .info file
	if err := RemoveWithDirs(store.infoPath(id), store.BasePath); err != nil {
		return err
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
			break
		}

		empty, err := isDirEmpty(parent)
		if empty {
			err = os.Remove(parent)
		}
		if err != nil {
			return err
		}
	}

	return nil
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

func durationToExpire(d time.Duration) int64 {
	timeStr := fmt.Sprintf("%.0f", d.Seconds())
	timeInt, _ := strconv.Atoi(timeStr)
	return time.Now().Unix() + int64(timeInt)
}
