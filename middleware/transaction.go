package middleware

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	log "github.com/sirupsen/logrus"
)

func Transactional(db *sqlx.DB) app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := log.WithFields(log.Fields{
				"topic":   "taos",
				"package": "middleware",
				"context": "transaction",
				"event":   "newtransaction",
			})

			tx, err := db.Beginx()
			if err != nil {
				logger.Panic("Could not create transaction")
			}
			logger.Debug("transaction created")

			rc := app.GetRequestContext(r)
			rc.SetTx(tx)

			logger.Debug("attached transaction to request context")

			var txe error
			defer func() {
				if err := recover(); err != nil {
					txe = tx.Rollback()
					logger.Error("transaction reverted")
				} else {
					txe = tx.Commit()
					logger.Debug("transaction commited")
				}
			}()

			if txe != nil {
				panic(err)
			}

			ctx := context.WithValue(r.Context(), app.RequestContextKey, rc)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
