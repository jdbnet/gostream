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

const uploadQueue = ref<File[]>([])
const currentFileIndex = ref(0)
const totalFiles = ref(0)

const processQueue = async () => {
  if (uploadQueue.value.length === 0) {
    isUploading.value = false
    if (fileInput.value) fileInput.value.value = ''
    return
  }

  isUploading.value = true
  const file = uploadQueue.value[0]
  
  if (!file.type.includes('audio/')) {
    error.value = `Skipping ${file.name}: Not an audio file`
    uploadQueue.value.shift()
    currentFileIndex.value++
    return processQueue()
  }

  error.value = ''
  
  const formData = new FormData()
  formData.append('file', file)
  
  try {
    await new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable) {
          uploadProgress.value = Math.round((event.loaded / event.total) * 100)
        }
      }
      
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          emit('uploaded')
          resolve()
        } else {
          reject(new Error(xhr.responseText || `Upload failed with status ${xhr.status}`))
        }
      }
      
      xhr.onerror = () => reject(new Error('Network error occurred'))
      
      xhr.open('POST', props.endpoint)
      xhr.send(formData)
    })
    
  } catch (e: any) {
    error.value = `Failed to upload ${file.name}: ${e.message}`
    // We intentionally don't reject here so the next file can continue processing
  }

  uploadQueue.value.shift()
  currentFileIndex.value++
  uploadProgress.value = 0
  
  processQueue()
}

const handleFile = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files?.length) return
  
  const newFiles = Array.from(target.files)
  
  if (!isUploading.value) {
    uploadQueue.value = newFiles
    totalFiles.value = newFiles.length
    currentFileIndex.value = 1
    processQueue()
  } else {
    uploadQueue.value.push(...newFiles)
    totalFiles.value += newFiles.length
  }
}
</script>

<template>
  <div class="border-2 border-dashed border-border rounded-xl p-8 text-center hover:border-accent/50 hover:bg-white/[0.02] transition-colors relative overflow-hidden">
    <input 
      type="file" 
      ref="fileInput"
      accept="audio/*"
      multiple
      class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
      @change="handleFile"
    />
    
    <div class="flex flex-col items-center pointer-events-none relative z-10">
      <div v-if="!isUploading">
        <UploadCloud class="w-12 h-12 text-accent mb-4 mx-auto" />
        <h3 class="text-lg font-medium text-white">{{ label }}</h3>
        <p class="text-textSecondary mt-2 text-sm">Drag and drop multiple files, or click to select</p>
      </div>
      
      <div v-else class="flex flex-col items-center w-full max-w-sm mx-auto">
        <Loader2 class="w-12 h-12 text-accent animate-spin mb-4" />
        <h3 class="text-lg font-medium text-white">
          Uploading {{ currentFileIndex }} of {{ totalFiles }}
        </h3>
        <p class="text-textSecondary text-sm truncate w-full mt-1">
          {{ uploadQueue[0]?.name }}
        </p>
        <div class="w-full h-2 bg-surface rounded-full overflow-hidden mt-4">
          <div 
            class="h-full bg-accent transition-all duration-300"
            :style="{ width: `${uploadProgress}%` }"
          ></div>
        </div>
      </div>
    </div>
    
    <div v-if="error" class="mt-4 text-red-400 text-sm pointer-events-none relative z-10">
      {{ error }}
    </div>
  </div>
</template>
