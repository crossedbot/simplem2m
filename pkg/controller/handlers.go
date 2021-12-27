package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crossedbot/common/golang/logger"
	"github.com/crossedbot/common/golang/server"

	"github.com/crossedbot/simplem2m/pkg/models"
)

func Authenticate(w http.ResponseWriter, r *http.Request, p server.Parameters) {
	var login models.ClientLogin
	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrFailedConversionCode,
			Message: fmt.Sprintf(
				"Failed to parse request body; %s",
				err,
			),
		}, http.StatusBadRequest)
		return
	}
	if login.ClientId == "" {
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to login; %s",
				ErrorClientIdRequired,
			),
		}, http.StatusBadRequest)
		return
	}
	if login.ClientSecret == "" {
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to login; %s",
				ErrorClientSecretRequired,
			),
		}, http.StatusBadRequest)
		return
	}
	tkn, err := V1().Authenticate(login)
	if err == ErrorBadCredentials {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to login; %s",
				err,
			),
		}, http.StatusBadRequest)
		return
	} else if err != nil {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to login; %s",
				err,
			),
		}, http.StatusInternalServerError)
		return
	}
	server.JsonResponse(w, &tkn, http.StatusOK)
}

func Register(w http.ResponseWriter, r *http.Request, p server.Parameters) {
	var client models.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrFailedConversionCode,
			Message: fmt.Sprintf(
				"Failed to parse request body; %s",
				err,
			),
		}, http.StatusBadRequest)
		return
	}
	tkn, err := V1().Register(client)
	if err != nil {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to signup; %s",
				err,
			),
		}, http.StatusInternalServerError)
		return
	}
	server.JsonResponse(w, &tkn, http.StatusCreated)
}

func GetJwk(w http.ResponseWriter, r *http.Request, p server.Parameters) {
	jwks, err := V1().GetJwks()
	if err != nil {
		logger.Error(err)
		server.JsonResponse(w, server.Error{
			Code: server.ErrProcessingRequestCode,
			Message: fmt.Sprintf(
				"Failed to retrieve jwk.json; %s",
				err,
			),
		}, http.StatusInternalServerError)
		return
	}
	server.JsonResponse(w, &jwks, http.StatusOK)
}
