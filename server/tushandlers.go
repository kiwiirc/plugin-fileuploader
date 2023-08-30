package server

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	goLog "log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/kiwiirc/plugin-fileuploader/events"
	"github.com/kiwiirc/plugin-fileuploader/logging"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	tusd "github.com/tus/tusd/pkg/handler"
)

func customizedCors(serv *UploadServer) gin.HandlerFunc {
	// convert slice values to keys of map for "contains" test
	originSet := make(map[string]struct{}, len(serv.cfg.Server.CorsOrigins))
	allowAll := false
	exists := struct{}{}
	for _, origin := range serv.cfg.Server.CorsOrigins {
		if origin == "*" {
			allowAll = true
			continue
		}
		originSet[origin] = exists
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		respHeader := c.Writer.Header()

		// only allow the origin if it's in the list from the config
		if allowAll && origin != "" {
			respHeader.Set("Access-Control-Allow-Origin", origin)
		} else if _, ok := originSet[origin]; ok {
			respHeader.Set("Access-Control-Allow-Origin", origin)
		} else {
			respHeader.Del("Access-Control-Allow-Origin")
			if c.Request.Method != "HEAD" && c.Request.Method != "GET" {
				// Don't log unknown cors origin for HEAD or GET requests
				serv.log.Warn().Str("origin", origin).Msg("Unknown cors origin")
			}
		}

		// lets the user-agent know the response can vary depending on the origin of the request.
		// ensures correct behaviour of browser cache.
		respHeader.Add("Vary", "Origin")
	}
}

func (serv *UploadServer) fileuploaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" && c.Request.Method != "DELETE" {
			// Metadata is only required for POST and DELETE requests
			return
		}

		metadata := tusd.ParseMetadataHeader(c.Request.Header.Get("Upload-Metadata"))

		// ensure the user does not try to provide their own RemoteIP
		delete(metadata, "RemoteIP")

		// determine the originating IP
		remoteIP, err := serv.getDirectOrForwardedRemoteIP(c.Request)
		if err != nil {
			if addrErr, ok := err.(*net.AddrError); ok {
				c.AbortWithError(http.StatusInternalServerError, addrErr).SetType(gin.ErrorTypePrivate)
			} else {
				c.AbortWithError(http.StatusNotAcceptable, err)
			}
			return
		}

		// add RemoteIP to metadata
		metadata["RemoteIP"] = remoteIP

		err = serv.processJwt(metadata)
		if err != nil {
			// Jwt failures are none fatal, but will result in the uploaded being treated as anonymous
			// Stick a warning in the log to help with debugging
			serv.log.Warn().
				Err(err).
				Str("extjwt", metadata["extjwt"]).
				Msg("Failed to process EXTJWT")
			err = nil
		}

		// extjwt is no longer needed, remove so it does not get stored with the file info
		delete(metadata, "extjwt")

		// Update metadata with any changes that have been made
		c.Request.Header.Set("Upload-Metadata", tusd.SerializeMetadataHeader(metadata))

		// store metadata in gin context so it does not need to be parsed again
		c.Set("metadata", metadata)
	}
}

func (serv *UploadServer) registerTusHandlers(r *gin.Engine, store *shardedfilestore.ShardedFileStore) error {
	maximumUploadSize := serv.cfg.Storage.MaximumUploadSize
	serv.log.Debug().Str("size", maximumUploadSize.String()).Msg("Using upload limit")

	config := tusd.Config{
		BasePath:                serv.cfg.Server.BasePath,
		StoreComposer:           serv.composer,
		MaxSize:                 int64(maximumUploadSize.Bytes()),
		Logger:                  goLog.New(ioutil.Discard, "", 0),
		NotifyCompleteUploads:   true,
		NotifyCreatedUploads:    true,
		NotifyTerminatedUploads: true,
		NotifyUploadProgress:    true,
	}

	routePrefix, err := routePrefixFromBasePath(serv.cfg.Server.BasePath)
	if err != nil {
		return err
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return err
	}

	// create event broadcaster
	serv.tusEventBroadcaster = events.NewTusEventBroadcaster(handler)

	// attach logger
	go logging.TusdLogger(serv.log, serv.tusEventBroadcaster)

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	tusdMiddleware := gin.WrapH(handler.Middleware(noopHandler))

	rg := r.Group(routePrefix)
	rg.Use(tusdMiddleware)
	rg.Use(customizedCors(serv))
	rg.Use(serv.fileuploaderMiddleware())
	rg.POST("", serv.postFile(handler))

	// Register a dummy handler for OPTIONS, without this the middleware's would not be called
	rg.OPTIONS("*any", gin.WrapH(noopHandler))

	headFile := gin.WrapF(handler.HeadFile)
	rg.HEAD(":id", headFile)
	rg.HEAD(":id/:filename", rewritePath(headFile, routePrefix))

	getFile := serv.getFile(handler, store)
	rg.GET(":id", getFile)
	rg.GET(":id/:filename", rewritePath(getFile, routePrefix))

	patchFile := gin.WrapF(handler.PatchFile)
	rg.PATCH(":id", patchFile)
	rg.PATCH(":id/:filename", rewritePath(patchFile, routePrefix))

	// Only attach the DELETE handler if the Terminate() method is provided
	if serv.composer.UsesTerminater {
		delFile := serv.delFile(handler)
		rg.DELETE(":id", delFile)
		rg.DELETE(":id/:filename", rewritePath(delFile, routePrefix))
	}

	return nil
}

