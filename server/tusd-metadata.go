/*
Copyright (c) 2013-2017 Transloadit Ltd and Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// parseMeta and serializeMeta unexported functions extracted from
// from https://github.com/tus/tusd/blob/0.10.0/unrouted_handler.go#L835-L879

package server

import (
	"encoding/base64"
	"strings"
)

// Parse the Upload-Metadata header as defined in the File Creation extension.
// e.g. Upload-Metadata: name bHVucmpzLnBuZw==,type aW1hZ2UvcG5n
func parseMeta(header string) map[string]string {
	meta := make(map[string]string)

	for _, element := range strings.Split(header, ",") {
		element := strings.TrimSpace(element)

		parts := strings.Split(element, " ")

		// Do not continue with this element if no key and value or presented
		if len(parts) != 2 {
			continue
		}

		// Ignore corrent element if the value is no valid base64
		key := parts[0]
		value, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}

		meta[key] = string(value)
	}

	return meta
}

// Serialize a map of strings into the Upload-Metadata header format used in the
// response for HEAD requests.
// e.g. Upload-Metadata: name bHVucmpzLnBuZw==,type aW1hZ2UvcG5n
func serializeMeta(meta map[string]string) string {
	header := ""
	for key, value := range meta {
		valueBase64 := base64.StdEncoding.EncodeToString([]byte(value))
		header += key + " " + valueBase64 + ","
	}

	// Remove trailing comma
	if len(header) > 0 {
		header = header[:len(header)-1]
	}

	return header
}
