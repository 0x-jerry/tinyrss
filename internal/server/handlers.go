package server

import (
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"tinyrss/internal/feeds"
)

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/feeds", s.handleListFeeds)
	mux.HandleFunc("POST /api/feeds", s.handleCreateFeed)
	mux.HandleFunc("GET /api/feeds/{id}", s.handleGetFeed)
	mux.HandleFunc("PUT /api/feeds/{id}", s.handleUpdateFeed)
	mux.HandleFunc("DELETE /api/feeds/{id}", s.handleDeleteFeed)
	mux.HandleFunc("POST /api/feeds/{id}/render-mode", s.handleSetRenderMode)
	mux.HandleFunc("POST /api/feeds/{id}/refresh", s.handleRefreshFeed)

	mux.HandleFunc("GET /api/folders", s.handleListFolders)
	mux.HandleFunc("POST /api/folders", s.handleCreateFolder)
	mux.HandleFunc("PUT /api/folders/{id}", s.handleUpdateFolder)
	mux.HandleFunc("DELETE /api/folders/{id}", s.handleDeleteFolder)

	mux.HandleFunc("GET /api/items", s.handleListItems)
	mux.HandleFunc("GET /api/items/{id}", s.handleGetItem)
	mux.HandleFunc("POST /api/items/{id}/read", s.handleMarkItem("read", true))
	mux.HandleFunc("POST /api/items/{id}/unread", s.handleMarkItem("unread", false))
	mux.HandleFunc("POST /api/items/{id}/star", s.handleMarkItem("star", true))
	mux.HandleFunc("POST /api/items/{id}/unstar", s.handleMarkItem("unstar", false))
	mux.HandleFunc("POST /api/items/read-all", s.handleReadAll)

	mux.HandleFunc("GET /api/render", s.handleRender)

	mux.HandleFunc("POST /api/opml/import", s.handleOpmlImport)
	mux.HandleFunc("GET /api/opml/export", s.handleOpmlExport)

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/refresh", s.handleRefreshAll)
	mux.HandleFunc("GET /api/refresh/progress", s.handleRefreshProgress)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/fetch-logs", s.handleListFetchLogs)
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)
}

