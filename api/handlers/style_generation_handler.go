package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

type StyleGenerationHandler struct {
	aiService *services.AIService
	cfg       *config.Config
	log       *logger.Logger
}

func NewStyleGenerationHandler(aiService *services.AIService, cfg *config.Config, log *logger.Logger) *StyleGenerationHandler {
	return &StyleGenerationHandler{
		aiService: aiService,
		cfg:       cfg,
		log:       log,
	}
}

type GenerateStyleRequest struct {
	Description string `json:"description" binding:"required"`
}

func (h *StyleGenerationHandler) GenerateStyle(c *gin.Context) {
	var req GenerateStyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	promptI18n := services.NewPromptI18n(h.cfg)
	prompt := promptI18n.GetStyleGenerationPrompt(req.Description)

	systemPrompt := "You are a helpful AI assistant that generates JSON configuration for visual styles."

	generatedText, err := h.aiService.GenerateText(prompt, systemPrompt, func(r *ai.ChatCompletionRequest) {
		r.Temperature = 0.7
		r.ResponseFormat = &ai.ResponseFormat{Type: "json_object"}
	})

	if err != nil {
		h.log.Errorw("Failed to generate style", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate style"})
		return
	}

	// Because GenerateText might return markdown block ```json ... ``` even with json_object mode sometimes or if the model doesn't support strict json mode perfectly
	// But assuming the underlying service handles it or we parse it.
	// For now, let's assume valid JSON string is returned.

	var styleConfig map[string]interface{}
	if err := json.Unmarshal([]byte(generatedText), &styleConfig); err != nil {
		h.log.Errorw("Failed to parse generated style JSON", "error", err, "text", generatedText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated style"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    styleConfig,
	})
}
