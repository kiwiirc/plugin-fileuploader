package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dyatlov/go-oembed/oembed"
	"github.com/gin-gonic/gin"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"willnorris.com/go/imageproxy"
)

type cacheItem struct {
	url     string
	html    string
	created int64
	wg      sync.WaitGroup
}

// HTML template
var template string
var templateLock sync.RWMutex

// In memory HTML cache
var cache = make(map[string]*cacheItem)
var cacheLock sync.Mutex
var cacheTicker *time.Ticker

var embed *oembed.Oembed

var imgProxy *imageproxy.Proxy

// Used to detect possible image urls after after failed oembed match
var isImage = regexp.MustCompile(`\.(jpe?g|png|gifv?)$`)

func (serv *UploadServer) registerEmbedHandlers(r *gin.Engine, cfg Config) error {
	data, err := getProviders(false)
	if err != nil {
		return err
	}
	embed = oembed.NewOembed()
	embed.ParseProviders(bytes.NewReader(*data))

	// Start the cleanup ticker
	startCleanupTicker(
		cfg.Embed.CacheCleanInterval.Duration,
		cfg.Embed.CacheMaxAge.Duration,
	)

	if err := loadTemplate(cfg.Embed.TemplatePath); err != nil {
		log.Println("Failed to load template: " + err.Error())
		return nil
	}

	rg := r.Group("/embed")
	rg.GET("", handleEmbed)

	// Attach imageproxy
	cache := diskCache(cfg.Embed.ImageCachePath, cfg.Embed.ImageCacheMaxSize)
	imgProxy = imageproxy.NewProxy(nil, cache)

	ic := r.Group("/image-cache/*id")
	ic.GET("", handleImageCache)
	return nil
}

func handleImageCache(c *gin.Context) {
	r := c.Request
	r.URL.Path = strings.Replace(r.URL.Path, "/image-cache", "", -1)
	spew.Dump(r.URL.Path)
	imgProxy.ServeHTTP(c.Writer, c.Request)
}

func diskCache(path string, maxSize uint64) *diskcache.Cache {
	d := diskv.New(diskv.Options{
		BasePath:     path,
		CacheSizeMax: maxSize,
		// For file "c0ffee", store file as "c0/ff/c0ffee"
		Transform: func(s string) []string { return []string{s[0:2], s[2:4]} },
	})
	return diskcache.NewWithDiskv(d)
}

func handleEmbed(c *gin.Context) {
	queryURL := c.Query("url")
	if !isValidURL(queryURL) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	queryCenter := c.Query("center")
	queryWidth := c.Query("width")
	queryHeight := c.Query("height")

	// Convert queryCenter to boolean
	center, err := strconv.ParseBool(queryCenter)
	if err != nil {
		center = false
	}

	width, err := strconv.Atoi(queryWidth)
	if err != nil {
		width = 1000
	}

	height, err := strconv.Atoi(queryHeight)
	if err != nil {
		height = 400
	}

	hash := getHash(queryURL)

	spew.Dump(queryURL, center, height, width)

	cacheLock.Lock()
	item, ok := cache[hash]
	if !ok {
		// Cache miss create new cache item
		item = &cacheItem{
			url:     queryURL,
			html:    "",
			created: time.Now().Unix(),
		}

		// Add to waitgroup so other clients can wait for the oEmbed result
		item.wg.Add(1)
		cache[hash] = item

		// Item added to cache, unlock so other requests can see the new item
		cacheLock.Unlock()

		// Attempt to fetch oEmbed data
		embedItem := embed.FindItem(queryURL)
		if embedItem != nil {
			options := oembed.Options{
				URL:       queryURL,
				MaxHeight: height,
				MaxWidth:  width,
			}
			info, err := embedItem.FetchOembed(options)
			if err != nil {
				// An unexpected error occurred
				fmt.Printf("An error occured: %s\n", err.Error())
			} else if info.Status >= 300 {
				// oEmbed returned an error status
				fmt.Printf("Response status code is: %d\n", info.Status)
			} else if info.HTML != "" {
				// oEmbed returned embedable html
				fmt.Printf("Oembed info:\n%s\n", info)
				item.html = info.HTML
			} else if info.Type == "photo" {
				// oEmbed returned a photo type the url should be an image
				fmt.Println("type photo " + info.URL)
				item.html = getImageHTML(c, info.URL, height)
			} else {
				spew.Dump(info)
			}
		}

		// oEmbed did not return any html, maybe the url is direct to an image
		if item.html == "" && isImage.MatchString(queryURL) {
			item.html = getImageHTML(c, queryURL, height)
		}

		// Still no html send an error to the parent
		if item.html == "" {
			item.html = "<script>window.parent.postMessage({ error: true }, '*');</script>"
		}

		// Decrease the waitgroup so other requests can complete
		item.wg.Done()
	} else {
		log.Printf("Cache HIT")
		// Cache hit unlock the cache
		cacheLock.Unlock()
	}

	// Wait until the cache item is fulfilled
	item.wg.Wait()

	// Prepare html and send it to the client
	style := "display: flex; justify-content: center;"
	if !center {
		style = ""
	}
	templateLock.RLock()
	htmlData := strings.Replace(template, "{{body.html}}", item.html, -1)
	templateLock.RUnlock()
	htmlData = strings.Replace(htmlData, "/* style.body */", style, -1)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlData))
}

