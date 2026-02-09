package services

import (
	"fmt"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type PoseService struct {
	db                     *gorm.DB
	aiService              *AIService
	taskService            *TaskService
	imageGenerationService *ImageGenerationService
	log                    *logger.Logger
	config                 *config.Config
	promptI18n             *PromptI18n
}

func NewPoseService(db *gorm.DB, aiService *AIService, taskService *TaskService, imageGenerationService *ImageGenerationService, log *logger.Logger, cfg *config.Config) *PoseService {
	return &PoseService{
		db:                     db,
		aiService:              aiService,
		taskService:            taskService,
		imageGenerationService: imageGenerationService,
		log:                    log,
		config:                 cfg,
		promptI18n:             NewPromptI18n(cfg),
	}
}

// ListPoses 获取剧本的姿态列表
func (s *PoseService) ListPoses(dramaID uint) ([]models.Pose, error) {
	var poses []models.Pose
	if err := s.db.Where("drama_id = ?", dramaID).Find(&poses).Error; err != nil {
		return nil, err
	}
	return poses, nil
}

// CreatePose 创建姿态
func (s *PoseService) CreatePose(pose *models.Pose) error {
	return s.db.Create(pose).Error
}

// UpdatePose 更新姿态
func (s *PoseService) UpdatePose(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.Pose{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePose 删除姿态
func (s *PoseService) DeletePose(id uint) error {
	return s.db.Delete(&models.Pose{}, id).Error
}

// ExtractPosesFromScript 从剧本提取姿态（异步）
func (s *PoseService) ExtractPosesFromScript(episodeID uint) (string, error) {
	var episode models.Episode
	if err := s.db.First(&episode, episodeID).Error; err != nil {
		return "", fmt.Errorf("episode not found: %w", err)
	}

	task, err := s.taskService.CreateTask("pose_extraction", fmt.Sprintf("%d", episodeID))
	if err != nil {
		return "", err
	}

	go s.processPoseExtraction(task.ID, episode)

	return task.ID, nil
}

func (s *PoseService) processPoseExtraction(taskID string, episode models.Episode) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在分析剧本...")

	script := ""
	if episode.ScriptContent != nil {
		script = *episode.ScriptContent
	}

	promptTemplate := s.promptI18n.GetPoseExtractionPrompt("", "")

	// Fetch Drama for style overrides
	var drama models.Drama
	if err := s.db.First(&drama, episode.DramaID).Error; err == nil {
		style := ""
		ratio := ""
		if drama.DefaultStyle != nil {
			style = *drama.DefaultStyle
		}
		// Try to use Prop style specific, or just fallback to default style if no specific Pose style
		if drama.DefaultPropStyle != nil {
			if style != "" {
				style += ", " + *drama.DefaultPropStyle
			} else {
				style = *drama.DefaultPropStyle
			}
		}
		if drama.DefaultPropRatio != nil {
			ratio = *drama.DefaultPropRatio
		} else if drama.DefaultImageRatio != nil {
			ratio = *drama.DefaultImageRatio
		}

		promptTemplate = s.promptI18n.GetPoseExtractionPrompt(style, ratio)
	}
	prompt := fmt.Sprintf(promptTemplate, script)

	response, err := s.aiService.GenerateText(prompt, "", ai.WithMaxTokens(2000)) // ai.WithMaxTokens(2000) simplified based on prop_service which might import ai package
	if err != nil {
		s.taskService.UpdateTaskError(taskID, err)
		return
	}

	var extractedPoses []struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		ImagePrompt string `json:"image_prompt"`
	}

	// Assuming utils.SafeParseAIJSON is available like in PropService
	// We need to check imports in PropService to be sure
	// But assuming same package context
	// Actually PropService imports "github.com/drama-generator/backend/pkg/utils"
	// We need to add that import if not present.

	// Since I am replacing the whole file content mostly or chunks, I will need to handle imports carefully.
	// But let's assume imports are handled or I will add them in a separate chunk if needed.
	// Wait, I am replacing the Struct definition too, so I can see imports at top.
	// The current file imports: models, config, logger, gorm.
	// I need to add utils and ai (for parsing perhaps, or just passing int).

	// Let's rely on standard encoding/json if utils is not imported, or just use what PropService used.
	// PropService used: "github.com/drama-generator/backend/pkg/utils"
	// I should probably add imports in a separate step or here if I can see full file?
	// I see lines 1-12.

	// I will just implement logic here assuming imports are fixed later or now.
	// Ah, I should use `utils.SafeParseAIJSON`.

	if err := utils.SafeParseAIJSON(response, &extractedPoses); err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("解析AI结果失败: %w", err))
		return
	}

	s.taskService.UpdateTaskStatus(taskID, "processing", 50, "正在保存姿态...")

	var createdPoses []models.Pose
	for _, p := range extractedPoses {
		pose := models.Pose{
			DramaID:     episode.DramaID,
			Name:        p.Name,
			Type:        &p.Type,
			Description: &p.Description,
			// Pose model does not have Prompt field? Let's check model definition.
			// Prop model has Prompt *string.
			// Pose model has Description *string. Does it have Prompt?
			// Let's check `domain/models/drama.go` again.
			// Pose: Name, Type, Description, ImageURL.
			// No "Prompt" field in Pose struct I saw earlier?
			// Line 188: type Pose struct ...
			// 193: Description *string
			// 194: ImageURL *string
			// It seems Pose doesn't have a separate "Prompt" field like Prop.
			// Prop has Description AND Prompt.
			// If Pose only has Description, maybe we use Description for prompt or add Prompt field?
			// Or maybe Description IS the prompt?
			// In PropService: Description: &p.Description, Prompt: &p.ImagePrompt.
			// If Pose is missing Prompt field, I should probably add it or mapping ImagePrompt to Description if suitable.
			// But Description is usually user facing.
			// Let's assume for now I map ImagePrompt to Description if Description is empty, or append it.
			// Or better: Add Prompt field to Pose model to match Prop.
			// But that requires migration.
			// For now, I will map ImagePrompt to Description (maybe format it: "Desc... \n[Prompt]...")
			// OR just use Description for both.

			// Actually, let's look at `PoseService.processPoseImageGeneration`:
			// req.Prompt = *pose.Description
			// So `Description` is treated as the prompt!
			// So I should put the ImagePrompt into Description.
		}

		if p.ImagePrompt != "" {
			// If we have a separate image prompt, maybe we prefer that for generation?
			// But user sees Description.
			// Let's use ImagePrompt as the Description since `GeneratePoseImage` uses `Description` as prompt.
			// Or combine them.
			// Let's just use ImagePrompt as Description?
			// Or "Description\nPrompt: ImagePrompt"?
			// Let's just use p.ImagePrompt as Description because user wants to generate image.
			// Or p.Description.
			// In PropService, Description is user visible desc, Prompt is AI prompt.
			// Since Pose lacks Prompt, I'll use ImagePrompt as Description.
			pose.Description = &p.ImagePrompt
			// And what about visual description? Maybe ignore p.Description or prepend it.
			if p.Description != "" && p.Description != p.ImagePrompt {
				combined := fmt.Sprintf("%s\n%s", p.Description, p.ImagePrompt)
				pose.Description = &combined
			}
		} else {
			pose.Description = &p.Description
		}

		// 检查是否已存在同名姿态（避免重复）
		var count int64
		s.db.Model(&models.Pose{}).Where("drama_id = ? AND name = ?", episode.DramaID, p.Name).Count(&count)
		if count == 0 {
			if err := s.db.Create(&pose).Error; err == nil {
				createdPoses = append(createdPoses, pose)
			}
		}
	}

	s.taskService.UpdateTaskResult(taskID, createdPoses)
}

