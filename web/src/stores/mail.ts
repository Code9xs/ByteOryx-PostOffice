import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client'
import type { Folder, Message, Mailbox } from '../types'

export const useMailStore = defineStore('mail', () => {
  const mailboxes = ref<Mailbox[]>([])
  const currentMailbox = ref<Mailbox | null>(null)
  const folders = ref<Folder[]>([])
  const currentFolder = ref<Folder | null>(null)
  const messages = ref<Message[]>([])
  const currentMessage = ref<Message | null>(null)
  const totalMessages = ref(0)
  const loading = ref(false)

  async function fetchMailboxes() {
    const { data } = await api.get('/mailboxes')
    mailboxes.value = data.mailboxes || []
    if (mailboxes.value.length > 0 && !currentMailbox.value) {
      currentMailbox.value = mailboxes.value[0]
    }
  }

  async function fetchFolders(mailboxId: string) {
    const { data } = await api.get(`/mailboxes/${mailboxId}/folders`)
    folders.value = data.folders || []
  }

  async function fetchMessages(mailboxId: string, folderName: string, offset = 0, limit = 50) {
    loading.value = true
    try {
      const { data } = await api.get(`/mailboxes/${mailboxId}/messages`, {
        params: { folder: folderName, offset, limit },
      })
      messages.value = data.messages || []
      totalMessages.value = data.total || 0
    } finally {
      loading.value = false
    }
  }

  async function fetchMessage(messageId: string) {
    const { data } = await api.get(`/messages/${messageId}`)
    currentMessage.value = data
  }

  async function markRead(messageId: string) {
    await api.patch(`/messages/${messageId}/flags`, { is_seen: true })
    const msg = messages.value.find((m) => m.id === messageId)
    if (msg) msg.is_seen = true
  }

  async function deleteMessage(messageId: string) {
    await api.delete(`/messages/${messageId}`)
    messages.value = messages.value.filter((m) => m.id !== messageId)
  }

  return {
    mailboxes, currentMailbox, folders, currentFolder,
    messages, currentMessage, totalMessages, loading,
    fetchMailboxes, fetchFolders, fetchMessages, fetchMessage,
    markRead, deleteMessage,
  }
})