func getProviders(force bool) (*[]byte, error) {
	log.Println("Getting oEmbed Providers")
	if _, err := os.Stat("providers.json"); force || os.IsNotExist(err) {
		resp, err := http.Get("https://oembed.com/providers.json")
		if err != nil {
			return nil, errors.New("Failed to fetch oEmbed providers: " + err.Error())
		}
		defer resp.Body.Close()

		log.Println("Fetched oEmbed Providers")

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New("Failed to read oEmbed providers: " + err.Error())
		}

		log.Println("Read oEmbed Providers")

		// Unmarshal to temp interface to ensure valid json
		var temp interface{}
		err = json.Unmarshal(data, &temp)
		if err != nil {
			return nil, errors.New("Failed to parse oEmbed providers: " + err.Error())
		}

		log.Println("Tested oEmbed Providers")

		// Data appears to be valid json open providers file for writing
		file, err := os.Create("providers.json")
		if err != nil {
			return nil, errors.New("Failed to create oEmbed providers: " + err.Error())
		}
		defer file.Close()

		log.Println("Created oEmbed Providers")

		// Write providers.json
		_, err = file.Write(data)
		if err != nil {
			return nil, errors.New("Failed to write oEmbed providers: " + err.Error())
		}

		log.Println("Written oEmbed Providers")

		return &data, nil
	}

	// Open existing providers.json
	file, err := os.Open("providers.json")
	if err != nil {
		return nil, errors.New("Failed to open oEmbed providers: " + err.Error())
	}
	defer file.Close()

	// Read providers.json data
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.New("Failed to read oEmbed providers: " + err.Error())
	}

	return &data, nil
}

func startCleanupTicker(cleanInterval, cacheMaxAge time.Duration) {
	cacheTicker = time.NewTicker(cleanInterval)
	go func() {
		for range cacheTicker.C {
			cleanCache(cacheMaxAge)
		}
	}()
}

func cleanCache(cacheMaxAge time.Duration) {
	createdBefore := time.Now().Unix() - int64(cacheMaxAge.Seconds())
	var expired []string
	for hash, item := range cache {
		if item.created >= createdBefore {
			continue
		}
		expired = append(expired, hash)
	}

	if len(expired) == 0 {
		// Nothing to clean
		log.Println("No cache items to clean")
		return
	}

	cacheLock.Lock()
	for _, hash := range expired {
		if hash == "" {
			break
		}
		log.Println("Deleting cache item: " + hash)
		delete(cache, hash)
	}
	cacheLock.Unlock()
}

func loadTemplate(templatePath string) error {
	html, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}
	templateLock.Lock()
	template = string(html)
	templateLock.Unlock()
	return nil
}

func getImageHTML(c *gin.Context, url string, height int) string {
	newURL := "http://"
	if c.Request.TLS != nil {
		newURL = "https://"
	}
	newURL += c.Request.Host
	// fixed height proportional width
	newURL += "/image-cache/x" + strconv.Itoa(height) + "/" + url
	return "<img src=\"" + newURL + "\" onerror=\"imgError()\" class=\"kiwi-embed-image\" />"
}

func getHash(url string) string {
	hasher := sha256.New()
	hasher.Write([]byte(url))
	return hex.EncodeToString(hasher.Sum(nil))
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Host != ""
}
