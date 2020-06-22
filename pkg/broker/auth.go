package broker

import (
	"context"
	"net/http"
	"strings"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/gorilla/mux"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
)

func authMiddleware(auth credentials.BrokerAuth) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if auth.Username != username || auth.Password != password {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SimpleAuthMiddleware is used to validate and parse Atlas API credentials passed
// using basic auth. The credentials parsed into an Atlas client which is
// attached to the request context. This client can later be retrieved by the
// broker from the context.
func simpleAuthMiddleware(baseURL string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, privKey, ok := r.BasicAuth()

			// The username contains both the group ID and public key
			// formatted as "<PUBLIC_KEY>@<GROUP_ID>".
			splitUsername := strings.Split(username, "@")

			// If the credentials are invalid we respond with 401 Unauthorized.
			// The username needs have the correct format and the password must
			// not be empty.
			validUsername := len(splitUsername) == 2
			validPassword := privKey != ""
			if !(ok && validUsername && validPassword) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			pubKey := splitUsername[0]
			gid := splitUsername[1]

			// Create a new client with the extracted API credentials and
			// attach it to the request context.
			hc, err := digest.NewTransport(pubKey, privKey).Client()
			if err != nil {
				panic(err)
			}

			client, err := mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL+"/api/atlas/v1.0/"))
			if err != nil {
				panic(err)
			}

			ctx := context.WithValue(r.Context(), ContextKeyAtlasClient, client)
			ctx = context.WithValue(ctx, ContextKeyGroupID, gid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