// ---- handlers ----

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	feedsList, err := s.repo.ListFeeds()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feedsList)
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeedURL  string `json:"feed_url"`
		FolderID *int   `json:"folder_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	body.FeedURL = strings.TrimSpace(body.FeedURL)
	if body.FeedURL == "" {
		writeError(w, http.StatusBadRequest, "feed_url is required")
		return
	}
	pf, err := s.fetcher.Probe(body.FeedURL)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity,
			"feed URL is not parseable as RSS/Atom; paste a direct feed URL: "+err.Error())
		return
	}
	siteURL := ""
	if pf.Link != "" {
		siteURL = pf.Link
	}
	feed, err := s.repo.CreateFeed(feeds.Feed{
		Title:       pf.Title,
		FeedURL:     body.FeedURL,
		SiteURL:     siteURL,
		Description: pf.Description,
		FolderID:    body.FolderID,
	})
	if err != nil {
		s.repoError(w, err)
		return
	}
	// First pull populates items so the new feed is immediately useful.
	_, _ = s.fetcher.RefreshFeed(feed.ID)
	feed, _ = s.repo.GetFeed(feed.ID)
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) handleGetFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	feed, err := s.repo.GetFeed(id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Title       string `json:"title"`
		FeedURL     *string `json:"feed_url"`
		SiteURL     *string `json:"site_url"`
		Description *string `json:"description"`
		FolderID    *int    `json:"folder_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	feedURL := body.FeedURL
	if body.FeedURL != nil {
		trimmed := strings.TrimSpace(*body.FeedURL)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "feed_url cannot be empty")
			return
		}
		if feed, err := s.repo.GetFeed(id); err != nil {
			s.repoError(w, err)
			return
		} else if trimmed != feed.FeedURL {
			// Re-validate a changed URL so we never store an unparseable feed.
			if _, err := s.fetcher.Probe(trimmed); err != nil {
				writeError(w, http.StatusUnprocessableEntity,
					"feed URL is not parseable as RSS/Atom; paste a direct feed URL: "+err.Error())
				return
			}
		}
		feedURL = &trimmed
	}
	feed, err := s.repo.UpdateFeed(id, body.Title, feedURL, body.SiteURL, body.Description, body.FolderID)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.repo.DeleteFeed(id); err != nil {
		s.repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetRenderMode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		RenderMode int `json:"render_mode"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.RenderMode != 0 && body.RenderMode != 1 && body.RenderMode != 2 {
		writeError(w, http.StatusBadRequest, "render_mode must be 0, 1 or 2")
		return
	}
	feed, err := s.repo.SetRenderMode(id, body.RenderMode)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) handleRefreshFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	newItems, err := s.fetcher.RefreshFeed(id)
	if err != nil {
		var nf feeds.ErrNotFound
		if errors.As(err, &nf) {
			s.repoError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"new_items": newItems,
		"error":     errorString(err),
	})
}

func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := s.repo.ListFolders()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, folders)
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	folder, err := s.repo.CreateFolder(body.Name)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, folder)
}

func (s *Server) handleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := s.repo.UpdateFolder(id, body.Name); err != nil {
		s.repoError(w, err)
		return
	}
	folder, err := s.repo.GetFolder(id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, folder)
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.repo.DeleteFolder(id); err != nil {
		s.repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := feeds.ItemFilter{
		FeedID:   atoiDefault(q.Get("feed_id"), 0),
		FolderID: atoiDefault(q.Get("folder_id"), 0),
		Unread:   truthy(q.Get("unread")),
		Starred:  truthy(q.Get("starred")),
		Search:   q.Get("search"),
		Page:     atoiDefault(q.Get("page"), 1),
		Limit:    atoiDefault(q.Get("limit"), 50),
	}
	items, total, err := s.repo.ListItems(filter)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := s.repo.GetItem(id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleMarkItem returns a handler that toggles read or star state then 204s.
func (s *Server) handleMarkItem(kind string, value bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var err error
		if kind == "read" || kind == "unread" {
			err = s.repo.SetRead(id, value)
		} else {
			err = s.repo.SetStarred(id, value)
		}
		if err != nil {
			s.repoError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleReadAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := feeds.ItemFilter{
		FeedID:   atoiDefault(q.Get("feed_id"), 0),
		FolderID: atoiDefault(q.Get("folder_id"), 0),
	}
	count, err := s.repo.MarkAllRead(filter)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "url must be http(s)")
		return
	}
	body, err := s.fetcher.FetchRender(raw)
	if err != nil {
		writeRenderError(w, "Could not load page: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms; referrer no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func writeRenderError(w http.ResponseWriter, msg string) {
	body := "<!doctype html><meta charset=utf-8><body style='font-family:system-ui;padding:24px;color:#444'>" + html.EscapeString(msg) + "</body>"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = w.Write([]byte(body))
}

func (s *Server) handleOpmlImport(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "multipart field 'file' is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 5<<20))
	if err != nil {
		s.serverError(w, err)
		return
	}
	added, err := s.repo.ImportOPML(data)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added})
}

func (s *Server) handleOpmlExport(w http.ResponseWriter, r *http.Request) {
	data, err := s.repo.ExportOPML()
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	if err := s.fetcher.RefreshAllAsync(); err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.fetcher.Progress())
}

func (s *Server) handleRefreshProgress(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.fetcher.Progress())
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	feedsCount, items, unread, err := s.repo.Counts()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"feeds":  feedsCount,
		"items":  items,
		"unread": unread,
	})
}

func (s *Server) handleListFetchLogs(w http.ResponseWriter, r *http.Request) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 100)
	logs, err := s.repo.ListFetchLogs(limit)
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.repo.GetSettings()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FetchLogCleanupDays *int `json:"fetch_log_cleanup_days"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.FetchLogCleanupDays == nil || *body.FetchLogCleanupDays < 0 {
		writeError(w, http.StatusBadRequest, "fetch_log_cleanup_days must be a non-negative integer")
		return
	}
	if err := s.repo.SetFetchLogCleanupDays(*body.FetchLogCleanupDays); err != nil {
		s.serverError(w, err)
		return
	}
	settings, err := s.repo.GetSettings()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// ---- helpers ----

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return err
	}
	return nil
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func truthy(s string) bool {
	return s == "1" || strings.EqualFold(s, "true")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) repoError(w http.ResponseWriter, err error) {
	var nf feeds.ErrNotFound
	switch {
	case errors.As(err, &nf):
		writeError(w, http.StatusNotFound, err.Error())
	case isConflict(err):
		writeError(w, http.StatusConflict, err.Error())
	default:
		s.serverError(w, err)
	}
}

func isConflict(err error) bool {
	var c feeds.ErrConflict
	return errors.As(err, &c)
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, "internal error")
}
