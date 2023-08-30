// Copyright (c) 2013-2017 Transloadit Ltd and Contributors

// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// these functions where extracted from: https://github.com/tus/tusd/blob/82bbff655b28648e6d8652a987932bc58f93c2cf/pkg/handler/unrouted_handler.go

package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/tus/tusd/pkg/handler"
	tusd "github.com/tus/tusd/pkg/handler"
)

var (
	reExtractFileID = regexp.MustCompile(`([^/]+)\/?$`)
	reMimeType      = regexp.MustCompile(`^[a-z]+\/[a-z0-9\-\+\.]+$`)

	errReadTimeout     = errors.New("read tcp: i/o timeout")
	errConnectionReset = errors.New("read tcp: connection reset by peer")
)

// GetFile handles requests to download a file using a GET request. This is not
// part of the specification.
// func (handler *UnroutedHandler) GetFile(w http.ResponseWriter, r *http.Request) {
func (serv *UploadServer) getFile(handler *tusd.UnroutedHandler, store *shardedfilestore.ShardedFileStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		r := c.Request
		w := c.Writer

		id, err := extractIDFromPath(r.URL.Path)
		if err != nil {
			serv.sendError(w, r, err)
			return
		}

		if serv.composer.UsesLocker {
			lock, err := serv.lockUpload(id)
			if err != nil {
				serv.sendError(w, r, err)
				return
			}

			defer lock.Unlock()
		}
		upload, err := store.GetFileUpload(ctx, id)
		// upload, err := config.StoreComposer.Core.GetUpload(ctx, id)
		if err != nil {
			serv.sendError(w, r, err)
			return
		}

		info, err := upload.GetInfo(ctx)
		if err != nil {
			serv.sendError(w, r, err)
			return
		}

		// Set headers before sending responses
		w.Header().Set("Accept-Ranges", "bytes")

		contentType, contentDisposition := filterContentType(info)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", contentDisposition)

		// If no data has been uploaded yet, respond with an empty "204 No Content" status.
		if info.Offset == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		file, err := upload.GetFile(ctx)
		if err != nil {
			serv.sendError(w, r, err)
			return
		}

		var fileReader io.Reader

		// Check if the request contains a range header
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			ranges, err := parseRange(rangeHeader, info.Offset)
			if err != nil {
				serv.sendError(w, r, err)
				return
			}

			// Check if there is more than one range specified
			if len(ranges) > 1 {
				// Multiple ranges are not supported, respond with "416 Range Not Satisfiable" status
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.Offset))
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Process the single range
			rangeStart := ranges[0].start
			rangeEnd := rangeStart + ranges[0].length - 1
			rangeSize := ranges[0].length

			if rangeStart < 0 || rangeEnd >= info.Offset {
				// Invalid range, respond with "416 Range Not Satisfiable" status
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.Offset))
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Set the appropriate headers for the range response
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, info.Offset))
			w.Header().Set("Content-Length", strconv.FormatInt(rangeSize, 10))

			// Send "206 Partial Content" status indicating a successful range request
			w.WriteHeader(http.StatusPartialContent)

			if rangeStart > 0 {
				_, err = file.Seek(rangeStart-1, io.SeekStart)
				if err != nil {
					serv.sendError(w, r, err)
					return
				}
			}

			fileReader = io.Reader(file)

			// Copy the range to the response writer
			io.CopyN(w, fileReader, rangeSize)
		} else {
			// No range requested, send the entire file

			// Set headers before sending responses
			w.Header().Set("Content-Length", strconv.FormatInt(info.Offset, 10))

			// Send "200 OK" status indicating a successful full file response
			w.WriteHeader(http.StatusOK)

			fileReader = io.Reader(file)

			// Copy the entire file to the response writer
			io.Copy(w, fileReader)

		}

		// Try to close the reader if the io.Closer interface is implemented
		if closer, ok := fileReader.(io.Closer); ok {
			closer.Close()
		}
	}
}

