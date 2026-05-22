<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

const router = useRouter()
const to = ref('')
const cc = ref('')
const subject = ref('')
const body = ref('')
const sending = ref(false)
const error = ref('')

async function handleSend() {
  if (!to.value.trim()) {
    error.value = 'Recipient is required'
    return
  }
  sending.value = true
  error.value = ''
  try {
    await api.post('/messages/send', {
      to: to.value.split(',').map((s) => s.trim()),
      cc: cc.value ? cc.value.split(',').map((s) => s.trim()) : [],
      subject: subject.value,
      body_text: body.value,
    })
    router.push('/')
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to send'
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="compose-view">
    <div class="compose-header">
      <h3>New Message</h3>
      <button class="send-btn" :disabled="sending" @click="handleSend">
        {{ sending ? 'Sending...' : 'Send' }}
      </button>
    </div>
    <form class="compose-form" @submit.prevent="handleSend">
      <div class="field">
        <label>To</label>
        <input v-model="to" type="text" placeholder="recipient@example.com" />
      </div>
      <div class="field">
        <label>Cc</label>
        <input v-model="cc" type="text" placeholder="cc@example.com" />
      </div>
      <div class="field">
        <label>Subject</label>
        <input v-model="subject" type="text" />
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <textarea v-model="body" class="body-input" placeholder="Write your message..."></textarea>
    </form>
  </div>
</template>

<style scoped>
.compose-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 1rem;
}
.compose-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}
.compose-header h3 {
  margin: 0;
}
.send-btn {
  padding: 0.5rem 1.5rem;
  background: #1a73e8;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.send-btn:disabled {
  opacity: 0.6;
}
.compose-form {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.field {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #f0f0f0;
  padding: 0.4rem 0;
}
.field label {
  width: 60px;
  font-size: 0.85rem;
  color: #666;
}
.field input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 0.85rem;
  padding: 0.25rem;
}
.error {
  color: #d93025;
  font-size: 0.8rem;
  margin: 0.5rem 0;
}
.body-input {
  flex: 1;
  margin-top: 0.75rem;
  border: none;
  outline: none;
  resize: none;
  font-size: 0.9rem;
  line-height: 1.6;
  font-family: inherit;
}
</style>
