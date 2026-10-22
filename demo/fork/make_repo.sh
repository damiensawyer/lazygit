#!/bin/bash
# Builds a small, deterministic repo for recording demos.
set -e
R=$1
rm -rf "$R"; mkdir -p "$R"; cd "$R"
git init -q -b main
git config user.name "Yang Wen-li"; git config user.email yang@example.com
git config commit.gpgsign false; git config tag.gpgsign false
n=0
commit() { n=$((n+1)); GIT_AUTHOR_DATE="2026-09-$(printf %02d $((n%28+1)))T10:00:00" GIT_COMMITTER_DATE="2026-09-$(printf %02d $((n%28+1)))T10:00:00" git commit -q -m "$1"; }
mkdir -p cmd internal/server internal/store docs
cat > go.mod <<'X'
module example.com/fleet

go 1.23
X
cat > cmd/main.go <<'X'
package main

import (
	"fmt"
	"os"

	"example.com/fleet/internal/server"
)

func main() {
	srv := server.New(":8080")
	if err := srv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
X
cat > internal/server/server.go <<'X'
package server

import (
	"log"
	"net/http"
)

// Server serves the fleet API.
type Server struct {
	addr string
	mux  *http.ServeMux
}

func New(addr string) *Server {
	s := &Server{addr: addr, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.health)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) Run() error {
	log.Printf("listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}
X
cat > internal/store/store.go <<'X'
package store

import "sync"

// Store keeps ships in memory.
type Store struct {
	mu    sync.Mutex
	ships map[string]Ship
}

type Ship struct {
	Name    string
	Class   string
	Crew    int
}

func New() *Store {
	return &Store{ships: map[string]Ship{}}
}

func (s *Store) Put(ship Ship) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ships[ship.Name] = ship
}
X
printf '# Fleet\n\nA tiny fleet tracking service.\n' > README.md
printf '# Architecture\n\nThe server talks to an in-memory store.\n' > docs/architecture.md
git add -A; commit "Initial commit"
git tag v0.1.0
feature() { # name, file, text, msg1, msg2
	git checkout -q -b "$1"
	echo "$3" >> "$2"; git add -A; commit "$4"
	echo "// $5" >> "$2"; git add -A; commit "$5"
	git checkout -q main
	git merge -q --no-ff "$1" -m "Merge branch '$1'"
	n=$((n+1))
}
feature add-ships-endpoint internal/server/server.go "// TODO: list ships" "Add a ships endpoint" "Return ships as JSON"
feature store-delete internal/store/store.go "// Delete removes a ship." "Add Store.Delete" "Test Store.Delete"
feature docs-api docs/architecture.md "Endpoints are documented in api.md." "Document the API" "Fix typos in API docs"
git tag v0.2.0
feature graceful-shutdown cmd/main.go "// graceful shutdown" "Shut down gracefully on SIGTERM" "Add shutdown timeout flag"
feature crew-validation internal/store/store.go "// Validate checks crew counts." "Validate crew counts" "Reject negative crew"
git tag v0.3.0
feature request-logging internal/server/server.go "// log requests" "Log every request" "Include latency in request logs"
feature ship-classes internal/store/store.go "// classes" "Add ship classes" "Default unknown classes to scout"
feature readme-badges README.md "Badges go here." "Add CI badge" "Add license badge"
git checkout -q -b fleet-metrics
echo "// metrics" >> internal/server/server.go; git add -A; commit "Expose Prometheus metrics"
echo "// metrics 2" >> internal/server/server.go; git add -A; commit "Count requests per route"
git checkout -q main
echo "Run with \`go run ./cmd\`." >> README.md; git add -A; commit "Explain how to run the server"
git checkout -q main
git branch -q -D add-ships-endpoint store-delete docs-api graceful-shutdown crew-validation request-logging ship-classes readme-badges
git tag v0.4.0 HEAD~1
sed -i "s#w.WriteHeader(http.StatusOK)#w.Header().Set(\"Content-Type\", \"text/plain\")\n\tw.WriteHeader(http.StatusOK)#" internal/server/server.go
sed -i "s#Crew    int#Crew    int\n\tHome    string#" internal/store/store.go
