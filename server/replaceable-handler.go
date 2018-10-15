package server

import (
	"net/http"
)

type ReplaceableHandler struct {
	Handler http.Handler
}

func (m *ReplaceableHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.Handler.ServeHTTP(w, req)
}
