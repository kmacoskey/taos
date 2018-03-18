package app

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// The RequestContext is passed as an *http.Request WithValue() for
// the hardcoded "request" key. If ever more keys are used, this should
// be immediately refactored to be more flexible and unique
var RequestContextKey string = "request"

// Middleware to add information contextual to the request by including
// it in the *http.Request context
// The requestContext struct is available as the value of the "request" key
func WithRequestContext() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "app",
				"context": "requestcontext",
				"event":   "newrequest",
			})
			rc := NewRequestContext(r.Context(), r)
			ctx := context.WithValue(r.Context(), RequestContextKey, rc)
			logger.Debug("created new request context")
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type RequestContext struct {
	tx              *sqlx.Tx
	terraformConfig []byte
	rollback        bool
	requestTime     time.Time
	requestID       string
}

func NewRequestContext(ctx context.Context, req *http.Request) RequestContext {
	uuid := uuid.Must(uuid.NewRandom())

	rc := RequestContext{
		requestID:   uuid.String(),
		requestTime: time.Now(),
	}

	return rc
}

func GetRequestContext(r *http.Request) RequestContext {
	return r.Context().Value(RequestContextKey).(RequestContext)
}

func (rs *RequestContext) Tx() *sqlx.Tx {
	return rs.tx
}

func (rs *RequestContext) SetTx(tx *sqlx.Tx) {
	rs.tx = tx
}

func (rs *RequestContext) TerraformConfig() []byte {
	return rs.terraformConfig
}

func (rs *RequestContext) SetTerraformConfig(tfcfg []byte) {
	rs.terraformConfig = tfcfg
}

func (rs *RequestContext) RequestTime() time.Time {
	return rs.requestTime
}

func (rs *RequestContext) RequestId() string {
	return rs.requestID
}
