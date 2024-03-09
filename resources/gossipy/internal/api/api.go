package api

import (
	"encoding/json"
	"gossipy/internal/cluster"
	"gossipy/internal/store"
	"gossipy/pkg/version"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type API struct {
	router  *mux.Router
	cluster *cluster.Cluster
	store   *store.KeyValueStore
}

func NewAPI(cluster *cluster.Cluster, store *store.KeyValueStore) *API {
	api := &API{
		router:  mux.NewRouter(),
		cluster: cluster,
		store:   store,
	}

	api.setupRoutes()
	return api
}

func (api *API) setupRoutes() {
	api.router.HandleFunc("/{key}/{value}", api.setHandler).Methods("PUT")
	api.router.HandleFunc("/version", api.versionHandler).Methods("GET") // slightly overlapping GET namespace ¯\_(ツ)_/¯
	api.router.HandleFunc("/{key}", api.getHandler).Methods("GET")
	api.router.HandleFunc("/{key}", api.deleteHandler).Methods("DELETE")
}

func (api *API) setHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("setHandler")

	// extract our key/value
	vars := mux.Vars(r)

	// update our store
	api.store.Set(vars["key"], vars["value"])

	// update our cluster
	api.cluster.Broadcasts.QueueBroadcast(cluster.BroadcastKVOperation{
		Operation: "set",
		Key:       vars["key"],
		Value:     vars["value"],
	})

	// respond
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		vars["key"]: vars["value"],
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("There was an error encoding the data struct")
	}
}

func (api *API) getHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("getHandler")

	// extract our key
	vars := mux.Vars(r)

	// update our store
	value, found := api.store.Get(vars["key"])

	// respond
	if found {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]string{
			vars["key"]: value,
		})
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("There was an error encoding the data struct")
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (api *API) deleteHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("deleteHandler")

	// extract our key
	vars := mux.Vars(r)

	// update the store
	api.store.Delete(vars["key"])

	// update our cluster
	api.cluster.Broadcasts.QueueBroadcast(cluster.BroadcastKVOperation{
		Operation: "delete",
		Key:       vars["key"],
		Value:     "",
	})

	// respond
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
}

// our internal version handler
func (api *API) versionHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("versionHandler")

	// respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&version.VersionResponse{
		Version:   version.Version,
		BuildHash: version.BuildHash,
		BuildDate: version.BuildDate,
	})
}

func (api *API) Run(addr string) error {
	log.Info().Str("listen", addr).Msg("API listening")
	return http.ListenAndServe(addr, api.router)
}
