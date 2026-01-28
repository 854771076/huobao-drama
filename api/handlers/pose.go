package handlers

import (
	"strconv"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PoseHandler struct {
	poseService *services.PoseService
	log         *logger.Logger
}

func NewPoseHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, aiService *services.AIService, imageGenerationService *services.ImageGenerationService) *PoseHandler {
	return &PoseHandler{
		poseService: services.NewPoseService(db, aiService, services.NewTaskService(db, log), imageGenerationService, log, cfg),
		log:         log,
	}
}

// ListPoses 获取姿态列表
func (h *PoseHandler) ListPoses(c *gin.Context) {
	dramaIDStr := c.Param("id") // Ensure consistency with routes: /dramas/:id/poses commonly used

	// If route is /poses?drama_id=X
	if dramaIDStr == "" {
		dramaIDStr = c.Query("drama_id")
	}

	if dramaIDStr == "" {
		response.BadRequest(c, "drama_id is required")
		return
	}

	dramaID, err := strconv.ParseUint(dramaIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid drama_id")
		return
	}

	poses, err := h.poseService.ListPoses(uint(dramaID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, poses)
}

// CreatePose 创建姿态
func (h *PoseHandler) CreatePose(c *gin.Context) {
	var pose models.Pose
	if err := c.ShouldBindJSON(&pose); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.poseService.CreatePose(&pose); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, pose)
}

// UpdatePose 更新姿态
func (h *PoseHandler) UpdatePose(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.poseService.UpdatePose(uint(id), updates); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeletePose 删除姿态
func (h *PoseHandler) DeletePose(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	if err := h.poseService.DeletePose(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GenerateImage 生成姿态图片
func (h *PoseHandler) GenerateImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	taskID, err := h.poseService.GeneratePoseImage(uint(id))
	if err != nil {
		h.log.Errorw("Failed to generate pose image", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID, "message": "图片生成任务已提交"})
}

// AssociatePoses 关联姿态
func (h *PoseHandler) AssociatePoses(c *gin.Context) {
	storyboardIDStr := c.Param("id")
	storyboardID, err := strconv.ParseUint(storyboardIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid storyboard_id")
		return
	}

	var req struct {
		PoseIDs []uint `json:"pose_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.poseService.AssociatePosesWithStoryboard(uint(storyboardID), req.PoseIDs); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ExtractPoses 提取姿态
func (h *PoseHandler) ExtractPoses(c *gin.Context) {
	episodeIDStr := c.Param("episode_id")
	episodeID, err := strconv.ParseUint(episodeIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid episode_id")
		return
	}

	taskID, err := h.poseService.ExtractPosesFromScript(uint(episodeID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID})
}
