// Command routes prints every API route as "METHOD /pattern", one per line.
// scripts/e2e-endpoints.sh uses it to check that every route was exercised.
package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	apphttp "github.com/SukGebeya/Backend/internal/http"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Handlers are only constructed, never called, so empty dependencies are enough.
	router := apphttp.NewRouter(apphttp.Deps{})

	var routes []string
	err := chi.Walk(router.Mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+route)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.Strings(routes)
	for _, r := range routes {
		fmt.Println(r)
	}
}
