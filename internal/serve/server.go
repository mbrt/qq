package serve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/mbrt/qq/internal/config"
	"github.com/mbrt/qq/internal/index"
)

func NewServer(idx *index.Index, uiPath string, dirs []config.Directory) (*Server, error) {
	api := &apiHandler{index: idx, dirs: dirs}

	tmplFS := os.DirFS(path.Join(uiPath, "templates"))
	dyn, err := newDynamicHandler(api, tmplFS)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic handler: %w", err)
	}

	staticHandler := staticFSHandler(path.Join(uiPath, "assets"))

	mux := http.NewServeMux()
	s := &Server{
		mux:     mux,
		api:     api,
		dynamic: dyn,
	}
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /read/{id...}", s.handleRead)
	mux.Handle("GET /assets/", http.StripPrefix("/assets", staticHandler))

	return s, nil
}

type Server struct {
	mux     *http.ServeMux
	api     *apiHandler
	dynamic dynamicHandler
}

func (s *Server) Serve(ctx context.Context, port int) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := s.dynamic.Home(r.Context(), w); err != nil {
		writeErr(w, err)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return
	}
	query := r.Form.Get("q")
	var buf bytes.Buffer
	if err := s.dynamic.Search(r.Context(), &buf, query); err != nil {
		writeErr(w, err)
		return
	}
	if _, err := io.Copy(w, &buf); err != nil {
		slog.Error("Writing response", "err", err)
	}
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := r.PathValue("id")
	var buf bytes.Buffer
	if err := s.dynamic.Read(r.Context(), &buf, id); err != nil {
		writeErr(w, err)
		return
	}
	if _, err := io.Copy(w, &buf); err != nil {
		slog.Error("Writing response", "err", err)
	}
}

func writeErr(w http.ResponseWriter, err error) {
	var se statusError
	if errors.As(err, &se) {
		http.Error(w, se.Err.Error(), se.StatusCode)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type statusError struct {
	StatusCode int
	Err        error
}

func (s statusError) Error() string {
	return fmt.Sprintf("%v (status %d)", s.Err, s.StatusCode)
}
