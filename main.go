package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dbongo/hackapp/datastore"
	"github.com/dbongo/hackapp/datastore/database"
	"github.com/dbongo/hackapp/middleware"
	"github.com/dbongo/hackapp/router"
	"gopkg.in/mgo.v2"

	"code.google.com/p/go.net/context"
	webctx "github.com/goji/context"
	"github.com/zenazn/goji/web"
)

var (
	port = flag.String("p", ":3000", "server port")
	db   *mgo.Database
)

func main() {
	flag.Parse()

	// create the db
	db = database.New()
	defer db.Session.Close()

	// create the router and add middleware
	mux := router.New()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.Options)
	mux.Use(ContextMiddleware)
	mux.Use(middleware.SetHeaders)
	mux.Use(middleware.HTTPLogger)
	mux.Use(middleware.SetUser)
	mux.Use(middleware.Recovery)
	http.Handle("/api/", mux)

	if err := http.ListenAndServe(*port, nil); err != nil {
		log.Fatal(err)
	}
}

// ContextMiddleware creates a new go.net/context and injects into the current goji context.
func ContextMiddleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.Background()
		ctx = datastore.NewContext(ctx, database.NewDatastore(db))
		webctx.Set(c, ctx)
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
