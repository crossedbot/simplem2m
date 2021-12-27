package controller

import (
	"net/http"

	"github.com/crossedbot/common/golang/server"
)

var Routes = []server.Route{
	server.Route{
		Handler:          Authenticate,
		Method:           http.MethodPost,
		Path:             "/m2m/authenticate",
		ResponseSettings: []server.ResponseSetting{},
	},
	server.Route{
		Handler:          Register,
		Method:           http.MethodPost,
		Path:             "/m2m/register",
		ResponseSettings: []server.ResponseSetting{},
	},
	server.Route{
		Handler:          GetJwk,
		Method:           http.MethodGet,
		Path:             "/.well-known/jwks.json",
		ResponseSettings: []server.ResponseSetting{},
	},
}
