package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wouter/voting-with-draw/internal/config"
	dbsqlc "github.com/wouter/voting-with-draw/internal/db/sqlc"
	ws "github.com/wouter/voting-with-draw/internal/websocket"
)

type ResultsHandler struct {
	cfg     *config.Config
	queries *dbsqlc.Queries
	hub     *ws.Hub
}

func NewResultsHandler(cfg *config.Config, queries *dbsqlc.Queries, hub *ws.Hub) *ResultsHandler {
	return &ResultsHandler{cfg: cfg, queries: queries, hub: hub}
}

// ResultsPage renders the live results page.
func (h *ResultsHandler) ResultsPage(c *gin.Context) {
	counts, err := h.queries.GetVoteCounts(c.Request.Context())
	if err != nil {
		log.Printf("error getting vote counts: %v", err)
	}

	total, err := h.queries.GetTotalVotes(c.Request.Context())
	if err != nil {
		log.Printf("error getting total votes: %v", err)
	}

	// Build a map of photo_id -> count for the template
	countMap := make(map[string]int64)
	for _, c := range counts {
		countMap[c.PhotoID] = c.VoteCount
	}

	type PhotoResult struct {
		ID        string
		Title     string
		File      string
		VoteCount int64
	}

	results := make([]PhotoResult, len(h.cfg.Photos))
	for i, p := range h.cfg.Photos {
		results[i] = PhotoResult{
			ID:        p.ID,
			Title:     p.Title,
			File:      p.File,
			VoteCount: countMap[p.ID],
		}
	}

	c.HTML(http.StatusOK, "results.html", gin.H{
		"Exhibition": h.cfg.Exhibition,
		"Results":    results,
		"Total":      total,
		"Photos":     h.cfg.Photos,
	})
}

// WebSocket handles the WebSocket connection for live results.
func (h *ResultsHandler) WebSocket(c *gin.Context) {
	h.hub.HandleConnection(c.Writer, c.Request)
}
