<script setup lang="ts">
import { ref } from 'vue'
import { UploadCloud, Loader2 } from 'lucide-vue-next'

const props = defineProps<{
  endpoint: string
  label: string
}>()

const emit = defineEmits(['uploaded'])

const fileInput = ref<HTMLInputElement | null>(null)
const isUploading = ref(false)
const uploadProgress = ref(0)
const error = ref('')

const handleFile = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files?.length) return
  
  const file = target.files[0]
  if (!file.type.includes('audio/')) {
    error.value = 'Please select an audio file'
    return
  }
  
  error.value = ''
  isUploading.value = true
  
  const formData = new FormData()
  formData.append('file', file)
  
  try {
    const xhr = new XMLHttpRequest()
    
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) {
        uploadProgress.value = Math.round((event.loaded / event.total) * 100)
      }
    }
    
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        emit('uploaded')
        if (fileInput.value) fileInput.value.value = ''
      } else {
        error.value = 'Upload failed: ' + xhr.responseText
      }
      isUploading.value = false
      uploadProgress.value = 0
    }
    
    xhr.onerror = () => {
      error.value = 'Network error occurred'
      isUploading.value = false
      uploadProgress.value = 0
    }
    
    xhr.open('POST', props.endpoint)
    xhr.send(formData)
    
  } catch (e: any) {
    error.value = e.message
    isUploading.value = false
    uploadProgress.value = 0
  }
}
</script>

<template>
  <div class="border-2 border-dashed border-border rounded-xl p-8 text-center hover:border-accent/50 hover:bg-white/[0.02] transition-colors relative">
    <input 
      type="file" 
      ref="fileInput"
      accept="audio/*"
      class="absolute inset-0 w-full h-full opacity-0 cursor-pointer disabled:cursor-not-allowed"
      @change="handleFile"
      :disabled="isUploading"
    />
    
    <div class="flex flex-col items-center pointer-events-none">
      <div v-if="!isUploading">
        <UploadCloud class="w-12 h-12 text-accent mb-4 mx-auto" />
        <h3 class="text-lg font-medium text-white">{{ label }}</h3>
        <p class="text-textSecondary mt-2 text-sm">Drag and drop or click to select</p>
      </div>
      
      <div v-else class="flex flex-col items-center">
        <Loader2 class="w-12 h-12 text-accent animate-spin mb-4" />
        <h3 class="text-lg font-medium text-white">Uploading & Normalising...</h3>
        <div class="w-64 h-2 bg-surface rounded-full overflow-hidden mt-4">
          <div 
            class="h-full bg-accent transition-all duration-300"
            :style="{ width: `${uploadProgress}%` }"
          ></div>
        </div>
      </div>
    </div>
    
    <div v-if="error" class="mt-4 text-red-400 text-sm pointer-events-none">
      {{ error }}
    </div>
  </div>
</template>
