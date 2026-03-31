package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"log"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wouter/voting-with-draw/internal/config"
	dbsqlc "github.com/wouter/voting-with-draw/internal/db/sqlc"
	ws "github.com/wouter/voting-with-draw/internal/websocket"
)

type AdminHandler struct {
	cfg     *config.Config
	db      *sql.DB
	queries *dbsqlc.Queries
	hub     *ws.Hub
}

func NewAdminHandler(cfg *config.Config, db *sql.DB, queries *dbsqlc.Queries, hub *ws.Hub) *AdminHandler {
	return &AdminHandler{cfg: cfg, db: db, queries: queries, hub: hub}
}

// AdminAuth is a Gin middleware that requires basic auth for admin routes.
func (h *AdminHandler) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok || user != "admin" || subtle.ConstantTimeCompare([]byte(pass), []byte(h.cfg.Server.AdminPassword)) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Admin"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// AdminPage renders the admin dashboard.
func (h *AdminHandler) AdminPage(c *gin.Context) {
	ctx := c.Request.Context()

	counts, err := h.queries.GetVoteCounts(ctx)
	if err != nil {
		log.Printf("error getting vote counts: %v", err)
	}

	total, err := h.queries.GetTotalVotes(ctx)
	if err != nil {
		log.Printf("error getting total votes: %v", err)
	}

	voters, err := h.queries.GetVotersWithContact(ctx)
	if err != nil {
		log.Printf("error getting voters: %v", err)
	}

	drawResults, err := h.queries.GetDrawResults(ctx)
	if err != nil {
		log.Printf("error getting draw results: %v", err)
	}

	countMap := make(map[string]int64)
	for _, c := range counts {
		countMap[c.PhotoID] = c.VoteCount
	}

	type PhotoResult struct {
		ID        string
		Title     string
		VoteCount int64
	}

	results := make([]PhotoResult, len(h.cfg.Photos))
	for i, p := range h.cfg.Photos {
		results[i] = PhotoResult{
			ID:        p.ID,
			Title:     p.Title,
			VoteCount: countMap[p.ID],
		}
	}

	// Enrich draw results with photo titles
	type DrawResultDisplay struct {
		ID        int64
		Name      string
		Email     string
		Phone     string
		PhotoID   string
		PhotoTitle string
		DrawnAt   string
	}

	drawDisplay := make([]DrawResultDisplay, len(drawResults))
	for i, dr := range drawResults {
		photoTitle := dr.PhotoID
		if p := h.cfg.PhotoByID(dr.PhotoID); p != nil {
			photoTitle = p.Title
		}
		drawDisplay[i] = DrawResultDisplay{
			ID:         dr.ID,
			Name:       dr.Name,
			Email:      dr.Email,
			Phone:      dr.Phone,
			PhotoID:    dr.PhotoID,
			PhotoTitle: photoTitle,
			DrawnAt:    dr.DrawnAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Exhibition":    h.cfg.Exhibition,
		"Results":       results,
		"Total":         total,
		"VotersCount":   len(voters),
		"DrawResults":   drawDisplay,
		"ConnectedClients": h.hub.ClientCount(),
	})
}

// ClearVotes deletes all votes and draw results.
func (h *AdminHandler) ClearVotes(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.queries.ClearAllDrawResults(ctx); err != nil {
		log.Printf("error clearing draw results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear draw results"})
		return
	}

	if err := h.queries.ClearAllVotes(ctx); err != nil {
		log.Printf("error clearing votes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear votes"})
		return
	}

	// Broadcast empty state
	h.hub.Broadcast(ws.VoteUpdate{
		Counts: []ws.PhotoCount{},
		Total:  0,
	})

	c.Redirect(http.StatusSeeOther, "/admin")
}

// RunDraw picks a random voter who left contact details.
func (h *AdminHandler) RunDraw(c *gin.Context) {
	ctx := c.Request.Context()

	voters, err := h.queries.GetVotersWithContact(ctx)
	if err != nil {
		log.Printf("error getting voters for draw: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get voters"})
		return
	}

	if len(voters) == 0 {
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"Exhibition": h.cfg.Exhibition,
			"DrawError":  "No voters with contact details found",
		})
		return
	}

	// Pick a random winner
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(voters))))
	if err != nil {
		log.Printf("error generating random number: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run draw"})
		return
	}
	winner := voters[n.Int64()]

	// Record the draw result
	if _, err := h.queries.RecordDrawResult(ctx, winner.ID); err != nil {
		log.Printf("error recording draw result: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record draw"})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}
