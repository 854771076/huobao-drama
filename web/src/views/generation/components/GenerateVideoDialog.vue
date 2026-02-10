<template>
  <el-dialog
    v-model="visible"
    title="AI 视频生成"
    width="700px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
      <el-form-item label="生成模式" prop="reference_mode">
        <el-radio-group v-model="form.reference_mode">
          <el-radio label="single">单图生视频</el-radio>
          <el-radio label="motion_sequence">动作序列 (九宫格)</el-radio>
          <el-radio label="none">纯文本生成</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="选择剧本" prop="drama_id">
        <el-select v-model="form.drama_id" placeholder="选择剧本" @change="onDramaChange">
          <el-option
            v-for="drama in dramas"
            :key="drama.id"
            :label="drama.title"
            :value="drama.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="选择图片" prop="image_gen_id">
        <el-select
          v-model="form.image_gen_id"
          placeholder="选择已生成的图片"
          clearable
          @change="onImageChange"
        >
          <el-option
            v-for="image in images"
            :key="image.id"
            :label="truncateText(image.prompt, 50)"
            :value="image.id"
          >
            <div class="image-option">
              <img v-if="image.image_url" :src="image.image_url" class="image-thumb" />
              <span>{{ truncateText(image.prompt, 40) }}</span>
            </div>
          </el-option>
        </el-select>
        <div class="form-tip">或直接输入图片 URL</div>
      </el-form-item>

      <el-form-item label="图片 URL" prop="image_url">
        <el-input
          v-model="form.image_url"
          placeholder="https://example.com/image.jpg"
          :disabled="!!form.image_gen_id"
        />
      </el-form-item>

      <el-form-item label="视频提示词" prop="prompt">
        <el-input
          v-model="form.prompt"
          type="textarea"
          :rows="8"
          placeholder="描述视频中的动作和运镜&#10;例如：Camera slowly zooms in, wind blowing through hair, cinematic lighting"
          maxlength="10000"
          show-word-limit
          v-loading="isGeneratingPrompt"
          element-loading-text="正在生成视频脚本..."
        />
      </el-form-item>

      <el-form-item label="AI 服务">
        <el-select v-model="form.provider" placeholder="选择服务">
          <el-option label="豆包视频" value="doubao" />
          <el-option label="Runway" value="runway" />
          <el-option label="Pika" value="pika" />
        </el-select>
      </el-form-item>

      <el-form-item label="视频时长">
        <el-slider
          v-model="form.duration"
          :min="3"
          :max="10"
          :marks="durationMarks"
          show-stops
        />
        <span class="slider-value">{{ form.duration }} 秒</span>
      </el-form-item>

      <el-form-item label="宽高比">
        <el-radio-group v-model="form.aspect_ratio">
          <el-radio label="16:9">16:9 (横屏)</el-radio>
          <el-radio label="9:16">9:16 (竖屏)</el-radio>
          <el-radio label="1:1">1:1 (方形)</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-collapse>
        <el-collapse-item title="高级设置" name="advanced">
          <el-form-item label="运动强度">
            <el-slider
              v-model="form.motion_level"
              :min="0"
              :max="100"
              :marks="motionMarks"
            />
            <span class="slider-value">{{ form.motion_level }}</span>
          </el-form-item>

          <el-form-item label="镜头运动">
            <el-select v-model="form.camera_motion" placeholder="选择镜头运动" clearable>
              <el-option label="静止" value="static" />
              <el-option label="推进 (Zoom In)" value="zoom_in" />
              <el-option label="拉远 (Zoom Out)" value="zoom_out" />
              <el-option label="左移 (Pan Left)" value="pan_left" />
              <el-option label="右移 (Pan Right)" value="pan_right" />
              <el-option label="上移 (Tilt Up)" value="tilt_up" />
              <el-option label="下移 (Tilt Down)" value="tilt_down" />
              <el-option label="环绕 (Orbit)" value="orbit" />
            </el-select>
          </el-form-item>

          <el-form-item label="风格" v-if="form.provider === 'doubao'">
            <el-input v-model="form.style" placeholder="例如：电影级、动画风格" />
          </el-form-item>

          <el-form-item label="随机种子">
            <el-input-number v-model="form.seed" :min="-1" placeholder="留空随机" />
            <span class="form-tip">设置相同种子可复现视频</span>
          </el-form-item>
        </el-collapse-item>
      </el-collapse>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="generating" @click="handleGenerate">
        生成视频
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { videoAPI } from '@/api/video'
import { imageAPI } from '@/api/image'
import { dramaAPI } from '@/api/drama'
import type { Drama } from '@/types/drama'
import type { ImageGeneration } from '@/types/image'
import type { GenerateVideoRequest } from '@/types/video'