func (serv *UploadServer) postFile(handler *tusd.UnroutedHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata := c.MustGet("metadata").(map[string]string)

		if serv.cfg.Server.RequireJwtAccount {
			if metadata["account"] == "" {
				c.Error(errors.New("Missing JWT account")).SetType(gin.ErrorTypePublic)
				c.AbortWithStatusJSON(http.StatusUnauthorized, "Account required")
				return
			}
		}

		handler.PostFile(c.Writer, c.Request)
	}
}

func (serv *UploadServer) delFile(handler *tusd.UnroutedHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		metadata := c.MustGet("metadata").(map[string]string)

		var uploaderIP, jwtAccount, jwtIssuer string
		row := serv.DBConn.DB.QueryRow(`SELECT uploader_ip, jwt_account, jwt_issuer FROM uploads WHERE id = ?`, id)
		err := row.Scan(&uploaderIP, &jwtAccount, &jwtIssuer)

		// no finalized upload exists
		if err == sql.ErrNoRows {
			c.AbortWithStatus(http.StatusNotFound)
			return
		} else if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePrivate)
			return
		}

		if jwtAccount != "" && (jwtAccount != metadata["account"] || jwtIssuer != metadata["issuer"]) {
			// The upload was created by an identified account that does not match this requests account
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else if uploaderIP == "" || uploaderIP != metadata["RemoteIP"] {
			// The upload was created by an anonymous user that does not match this requests ip address
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		handler.DelFile(c.Writer, c.Request)
	}
}

func (serv *UploadServer) getSecretForToken(token *jwt.Token) (interface{}, error) {
	// Don't forget to validate the alg is what you expect:
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("Failed to get claims")
	}

	issuer, ok := claims["iss"]
	if !ok {
		return nil, errors.New("Issuer field 'iss' missing from JWT")
	}

	issuerStr, ok := issuer.(string)
	if !ok {
		return nil, errors.New("Failed to coerce issuer to string")
	}

	secret, ok := serv.cfg.JwtSecretsByIssuer[issuerStr]
	if !ok {
		// Attempt to get fallback issuer
		secret, ok = serv.cfg.JwtSecretsByIssuer["*"]

		if !ok {
			return nil, fmt.Errorf("Issuer %#v not configured", issuerStr)
		} else {
			serv.log.Warn().
				Msg(fmt.Sprintf("Issuer %#v not configured, used fallback", issuerStr))
		}
	}

	return []byte(secret), nil
}

func (serv *UploadServer) processJwt(metadata map[string]string) (err error) {
	// ensure the client doesn't attempt to specify their own account/issuer fields
	delete(metadata, "issuer")
	delete(metadata, "account")

	tokenString := metadata["extjwt"]
	if tokenString == "" {
		return nil
	}

	token, err := jwt.Parse(tokenString, serv.getSecretForToken)
	if err != nil {
		return err
	}

	if !token.Valid {
		return errors.New("invalid jwt")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("no jwt claims")
	}

	if issuer, ok := claims["iss"].(string); ok {
		metadata["issuer"] = issuer
	}
	if account, ok := claims["account"].(string); ok {
		metadata["account"] = account
	}

	return nil
}

// ErrInvalidXForwardedFor occurs if the X-Forwarded-For header is trusted but invalid
var ErrInvalidXForwardedFor = errors.New("Failed to parse IP from X-Forwarded-For header")

func (serv *UploadServer) getDirectOrForwardedRemoteIP(req *http.Request) (string, error) {
	// extract direct IP
	remoteIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		serv.log.Error().
			Err(err).
			Msg("Could not split address into host and port")
		return "", err
	}

	// use X-Forwarded-For header if direct IP is a trusted reverse proxy
	if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if serv.remoteIPisTrusted(net.ParseIP(remoteIP)) {
			// We do not check intermediary proxies against the whitelist.
			// If a trusted proxy is appending to and forwarding the value of the
			// header it is receiving, that is an implicit expression of trust
			// which we will honor transitively.

			// take the first comma delimited address
			// this is the original client address
			parts := strings.Split(forwardedFor, ",")
			forwardedForClient := strings.TrimSpace(parts[0])
			forwardedForIP := net.ParseIP(forwardedForClient)
			if forwardedForIP == nil {
				err := ErrInvalidXForwardedFor
				serv.log.Error().
					Err(err).
					Str("client", forwardedForClient).
					Str("remoteIP", remoteIP).
					Msg("Couldn't use trusted X-Forwarded-For header")
				return "", err
			}
			return forwardedForIP.String(), nil
		}
		serv.log.Warn().
			Str("X-Forwarded-For", forwardedFor).
			Str("remoteIP", remoteIP).
			Msg("Untrusted remote attempted to override stored IP")
	}

	// otherwise use direct IP
	return remoteIP, nil
}

func (serv *UploadServer) remoteIPisTrusted(remoteIP net.IP) bool {
	// check if remote IP is a trusted reverse proxy
	for _, trustedNet := range serv.cfg.Server.TrustedReverseProxyRanges {
		if trustedNet.Contains(remoteIP) {
			return true
		}
	}
	return false
}

func routePrefixFromBasePath(basePath string) (string, error) {
	url, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}

	return url.Path, nil
}

func rewritePath(handler gin.HandlerFunc, routePrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// rewrite request path to ":id" route pattern
		c.Request.URL.Path = path.Join(routePrefix, url.PathEscape(c.Param("id")))

		// call the normal handler
		handler(c)
	}
}
