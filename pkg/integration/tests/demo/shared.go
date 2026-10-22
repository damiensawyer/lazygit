package demo

import (
	"maps"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// Gives us nicer colours when we generate a git repo history with `shell.CreateRepoHistory()`
func setGeneratedAuthorColours(config *config.AppConfig) {
	config.GetUserConfig().Gui.Theme.AuthorColors = map[string]string{
		"Fredrica Greenhill": "#fb5aa3",
		"Oscar Reuenthal":    "#86c82f",
		"Paul Oberstein":     "#ffd500",
		"Siegfried Kircheis": "#fe7e11",
		"Yang Wen-li":        "#8e3ccb",
	}
}

func setDefaultDemoConfig(config *config.AppConfig) {
	// demos look much nicer with icons shown
	config.GetUserConfig().Gui.NerdFontsVersion = "3"
	// The terminal that we record demos in doesn't tell us that it can draw the
	// detailed graph; demo/settings.tape gives it a font for it
	config.GetUserConfig().Gui.CommitGraphStyle = "detailed"
}

// The demos of this fork's features share a small Go project, with a history
// of merged feature branches and release tags, and a couple of uncommitted
// changes.

const fleetMainGo = `package main

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
`

const fleetServerGo = `package server

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
`

const fleetStoreGo = `package store

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
`

func setFleetDemoConfig(config *config.AppConfig) {
	setDefaultDemoConfig(config)
	config.GetUserConfig().Gui.ExpandFocusedSidePanel = true
	config.GetUserConfig().Gui.ShowCommandLog = false
	config.GetUserConfig().Gui.Theme.AuthorColors = map[string]string{
		"Yang Wen-li": "#8e3ccb",
	}
}

type fleetRepo struct {
	shell *Shell
	files map[string]string
}

func setupFleetRepo(shell *Shell) {
	repo := &fleetRepo{shell: shell, files: map[string]string{}}
	repo.build()
}

func (self *fleetRepo) write(path string, content string) {
	self.files[path] = content
	self.shell.CreateFileAndAdd(path, content)
}

func (self *fleetRepo) appendLine(path string, line string) {
	self.write(path, self.files[path]+line+"\n")
}

// a feature branch with two commits, merged into main
func (self *fleetRepo) feature(branch string, path string, line string, firstCommit string, secondCommit string) {
	self.shell.NewBranch(branch)
	self.appendLine(path, line)
	self.shell.Commit(firstCommit)
	self.appendLine(path, "// "+secondCommit)
	self.shell.Commit(secondCommit)
	self.shell.Checkout("main")
	self.shell.Merge(branch)
}

func (self *fleetRepo) build() {
	shell := self.shell
	shell.SetAuthor("Yang Wen-li", "yang@example.com")

	self.write("go.mod", "module example.com/fleet\n\ngo 1.23\n")
	self.write("cmd/main.go", fleetMainGo)
	self.write("internal/server/server.go", fleetServerGo)
	self.write("internal/store/store.go", fleetStoreGo)
	self.write("README.md", "# Fleet\n\nA tiny fleet tracking service.\n")
	self.write("docs/architecture.md", "# Architecture\n\nThe server talks to an in-memory store.\n")
	shell.Commit("Initial commit")
	shell.RenameCurrentBranch("main")
	shell.CreateLightweightTag("v0.1.0", "HEAD")

	self.feature("add-ships-endpoint", "internal/server/server.go", "// TODO: list ships", "Add a ships endpoint", "Return ships as JSON")
	self.feature("store-delete", "internal/store/store.go", "// Delete removes a ship.", "Add Store.Delete", "Test Store.Delete")
	self.feature("docs-api", "docs/architecture.md", "Endpoints are documented in api.md.", "Document the API", "Fix typos in API docs")
	shell.CreateLightweightTag("v0.2.0", "HEAD")

	self.feature("graceful-shutdown", "cmd/main.go", "// graceful shutdown", "Shut down gracefully on SIGTERM", "Add shutdown timeout flag")
	self.feature("crew-validation", "internal/store/store.go", "// Validate checks crew counts.", "Validate crew counts", "Reject negative crew")
	shell.CreateLightweightTag("v0.3.0", "HEAD")

	self.feature("request-logging", "internal/server/server.go", "// log requests", "Log every request", "Include latency in request logs")
	self.feature("ship-classes", "internal/store/store.go", "// classes", "Add ship classes", "Default unknown classes to scout")
	self.feature("readme-badges", "README.md", "Badges go here.", "Add CI badge", "Add license badge")

	// a branch that isn't merged, so its changes mustn't reach main's files
	mainFiles := maps.Clone(self.files)
	shell.NewBranch("fleet-metrics")
	self.appendLine("internal/server/server.go", "// metrics")
	shell.Commit("Expose Prometheus metrics")
	self.appendLine("internal/server/server.go", "// metrics 2")
	shell.Commit("Count requests per route")
	shell.Checkout("main")
	self.files = mainFiles

	self.appendLine("README.md", "Run with `go run ./cmd`.")
	shell.Commit("Explain how to run the server")
	shell.CreateLightweightTag("v0.4.0", "HEAD~1")

	for _, branch := range []string{
		"add-ships-endpoint", "store-delete", "docs-api", "graceful-shutdown",
		"crew-validation", "request-logging", "ship-classes", "readme-badges",
	} {
		shell.RemoveBranch(branch)
	}

	// uncommitted changes
	shell.UpdateFile("internal/server/server.go", strings.Replace(self.files["internal/server/server.go"],
		"\tw.WriteHeader(http.StatusOK)",
		"\tw.Header().Set(\"Content-Type\", \"text/plain\")\n\tw.WriteHeader(http.StatusOK)", 1))
	shell.UpdateFile("internal/store/store.go", strings.Replace(self.files["internal/store/store.go"],
		"\tCrew    int\n",
		"\tCrew    int\n\tHome    string\n", 1))
}