interface Props {
  modelValue: boolean
  dramaId?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const formRef = ref<FormInstance>()
const generating = ref(false)
const isGeneratingPrompt = ref(false)
const dramas = ref<Drama[]>([])
const images = ref<ImageGeneration[]>([])

const form = reactive<GenerateVideoRequest & { image_gen_id?: number }>({
  drama_id: props.dramaId || '',
  image_gen_id: undefined,
  image_url: '',
  prompt: '',
  provider: 'doubao',
  duration: 5,
  aspect_ratio: '16:9',
  motion_level: 50,
  camera_motion: undefined,
  style: undefined,
  seed: undefined,
  reference_mode: 'single'
})

const rules: FormRules = {
  drama_id: [
    { required: true, message: '请选择剧本', trigger: 'change' }
  ],
  prompt: [
    { required: true, message: '请输入视频提示词', trigger: 'blur' },
    { min: 5, message: '提示词至少5个字符', trigger: 'blur' }
  ]
}

const durationMarks = {
  3: '3s',
  5: '5s',
  7: '7s',
  10: '10s'
}

const motionMarks = {
  0: '静态',
  50: '适中',
  100: '剧烈'
}

watch(() => props.modelValue, (val) => {
  if (val) {
    loadDramas()
    if (props.dramaId) {
      form.drama_id = props.dramaId
      loadImages(props.dramaId)
      
      // 设置默认比例
      const drama = dramas.value.find(d => d.id.toString() === props.dramaId)
      if (drama && drama.default_video_ratio) {
        form.aspect_ratio = drama.default_video_ratio
      }
    }
  }
})

const loadDramas = async () => {
  try {
    const result = await dramaAPI.list({ page: 1, page_size: 100 })
    dramas.value = result.items
  } catch (error: any) {
    console.error('Failed to load dramas:', error)
  }
}

const loadImages = async (dramaId: string) => {
  try {
    const result = await imageAPI.listImages({
      drama_id: dramaId,
      status: 'completed',
      page: 1,
      page_size: 100
    })
    images.value = result.items
  } catch (error: any) {
    console.error('Failed to load images:', error)
  }
}

const onDramaChange = (dramaId: string) => {
  form.image_gen_id = undefined
  form.image_url = ''
  images.value = []
  if (dramaId) {
    loadImages(dramaId)
    const drama = dramas.value.find(d => d.id.toString() === dramaId)
    if (drama && drama.default_video_ratio) {
      form.aspect_ratio = drama.default_video_ratio
    }
  }
}

const onImageChange = (imageGenId: number | undefined) => {
  if (!imageGenId) {
    form.image_url = ''
    return
  }
  
  const image = images.value.find(img => img.id === imageGenId)
  if (image && image.image_url) {
    form.image_url = image.image_url
    
    // 如果是动作序列图，自动切换模式
    if (image.frame_type === 'motion_sequence') {
      form.reference_mode = 'motion_sequence'
      // 自动生成详细脚本
      generatePrompt()
    } else {
      form.prompt = image.prompt
      form.reference_mode = 'single'
    }
  }
}

const pollTimer = ref<number | null>(null)

// 清理定时器
const clearPolling = () => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
}

