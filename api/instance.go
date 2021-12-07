package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
)

func (a *API) loadInstance(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	instanceID, err := uuid.FromString(chi.URLParam(r, "instance_id"))
	if err != nil {
		return nil, badRequestError("Invalid instance ID")
	}
	logEntrySetField(r, "instance_id", instanceID)

	i, err := models.GetInstance(a.db, instanceID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("Instance not found")
		}
		return nil, internalServerError("Database error loading instance").WithInternalError(err)
	}

	return withInstance(r.Context(), i), nil
}

func (a *API) GetAppManifest(w http.ResponseWriter, r *http.Request) error {
	// TODO update to real manifest
	return sendJSON(w, http.StatusOK, map[string]string{
		"version":     a.version,
		"name":        "GoTrue",
		"description": "GoTrue is a user registration and authentication API",
	})
}

type InstanceRequestParams struct {
	UUID       uuid.UUID           `json:"uuid"`
	BaseConfig *conf.Configuration `json:"config"`
}

type InstanceResponse struct {
	models.Instance
	Endpoint string `json:"endpoint"`
	State    string `json:"state"`
}

func (a *API) CreateInstance(w http.ResponseWriter, r *http.Request) error {
	params := InstanceRequestParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return badRequestError("Error decoding params: %v", err)
	}

	_, err := models.GetInstanceByUUID(a.db, params.UUID)
	if err != nil {
		if !models.IsNotFoundError(err) {
			return internalServerError("Database error looking up instance").WithInternalError(err)
		}
	} else {
		return badRequestError("An instance with that UUID already exists")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "Error generating id")
	}

	i := models.Instance{
		ID:         id,
		UUID:       params.UUID,
		BaseConfig: params.BaseConfig,
	}
	if err = a.db.Create(&i); err != nil {
		return internalServerError("Database error creating instance").WithInternalError(err)
	}

	// hide pass in response
	if i.BaseConfig != nil {
		i.BaseConfig.SMTP.Pass = ""
	}

	resp := InstanceResponse{
		Instance: i,
		Endpoint: a.config.API.Endpoint,
		State:    "active",
	}
	return sendJSON(w, http.StatusCreated, resp)
}

func (a *API) GetInstance(w http.ResponseWriter, r *http.Request) error {
	i := getInstance(r.Context())
	if i.BaseConfig != nil {
		i.BaseConfig.SMTP.Pass = ""
	}
	return sendJSON(w, http.StatusOK, i)
}

func (a *API) UpdateInstance(w http.ResponseWriter, r *http.Request) error {
	i := getInstance(r.Context())

	params := InstanceRequestParams{BaseConfig: i.BaseConfig}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return badRequestError("Error decoding params: %v", err)
	}

	if err := i.UpdateConfig(a.db, params.BaseConfig); err != nil {
		return internalServerError("Database error updating instance").WithInternalError(err)
	}

	// Hide SMTP credential from response
	if i.BaseConfig != nil {
		i.BaseConfig.SMTP.Pass = ""
	}
	return sendJSON(w, http.StatusOK, i)
}

func (a *API) DeleteInstance(w http.ResponseWriter, r *http.Request) error {
	i := getInstance(r.Context())
	if err := models.DeleteInstance(a.db, i); err != nil {
		return internalServerError("Database error deleting instance").WithInternalError(err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
