package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	"github.com/dyatlov/go-oembed/oembed"
	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/plugin-fileuploader/noembed"
	"willnorris.com/go/imageproxy"
)

type cacheItem struct {
	url     string
	html    string
	created int64
	wg      sync.WaitGroup
}

type imgWaiterItem struct {
	url     string
	status  int
	created int64
	wg      sync.WaitGroup
}

// HTML template
var template string
var templateLock sync.RWMutex

// In memory HTML cache
var cache = make(map[string]*cacheItem)
var cacheMutex sync.Mutex
var cacheTicker *time.Ticker

// Image waiter
var imgWaiter = make(map[string]*imgWaiterItem)
var imgWaiterMutex sync.Mutex

var oEmbed *oembed.Oembed
var noEmbed *noembed.NoEmbed
var imgProxy *imageproxy.Proxy

// Used to detect possible image urls
var isImage = regexp.MustCompile(`\.(jpe?g|png|gifv?)$`)

func (serv *UploadServer) registerEmbedHandlers(r *gin.Engine, cfg Config) error {
	serv.log.Info().
		Msg("Starting embed handlers")

	// Prepare oEmbed provider
	oembedJSON, err := getProvidersCached("https://oembed.com/providers.json", "oembed-providers.json", false)
	if err != nil {
		serv.log.Error().
			Err(err).
			Msg("Failed to get oembed providers json")
		return err
	}
	oEmbed = oembed.NewOembed()
	err = oEmbed.ParseProviders(bytes.NewReader(*oembedJSON))
	if err != nil {
		serv.log.Error().
			Err(err).
			Msg("Failed to parse oembed providers json")
		return err
	}

	// Prepare noEmbed provider
	noembedJSON, err := getProvidersCached("https://noembed.com/providers", "noembed-providers.json", false)
	if err != nil {
		serv.log.Error().
			Err(err).
			Msg("Failed to get noembed providers json")
		return err
	}
	noEmbed = noembed.New()
	err = noEmbed.ParseProviders(bytes.NewReader(*noembedJSON))
	if err != nil {
		serv.log.Error().
			Err(err).
			Msg("Failed to parse noembed providers json")
		return err
	}

	// Check config defaults
	cacheCleanInterval := cfg.Embed.CacheCleanInterval.Duration
	if cacheCleanInterval == time.Duration(0) {
		cacheCleanInterval, _ = time.ParseDuration("15m")
	}

	cacheMaxAge := cfg.Embed.CacheMaxAge.Duration
	if cacheMaxAge == time.Duration(0) {
		cacheMaxAge, _ = time.ParseDuration("1h")
	}

	templatePath := cfg.Embed.TemplatePath
	if templatePath == "" {
		templatePath = "templates/embed.html"
	}

	// Start the cleanup ticker
	serv.startCleanupTicker(
		cacheCleanInterval,
		cacheMaxAge,
	)

	// Load embed html template
	if err := loadTemplate(templatePath); err != nil {
		serv.log.Error().
			Err(err).
			Msg("Failed to load template")
		return err
	}

	// Register our handler
	rg := r.Group("/embed")
	rg.GET("", serv.handleEmbed)

	// Create imageproxy and provide interface to shardedfilestore
	imgCache := NewImageProxyCache(serv.store, serv.log)
	imgProxy = imageproxy.NewProxy(nil, imgCache)

	// Attach imageproxy
	ic := r.Group("/image-cache/*id")
	ic.GET("", serv.handleImageCache)

	return nil
}

func (serv *UploadServer) handleImageCache(c *gin.Context) {
	r := c.Request
	r.URL.Path = strings.Replace(r.URL.Path, "/image-cache", "", -1)

	hash := getHash(r.URL.Path)

	serv.log.Debug().
		Msgf("Image request\n\turl: %s\n\thash: %s", r.URL.Path, hash)

	imgWaiterMutex.Lock()
	item, ok := imgWaiter[hash]
	if !ok {
		// This is the first client to request this url
		// create a waiter item and add it to the map
		item = &imgWaiterItem{
			url:     r.URL.Path,
			created: time.Now().Unix(),
		}
		// Other requests will wait on this waitgroup once the mutex is unlocked
		item.wg.Add(1)
		imgWaiter[hash] = item

		// Other requests are currently waiting for this mutex
		imgWaiterMutex.Unlock()

		// Pass this request to the image proxy
		imgProxy.ServeHTTP(c.Writer, c.Request)

		// Image proxy is done, store resulting status
		item.status = c.Writer.Status()

		// Ready for other clients to access this url
		item.wg.Done()
	} else {
		// Not the first client to request this url
		// We no longer need the mutex as we will use the waitgroup
		imgWaiterMutex.Unlock()
		item.wg.Wait()

		// Waitgroup is complete check if the first request was successful
		if item.status == 200 {
			// The first request was successful pass this request to the image proxy
			imgProxy.ServeHTTP(c.Writer, c.Request)
		} else {
			// First request failed return its status code to the client
			c.Status(item.status)
		}
	}
}