const generatePrompt = async () => {
  if (!form.image_gen_id) return
  
  clearPolling()
  isGeneratingPrompt.value = true
  
  try {
    // 触发生成
    await videoAPI.generateActionSequencePrompt(form.image_gen_id)
    
    // 开始轮询
    pollTimer.value = window.setInterval(async () => {
      if (!form.image_gen_id) {
        clearPolling()
        return
      }
      
      try {
        // 使用 getImage 还是 listImages? listImages 可能比较重，但 getImage 需要 id
        // imageAPI 应该有 getImage 方法，如果没有，我需要确认
        // 假设 imageAPI.getImage(id) 存在
        // 如果没有，我需要去 api/image.ts 确认
        // 看来 imageAPI.getImage 可能不存在，我看不到 api/image.ts 的内容
        // 保险起见，用 listImages 过滤 id，或者去确认一下 api/image.ts
        // 为了稳妥，先用 listImages 过滤，或者假设 getImage 存在并在出错时修复
        // 但为了避免来回改，我先去确认 api/image.ts
        // 实际上之前的代码用了 listImages，这里也用 listImages 吧，虽然低效一点
        
        const result = await imageAPI.listImages({
          storyboard_id: undefined, // 无法只通过 id 获取，除非后端支持
          // 这里的 filter 可能不够，listImages 通常是按 drama/storyboard 过滤
          // 这里我们有 drama_id，可以传
          drama_id: form.drama_id,
          page: 1,
          page_size: 100 // 假设最近生成的在前
        })
        
        const targetImage = result.items.find((img: ImageGeneration) => img.id === form.image_gen_id)
        
        if (targetImage) {
          if (targetImage.video_prompt_status === 'completed' && targetImage.video_prompt) {
            form.prompt = targetImage.video_prompt
            isGeneratingPrompt.value = false
            ElMessage.success('动作序列视频脚本已生成')
            clearPolling()
          } else if (targetImage.video_prompt_status === 'failed') {
            isGeneratingPrompt.value = false
            ElMessage.error('视频脚本生成失败')
            form.prompt = targetImage.prompt // 回退
            clearPolling()
          }
          // processing or pending: continue polling
        }
      } catch (err) {
        console.error('Polling error:', err)
        // 轮询出错不一定要停止，可能是网络抖动
      }
    }, 2000) // 每2秒轮询一次
    
  } catch (error) {
    console.error('Failed to start generate prompt:', error)
    ElMessage.error('无法开始生成视频脚本')
    isGeneratingPrompt.value = false
  }
}

// 监听对话框关闭，清理定时器
watch(visible, (val) => {
  if (!val) {
    clearPolling()
    isGeneratingPrompt.value = false
  }
})

// 监听模式变化，如果切换到动作序列模式且有图片，尝试生成脚本
watch(() => form.reference_mode, (newVal) => {
  if (newVal === 'motion_sequence' && form.image_gen_id) {
    // 只有当当前prompt看起来不像是一个完整的脚本（简短）时才自动生成？
    // 或者总是生成？用户体验上，切换模式通常意味着用户想用这个模式的功能。
    // 为了避免覆盖用户已编辑的内容，可以加个判断。但在当前流程中，切换模式通常是用户主动操作。
    // 简单起见，如果是切换到该模式，且有图片，就生成。
    generatePrompt()
  }
})

const truncateText = (text: string, length: number) => {
  if (text.length <= length) return text
  return text.substring(0, length) + '...'
}

const handleGenerate = async () => {
  console.log('handleGenerate called')
  
  if (!formRef.value) {
    console.error('formRef is null')
    ElMessage.error('表单初始化失败，请刷新页面重试')
    return
  }

  try {
    const valid = await formRef.value.validate()
    console.log('Form validation result:', valid)
    
    if (!valid) {
      console.log('Form validation failed')
      return
    }

    generating.value = true
    console.log('Starting video generation...', form)
    
    try {
      const params: GenerateVideoRequest = {
        drama_id: form.drama_id,
        prompt: form.prompt,
        provider: form.provider,
        image_gen_id: form.image_gen_id,
        reference_mode: form.reference_mode
      }

      if (form.image_url && form.image_url.trim()) {
        params.image_url = form.image_url
      }

      if (form.duration) params.duration = form.duration
      if (form.aspect_ratio) params.aspect_ratio = form.aspect_ratio
      if (form.motion_level !== undefined) params.motion_level = form.motion_level
      if (form.camera_motion) params.camera_motion = form.camera_motion
      if (form.style) params.style = form.style
      if (form.seed && form.seed > 0) params.seed = form.seed

      console.log('Generating video with params:', params)
      await videoAPI.generateVideo(params)
      
      ElMessage.success('视频生成任务已提交，请稍后查看结果')
      emit('success')
      handleClose()
    } catch (error: any) {
      console.error('Video generation failed:', error)
      ElMessage.error(error.response?.data?.message || error.message || '生成失败')
    } finally {
      generating.value = false
    }
  } catch (error: any) {
    console.error('Form validation error:', error)
    ElMessage.warning('请检查表单填写是否完整')
  }
}

const handleClose = () => {
  visible.value = false
  formRef.value?.resetFields()
}
</script>

<style scoped>
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #999;
}

.slider-value {
  margin-left: 12px;
  font-size: 14px;
  font-weight: 500;
  color: #409eff;
}

.image-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-thumb {
  width: 40px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
}
</style>
