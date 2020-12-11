package noembed

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var noembedURL = "https://noembed.com/embed?url={url}"

// NoEmbed represents this package
type NoEmbed struct {
	data       *Data
	httpClient *http.Client
}

// Data represents the data for noembed providers
type Data []struct {
	Name     string  `json:"name"`
	Patterns []Regex `json:"patterns"`
}

// Response represents the data returned by noembed server
type Response struct {
	AuthorName      string `json:"author_name"`
	AuthorURL       string `json:"author_url"`
	ProviderName    string `json:"provider_name"`
	ProviderURL     string `json:"provider_url"`
	Title           string `json:"title"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	HTML            string `json:"html"`
	Version         string `json:"version"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_width,string"`
	ThumbnailHeight int    `json:"thumbnail_height,string"`
	Width           int    `json:"width,string"`
	Height          int    `json:"height,string"`
}

// New returns a Noembed object
func New() *NoEmbed {
	return &NoEmbed{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// ParseProviders parses the raw json obtained from noembed.com
func (n *NoEmbed) ParseProviders(buf io.Reader) error {
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return err
	}

	var noembedData Data
	err = json.Unmarshal(data, &noembedData)
	if err != nil {
		return err
	}

	n.data = &noembedData
	return nil
}

// Get returns a noembed response object
func (n *NoEmbed) Get(url string) (resp *Response, err error) {
	if !n.ValidURL(url) {
		err = errors.New("Unsupported URL")
		return
	}

	reqURL := strings.Replace(noembedURL, "{url}", url, 1)

	var httpResp *http.Response
	httpResp, err = n.httpClient.Get(reqURL)
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	var body []byte
	body, err = ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}

	return
}

// ValidURL is used to test if a url is supported by noembed
func (n *NoEmbed) ValidURL(url string) bool {
	for _, entry := range *n.data {
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
