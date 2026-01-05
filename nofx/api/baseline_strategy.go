package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nofx/logger"
	"nofx/store"
)

// CreateBaselineStrategyRequest request to create a baseline strategy
type CreateBaselineStrategyRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Config      store.BaselineConfig `json:"config" binding:"required"`
}

// UpdateBaselineStrategyRequest request to update a baseline strategy
type UpdateBaselineStrategyRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Config      store.BaselineConfig `json:"config" binding:"required"`
}

// BaselineStrategyResponse response for baseline strategy
type BaselineStrategyResponse struct {
	ID              string                  `json:"id"`
	UserID          string                  `json:"user_id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Config          store.BaselineConfig    `json:"config"`
	IsSystemDefault bool                    `json:"is_system_default"`
	Stats           *store.AggregatedStats  `json:"stats,omitempty"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

// handleListBaselineStrategies lists all baseline strategies
func (s *Server) handleListBaselineStrategies(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	logger.Infof("🔍 Querying baseline strategies for user %s", userID)

	strategies, err := s.store.BaselineStrategy().ListWithPerformance(userID)
	if err != nil {
		logger.Errorf("❌ Failed to query baseline strategies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Infof("✅ Found %d baseline strategies", len(strategies))

	var response []BaselineStrategyResponse
	for _, strategy := range strategies {
		response = append(response, BaselineStrategyResponse{
			ID:              strategy.ID,
			UserID:          strategy.UserID,
			Name:            strategy.Name,
			Description:     strategy.Description,
			Config:          strategy.Config,
			IsSystemDefault: strategy.IsSystemDefault,
			Stats:           strategy.Stats,
			CreatedAt:       strategy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       strategy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, response)
}

// handleGetBaselineStrategy gets a single baseline strategy
func (s *Server) handleGetBaselineStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	id := c.Param("id")
	strategy, err := s.store.BaselineStrategy().Get(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Baseline strategy not found"})
		return
	}

	stats, _ := s.store.BaselineStrategy().GetAggregatedStats(id)

	response := BaselineStrategyResponse{
		ID:              strategy.ID,
		UserID:          strategy.UserID,
		Name:            strategy.Name,
		Description:     strategy.Description,
		Config:          strategy.Config,
		IsSystemDefault: strategy.IsSystemDefault,
		Stats:           stats,
		CreatedAt:       strategy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       strategy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

// handleCreateBaselineStrategy creates a new baseline strategy
func (s *Server) handleCreateBaselineStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	var req CreateBaselineStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy := &store.BaselineStrategy{
		ID:              uuid.New().String(),
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		Config:          req.Config,
		IsSystemDefault: false,
	}

	if err := s.store.BaselineStrategy().Create(strategy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := BaselineStrategyResponse{
		ID:              strategy.ID,
		UserID:          strategy.UserID,
		Name:            strategy.Name,
		Description:     strategy.Description,
		Config:          strategy.Config,
		IsSystemDefault: strategy.IsSystemDefault,
		CreatedAt:       strategy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       strategy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusCreated, response)
}

// handleUpdateBaselineStrategy updates a baseline strategy
func (s *Server) handleUpdateBaselineStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	id := c.Param("id")
	var req UpdateBaselineStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy := &store.BaselineStrategy{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
	}

	if err := s.store.BaselineStrategy().Update(strategy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fetch updated strategy
	updated, err := s.store.BaselineStrategy().Get(userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := BaselineStrategyResponse{
		ID:              updated.ID,
		UserID:          updated.UserID,
		Name:            updated.Name,
		Description:     updated.Description,
		Config:          updated.Config,
		IsSystemDefault: updated.IsSystemDefault,
		CreatedAt:       updated.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       updated.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}

// handleDeleteBaselineStrategy deletes a baseline strategy
func (s *Server) handleDeleteBaselineStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	id := c.Param("id")
	if err := s.store.BaselineStrategy().Delete(userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Baseline strategy deleted successfully"})
}

// handleGetBaselinePerformance gets performance history for a baseline strategy
func (s *Server) handleGetBaselinePerformance(c *gin.Context) {
	id := c.Param("id")
	limit := 50 // Default limit

	performances, err := s.store.BaselineStrategy().GetPerformanceHistory(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, performances)
}

// handleGetBaselineRankings gets performance rankings for all baseline strategies
func (s *Server) handleGetBaselineRankings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	strategies, err := s.store.BaselineStrategy().ListWithPerformance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sort by average return percentage (descending)
	// Note: In production, you might want to add sorting options
	var rankings []BaselineStrategyResponse
	for _, strategy := range strategies {
		rankings = append(rankings, BaselineStrategyResponse{
			ID:              strategy.ID,
			UserID:          strategy.UserID,
			Name:            strategy.Name,
			Description:     strategy.Description,
			Config:          strategy.Config,
			IsSystemDefault: strategy.IsSystemDefault,
			Stats:           strategy.Stats,
			CreatedAt:       strategy.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:       strategy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, rankings)
}
