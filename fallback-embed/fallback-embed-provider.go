package fallbackembed

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FallbackEmbed represents this package
type FallbackEmbed struct {
	data        *Data
	httpClient  *http.Client
	targetKey   string
	providerURL string
}

// Data represents the data for FallbackEmbed providers
type Data []struct {
	Name     string  `json:"name"`
	Patterns []Regex `json:"patterns"`
}

// New returns a FallbackEmbed object
func New(providerURL, targetKey string) *FallbackEmbed {
	obj := &FallbackEmbed{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		providerURL: providerURL,
		targetKey:   targetKey,
	}

	return obj
}

// ParseProviders parses the raw json obtained from noembed.com
func (f *FallbackEmbed) ParseProviders(buf io.Reader) error {
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return err
	}

	var providerData Data
	err = json.Unmarshal(data, &providerData)
	if err != nil {
		return err
	}

	f.data = &providerData
	return nil
}

// Get returns html string
func (f *FallbackEmbed) Get(url string, width int, height int) (html string, err error) {
	if !f.ValidURL(url) {
		return
	}

	// Do replacements
	reqURL := strings.Replace(f.providerURL, "{url}", url, 1)
	reqURL = strings.Replace(reqURL, "{width}", strconv.Itoa(width), 1)
	reqURL = strings.Replace(reqURL, "{height}", strconv.Itoa(height), 1)

	var httpResp *http.Response
	httpResp, err = f.httpClient.Get(reqURL)
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	var body []byte
	body, err = ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return
	}

	// Try to parse json response
	resp := make(map[string]interface{})
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}

	// Check targetKey exists
	if jsonVal, ok := resp[f.targetKey]; ok {
		// Check targetVal is string
		if htmlString, ok := jsonVal.(string); ok {
			html = htmlString
			return
		}
	}

	err = errors.New("Failed to get target json key")
	return
}

// ValidURL is used to test if a url is supported by noembed
func (f *FallbackEmbed) ValidURL(url string) bool {
	for _, entry := range *f.data {
		for _, pattern := range entry.Patterns {
			if pattern.Regexp.MatchString(url) {
				return true
			}
		}
	}
	return false
}

// Regex Unmarshaler
type Regex struct {
	regexp.Regexp
}

// UnmarshalText used to unmarshal regexp's from text
func (r *Regex) UnmarshalText(text []byte) error {
	reg, err := regexp.Compile(string(text))
	r.Regexp = *reg
	return err
}