// GeneratePoseImage 生成姿态图片
func (s *PoseService) GeneratePoseImage(poseID uint) (string, error) {
	// ... (Same as before)
	// 1. 获取姿态信息
	var pose models.Pose
	if err := s.db.First(&pose, poseID).Error; err != nil {
		return "", err
	}

	if pose.Description == nil || *pose.Description == "" {
		return "", fmt.Errorf("姿态没有描述")
	}

	// 2. 创建任务
	task, err := s.taskService.CreateTask("pose_image_generation", fmt.Sprintf("%d", poseID))
	if err != nil {
		return "", err
	}

	go s.processPoseImageGeneration(task.ID, pose)
	return task.ID, nil
}

func (s *PoseService) processPoseImageGeneration(taskID string, pose models.Pose) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "正在生成图片...")

	imageSize := "1024x1024"
	if s.config != nil && s.config.Style.DefaultImageSize != "" {
		imageSize = s.config.Style.DefaultImageSize
	}

	// Fetch Drama for overrides
	var drama models.Drama
	if err := s.db.First(&drama, pose.DramaID).Error; err == nil {
		if drama.DefaultImageSize != nil {
			imageSize = *drama.DefaultImageSize
		}
	}

	// 创建生成请求
	req := &GenerateImageRequest{
		DramaID:   fmt.Sprintf("%d", pose.DramaID),
		ImageType: "pose", // Assuming "pose" is handled or generic
		Prompt:    *pose.Description + "标准动捕骨架图",
		Size:      imageSize,
		Provider:  s.config.AI.DefaultImageProvider,
	}

	// 调用 ImageGenerationService
	imageGen, err := s.imageGenerationService.GenerateImage(req)
	if err != nil {
		s.taskService.UpdateTaskError(taskID, err)
		return
	}

	// 轮询 ImageGeneration 状态直到完成
	maxAttempts := 60
	pollInterval := 2 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		// 重新加载 imageGen
		var currentImageGen models.ImageGeneration
		if err := s.db.First(&currentImageGen, imageGen.ID).Error; err != nil {
			s.log.Errorw("Failed to poll image generation", "error", err, "id", imageGen.ID)
			continue
		}

		if currentImageGen.Status == models.ImageStatusCompleted {
			if currentImageGen.ImageURL != nil {
				// 任务成功
				s.db.Model(&models.Pose{}).Where("id = ?", pose.ID).Update("image_url", *currentImageGen.ImageURL)
				s.taskService.UpdateTaskResult(taskID, map[string]string{"image_url": *currentImageGen.ImageURL})
				return
			}
		} else if currentImageGen.Status == models.ImageStatusFailed {
			errMsg := "图片生成失败"
			if currentImageGen.ErrorMsg != nil {
				errMsg = *currentImageGen.ErrorMsg
			}
			s.taskService.UpdateTaskError(taskID, fmt.Errorf(errMsg))
			return
		}

		// 更新进度
		s.taskService.UpdateTaskStatus(taskID, "processing", 10+i, "正在生成图片...")
	}

	s.taskService.UpdateTaskError(taskID, fmt.Errorf("生成超时"))
}

// AssociatePosesWithStoryboard 关联姿态到分镜
func (s *PoseService) AssociatePosesWithStoryboard(storyboardID uint, poseIDs []uint) error {
	var storyboard models.Storyboard
	if err := s.db.First(&storyboard, storyboardID).Error; err != nil {
		return err
	}

	var poses []models.Pose
	if len(poseIDs) > 0 {
		if err := s.db.Where("id IN ?", poseIDs).Find(&poses).Error; err != nil {
			return err
		}
	}

	return s.db.Model(&storyboard).Association("Poses").Replace(poses)
}