// Send the error in the response body. The status code will be looked up in
// ErrStatusCodes. If none is found 500 Internal Error will be used.
func (serv *UploadServer) sendError(w http.ResponseWriter, r *http.Request, err error) {
	// Errors for read timeouts contain too much information which is not
	// necessary for us and makes grouping for the metrics harder. The error
	// message looks like: read tcp 127.0.0.1:1080->127.0.0.1:53673: i/o timeout
	// Therefore, we use a common error message for all of them.
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		err = errReadTimeout
	}

	// Errors for connnection resets also contain TCP details, we don't need, e.g:
	// read tcp 127.0.0.1:1080->127.0.0.1:10023: read: connection reset by peer
	// Therefore, we also trim those down.
	if strings.HasSuffix(err.Error(), "read: connection reset by peer") {
		err = errConnectionReset
	}

	// TODO: Decide if we should handle this in here, in body_reader or not at all.
	// If the HTTP PATCH request gets interrupted in the middle (e.g. because
	// the user wants to pause the upload), Go's net/http returns an io.ErrUnexpectedEOF.
	// However, for the handler it's not important whether the stream has ended
	// on purpose or accidentally.
	//if err == io.ErrUnexpectedEOF {
	//	err = nil
	//}

	// TODO: Decide if we want to ignore connection reset errors all together.
	// In some cases, the HTTP connection gets reset by the other peer. This is not
	// necessarily the tus client but can also be a proxy in front of tusd, e.g. HAProxy 2
	// is known to reset the connection to tusd, when the tus client closes the connection.
	// To avoid erroring out in this case and loosing the uploaded data, we can ignore
	// the error here without causing harm.
	//if strings.Contains(err.Error(), "read: connection reset by peer") {
	//	err = nil
	//}

	statusErr, ok := err.(handler.HTTPError)
	if !ok {
		statusErr = handler.NewHTTPError(err, http.StatusInternalServerError)
	}

	reason := append(statusErr.Body(), '\n')
	if r.Method == "HEAD" {
		reason = nil
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(reason)))
	w.WriteHeader(statusErr.StatusCode())
	w.Write(reason)

	// handler.log("ResponseOutgoing", "status", strconv.Itoa(statusErr.StatusCode()), "method", r.Method, "path", r.URL.Path, "error", err.Error(), "requestId", getRequestId(r))
}

// lockUpload creates a new lock for the given upload ID and attempts to lock it.
// The created lock is returned if it was aquired successfully.
func (serv *UploadServer) lockUpload(id string) (handler.Lock, error) {
	lock, err := serv.composer.Locker.NewLock(id)
	if err != nil {
		return nil, err
	}

	if err := lock.Lock(); err != nil {
		return nil, err
	}

	return lock, nil
}

// extractIDFromPath pulls the last segment from the url provided
func extractIDFromPath(url string) (string, error) {
	result := reExtractFileID.FindStringSubmatch(url)
	if len(result) != 2 {
		return "", handler.ErrNotFound
	}
	return result[1], nil
}

// mimeInlineBrowserWhitelist is a map containing MIME types which should be
// allowed to be rendered by browser inline, instead of being forced to be
// downloaded. For example, HTML or SVG files are not allowed, since they may
// contain malicious JavaScript. In a similiar fashion PDF is not on this list
// as their parsers commonly contain vulnerabilities which can be exploited.
// The values of this map does not convey any meaning and are therefore just
// empty structs.
var mimeInlineBrowserWhitelist = map[string]struct{}{
	"text/plain": struct{}{},

	"image/png":  struct{}{},
	"image/jpeg": struct{}{},
	"image/gif":  struct{}{},
	"image/bmp":  struct{}{},
	"image/webp": struct{}{},

	"audio/wave":      struct{}{},
	"audio/wav":       struct{}{},
	"audio/x-wav":     struct{}{},
	"audio/x-pn-wav":  struct{}{},
	"audio/webm":      struct{}{},
	"video/webm":      struct{}{},
	"audio/ogg":       struct{}{},
	"video/ogg":       struct{}{},
	"application/ogg": struct{}{},
}

// filterContentType returns the values for the Content-Type and
// Content-Disposition headers for a given upload. These values should be used
// in responses for GET requests to ensure that only non-malicious file types
// are shown directly in the browser. It will extract the file name and type
// from the "fileame" and "filetype".
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Disposition
func filterContentType(info handler.FileInfo) (contentType string, contentDisposition string) {
	filetype := info.MetaData["filetype"]

	if reMimeType.MatchString(filetype) {
		// If the filetype from metadata is well formed, we forward use this
		// for the Content-Type header. However, only whitelisted mime types
		// will be allowed to be shown inline in the browser
		contentType = filetype
		if _, isWhitelisted := mimeInlineBrowserWhitelist[filetype]; isWhitelisted {
			contentDisposition = "inline"
		} else {
			contentDisposition = "attachment"
		}
	} else {
		// If the filetype from the metadata is not well formed, we use a
		// default type and force the browser to download the content.
		contentType = "application/octet-stream"
		contentDisposition = "attachment"
	}

	// Add a filename to Content-Disposition if one is available in the metadata
	if filename, ok := info.MetaData["filename"]; ok {
		contentDisposition += ";filename=" + strconv.Quote(filename)
	}

	return contentType, contentDisposition
}
