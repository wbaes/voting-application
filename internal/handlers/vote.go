package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/wouter/voting-with-draw/internal/config"
	dbsqlc "github.com/wouter/voting-with-draw/internal/db/sqlc"
	ws "github.com/wouter/voting-with-draw/internal/websocket"
)

const sessionCookieName = "voter_session"

type VoteHandler struct {
	cfg     *config.Config
	queries *dbsqlc.Queries
	hub     *ws.Hub
}

func NewVoteHandler(cfg *config.Config, queries *dbsqlc.Queries, hub *ws.Hub) *VoteHandler {
	return &VoteHandler{cfg: cfg, queries: queries, hub: hub}
}

// VotePage renders the voting page with photo thumbnails.
func (h *VoteHandler) VotePage(c *gin.Context) {
	sessionID := h.getOrCreateSession(c)

	// Check if this session already voted
	existingVote, err := h.queries.GetVoteBySession(c.Request.Context(), sessionID)
	hasVoted := err == nil && existingVote.ID != 0

	c.HTML(http.StatusOK, "vote.html", gin.H{
		"Exhibition": h.cfg.Exhibition,
		"Photos":     h.cfg.Photos,
		"HasVoted":   hasVoted,
		"VotedFor":   existingVote.PhotoID,
	})
}

// SubmitVote handles the vote submission.
func (h *VoteHandler) SubmitVote(c *gin.Context) {
	sessionID := h.getOrCreateSession(c)

	var req struct {
		PhotoID string `json:"photo_id" binding:"required"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate photo_id exists in config
	if h.cfg.PhotoByID(req.PhotoID) == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo"})
		return
	}

	// Check if already voted
	if _, err := h.queries.GetVoteBySession(c.Request.Context(), sessionID); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already voted"})
		return
	}

	// Cast vote
	_, err := h.queries.CastVote(c.Request.Context(), dbsqlc.CastVoteParams{
		PhotoID:   req.PhotoID,
		SessionID: sessionID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
	})
	if err != nil {
		log.Printf("error casting vote: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote"})
		return
	}

	// Broadcast updated counts
	h.broadcastUpdate(c)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ThankYouPage renders a thank-you page after voting.
func (h *VoteHandler) ThankYouPage(c *gin.Context) {
	c.HTML(http.StatusOK, "thankyou.html", gin.H{
		"Exhibition": h.cfg.Exhibition,
	})
}

func (h *VoteHandler) getOrCreateSession(c *gin.Context) string {
	cookie, err := c.Cookie(sessionCookieName)
	if err == nil && cookie != "" {
		return cookie
	}

	sessionID := uuid.New().String()
	c.SetCookie(sessionCookieName, sessionID, 60*60*24*30, "/", "", false, true)
	return sessionID
}

func (h *VoteHandler) broadcastUpdate(c *gin.Context) {
	counts, err := h.queries.GetVoteCounts(c.Request.Context())
	if err != nil {
		log.Printf("error getting vote counts: %v", err)
		return
	}

	total, err := h.queries.GetTotalVotes(c.Request.Context())
	if err != nil {
		log.Printf("error getting total votes: %v", err)
		return
	}

	photoCounts := make([]ws.PhotoCount, len(counts))
	for i, c := range counts {
		photoCounts[i] = ws.PhotoCount{
			PhotoID:   c.PhotoID,
			VoteCount: c.VoteCount,
		}
	}

	h.hub.Broadcast(ws.VoteUpdate{
		Counts: photoCounts,
		Total:  total,
	})
}
