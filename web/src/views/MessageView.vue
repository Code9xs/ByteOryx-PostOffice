<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()
const route = useRoute()
const router = useRouter()
const loading = ref(true)

onMounted(async () => {
  const id = route.params.id as string
  await mail.fetchMessage(id)
  if (mail.currentMessage && !mail.currentMessage.is_seen) {
    await mail.markRead(id)
  }
  loading.value = false
})

function goBack() {
  router.back()
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
}
</script>

<template>
  <div class="message-view">
    <div class="view-header">
      <button class="back-btn" @click="goBack">&larr; Back</button>
    </div>
    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="mail.currentMessage" class="message-content">
      <h2>{{ mail.currentMessage.subject || '(no subject)' }}</h2>
      <div class="meta">
        <div class="from">
          <strong>From:</strong> {{ mail.currentMessage.from_name || mail.currentMessage.from_address }}
          <span class="email">&lt;{{ mail.currentMessage.from_address }}&gt;</span>
        </div>
        <div class="to">
          <strong>To:</strong> {{ mail.currentMessage.to_addresses?.join(', ') }}
        </div>
        <div class="date">{{ formatDate(mail.currentMessage.date) }}</div>
      </div>
      <div v-if="mail.currentMessage.attachments?.length" class="attachments">
        <strong>Attachments:</strong>
        <a
          v-for="att in mail.currentMessage.attachments"
          :key="att.filename"
          :href="att.download_url"
          class="attachment"
          target="_blank"
        >
          {{ att.filename }} ({{ (att.size / 1024).toFixed(1) }}KB)
        </a>
      </div>
      <div class="body">
        <div v-if="mail.currentMessage.body_html" v-html="mail.currentMessage.body_html"></div>
        <pre v-else>{{ mail.currentMessage.body_text }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-view {
  padding: 1rem;
}
.view-header {
  margin-bottom: 1rem;
}
.back-btn {
  background: none;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 0.4rem 0.75rem;
  cursor: pointer;
  font-size: 0.85rem;
}
.loading {
  text-align: center;
  padding: 2rem;
  color: #666;
}
.message-content h2 {
  margin: 0 0 1rem;
  font-size: 1.25rem;
}
.meta {
  background: #f8f9fa;
  padding: 0.75rem;
  border-radius: 4px;
  font-size: 0.85rem;
  margin-bottom: 1rem;
}
.meta .email {
  color: #666;
}
.meta .date {
  color: #666;
  margin-top: 0.25rem;
}
.attachments {
  margin-bottom: 1rem;
  font-size: 0.85rem;
}
.attachment {
  display: inline-block;
  margin: 0.25rem 0.5rem;
  padding: 0.25rem 0.5rem;
  background: #e8eaed;
  border-radius: 4px;
  text-decoration: none;
  color: #1a73e8;
}
.body {
  line-height: 1.6;
  font-size: 0.9rem;
}
.body pre {
  white-space: pre-wrap;
  font-family: inherit;
}
</style>