func (serv *UploadServer) handleEmbed(c *gin.Context) {
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

	serv.log.Debug().
		Msgf("Embed request\n\turl: %s\n\thash: %s", queryURL, hash)

	cacheMutex.Lock()
	item, ok := cache[hash]
	if !ok {
		// Cache miss create new cache item
		serv.log.Debug().
			Msgf("HTML cache miss")
		item = &cacheItem{
			url:     queryURL,
			html:    "",
			created: time.Now().Unix(),
		}

		// Add to waitgroup so other clients can wait for the embed result
		item.wg.Add(1)
		cache[hash] = item

		// Item added to cache, unlock so other requests can see the new item
		cacheMutex.Unlock()

		// Check if the url looks like an image
		if isImage.MatchString(queryURL) {
			item.html = getImageHTML(c, queryURL, height)
		}

		// Attempt to fetch oEmbed data
		embedItem := oEmbed.FindItem(queryURL)
		if embedItem != nil {
			options := oembed.Options{
				URL:       queryURL,
				MaxHeight: height,
				MaxWidth:  width,
			}
			info, err := embedItem.FetchOembed(options)
			if err != nil {
				serv.log.Error().
					Err(err).
					Msg("Unexpected error in oEmbed")
			} else if info.Status >= 300 {
				// oEmbed returned a bad status code
				serv.log.Debug().
					Msgf("Bad response code from oEmbed: %d", info.Status)
			} else if info.HTML != "" {
				// oEmbed returned embedable html
				serv.log.Debug().
					Msgf("oEmbed info:\n%s", info)
				item.html = info.HTML
			} else if info.Type == "photo" {
				// oEmbed returned a photo type the url should be an image
				serv.log.Debug().
					Msgf("oEmbed info:\n%s", info)
				item.html = getImageHTML(c, info.URL, height)
			}
		}

		// No embedable html, time to try noembed
		if item.html == "" {
			noEmbedResp, err := noEmbed.Get(queryURL)
			if err != nil {
				serv.log.Error().
					Err(err).
					Msg("Unexpected error in noEmbed")
			} else {
				item.html = noEmbedResp.HTML
			}
		}

		// Still no html send an error to the parent
		if item.html == "" {
			item.html = "<script>window.parent.postMessage({ error: true }, '*');</script>"
		}

		// Decrease the waitgroup so other requests can complete
		item.wg.Done()
	} else {
		// Cache hit unlock the cache
		serv.log.Debug().
			Msg("HTML cache hit")
		cacheMutex.Unlock()
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

func getProvidersCached(url string, filePath string, force bool) (*[]byte, error) {
	var err error
	if _, err = os.Stat(filePath); force || os.IsNotExist(err) {
		var httpResp *http.Response
		httpResp, err = http.Get(url)
		if err != nil {
			return nil, errors.New("Failed to fetch providers: " + err.Error())
		}
		defer httpResp.Body.Close()

		var rawJSON []byte
		rawJSON, err = ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return nil, errors.New("Failed to read providers: " + err.Error())
		}

		// Unmarshal to temp interface to ensure valid json
		var temp interface{}
		err = json.Unmarshal(rawJSON, &temp)
		if err != nil {
			return nil, errors.New("Failed to parse providers: " + err.Error())
		}

		// Data appears to be valid json open providers file for writing
		var file *os.File
		file, err = os.Create(filePath)
		if err != nil {
			return nil, errors.New("Failed to create providers file: " + err.Error())
		}
		defer file.Close()

		// Write providers.json
		_, err = file.Write(rawJSON)
		if err != nil {
			return nil, errors.New("Failed to write providers: " + err.Error())
		}

		return &rawJSON, nil
	} else if err != nil {
		return nil, errors.New("Failed to stat providers file: " + err.Error())
	}

	// Open existing providers file
	var file *os.File
	file, err = os.Open(filePath)
	if err != nil {
		return nil, errors.New("Failed to open providers: " + err.Error())
	}
	defer file.Close()

	// Read existing providers file
	var rawJSON []byte
	rawJSON, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.New("Failed to read providers: " + err.Error())
	}

	return &rawJSON, nil
}

func (serv *UploadServer) startCleanupTicker(cleanInterval, cacheMaxAge time.Duration) {
	cacheTicker = time.NewTicker(cleanInterval)
	go func() {
		for range cacheTicker.C {
			serv.cleanCache(cacheMaxAge)
		}
	}()
}

func (serv *UploadServer) cleanCache(cacheMaxAge time.Duration) {
	createdBefore := time.Now().Unix() - int64(cacheMaxAge.Seconds())

	// Find expired items in HTML cache
	var expired []string
	for hash, item := range cache {
		if item.created >= createdBefore {
			continue
		}
		expired = append(expired, hash)
	}

	// Find expired items in imgWaiter
	var expiredWaiters []string
	for hash, item := range imgWaiter {
		if item.created >= createdBefore {
			continue
		}
		expired = append(expiredWaiters, hash)
	}

	// Remove expired items from HTML cache
	if len(expired) > 0 {
		serv.log.Debug().
			Msgf("Cleaning %d item from HTML cache", len(expired))

		cacheMutex.Lock()
		for _, hash := range expired {
			if hash == "" {
				break
			}
			log.Println("Deleting cache item: " + hash)
			delete(cache, hash)
		}
		cacheMutex.Unlock()
	}

	// Remove expired items from img waiter
	if len(expiredWaiters) > 0 {
		serv.log.Debug().
			Msgf("Cleaning %d item from img waiter cache", len(expired))

		cacheMutex.Lock()
		for _, hash := range expiredWaiters {
			if hash == "" {
				break
			}
			log.Println("Deleting cache item: " + hash)
			delete(imgWaiter, hash)
		}
		cacheMutex.Unlock()
	}
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
