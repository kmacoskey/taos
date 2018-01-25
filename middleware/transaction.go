package middleware

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/kmacoskey/taos/app"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func Transactional(db *sqlx.DB) app.Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			txLogger := log.WithFields(log.Fields{
				"topic": "taos",
				"event": "transaction",
			})

			tx, err := db.Beginx()
			if err != nil {
				txLogger.Panic("Could not create transaction")
			}
			txLogger.Debug("transaction created")

			rc := app.GetRequestContext(r)
			rc.SetTx(tx)

			var txe error
			defer func() {
				if err := recover(); err != nil {
					txe = tx.Rollback()
					txLogger.Error("transaction reverted")
				} else {
					txe = tx.Commit()
					txLogger.Debug("transaction commited")
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
