package app

import (
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"net/http"
	"time"
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
			rc := NewRequestContext(r.Context(), r)
			ctx := context.WithValue(r.Context(), RequestContextKey, rc)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type RequestContext struct {
	tx          *sqlx.Tx
	rollback    bool
	requestTime time.Time
	requestID   string
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

func (rs *RequestContext) Rollback() bool {
	return rs.rollback
}

func (rs *RequestContext) SetRollback(v bool) {
	rs.rollback = v
}

func (rs *RequestContext) RequestTime() time.Time {
	return rs.requestTime
}

func (rs *RequestContext) RequestID() string {
	return rs.requestID
}
