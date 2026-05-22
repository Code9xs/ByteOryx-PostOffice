<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMailStore } from '../stores/mail'

const mail = useMailStore()
const route = useRoute()
const router = useRouter()

async function loadMessages() {
  if (!mail.currentMailbox) return
  const folderName = (route.params.name as string) || 'INBOX'
  await mail.fetchMessages(mail.currentMailbox.id, folderName)
}

onMounted(loadMessages)
watch(() => route.params.name, loadMessages)

function openMessage(msg: any) {
  mail.currentMessage = msg
  router.push({ name: 'message', params: { id: msg.id } })
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="message-list">
    <div class="list-header">
      <h3>{{ (route.params.name as string) || 'INBOX' }}</h3>
      <span class="count">{{ mail.totalMessages }} messages</span>
    </div>
    <div v-if="mail.loading" class="loading">Loading...</div>
    <div v-else-if="mail.messages.length === 0" class="empty">No messages</div>
    <div v-else class="messages">
      <div
        v-for="msg in mail.messages"
        :key="msg.id"
        class="message-row"
        :class="{ unread: !msg.is_seen, flagged: msg.is_flagged }"
        @click="openMessage(msg)"
      >
        <div class="msg-from">{{ msg.from_name || msg.from_address }}</div>
        <div class="msg-subject">
          {{ msg.subject || '(no subject)' }}
          <span v-if="msg.has_attachments" class="attachment-icon">&#128206;</span>
        </div>
        <div class="msg-date">{{ formatDate(msg.date) }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-list {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e0e0e0;
}
.list-header h3 {
  margin: 0;
  font-size: 1rem;
}
.count {
  font-size: 0.75rem;
  color: #666;
}
.loading, .empty {
  padding: 2rem;
  text-align: center;
  color: #666;
}
.messages {
  flex: 1;
  overflow-y: auto;
}
.message-row {
  display: grid;
  grid-template-columns: 180px 1fr 80px;
  align-items: center;
  padding: 0.6rem 1rem;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  font-size: 0.85rem;
}
.message-row:hover {
  background: #f8f9fa;
}
.message-row.unread {
  font-weight: 600;
}
.message-row.flagged {
  border-left: 3px solid #f4b400;
}
.msg-from {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.msg-subject {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #444;
}
.msg-date {
  text-align: right;
  color: #666;
  font-size: 0.75rem;
}
.attachment-icon {
  margin-left: 0.25rem;
}
</style>
