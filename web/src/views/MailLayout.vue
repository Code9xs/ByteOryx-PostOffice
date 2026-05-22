<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useMailStore } from '../stores/mail'

const auth = useAuthStore()
const mail = useMailStore()
const router = useRouter()

onMounted(async () => {
  await mail.fetchMailboxes()
  if (mail.currentMailbox) {
    await mail.fetchFolders(mail.currentMailbox.id)
  }
})

function selectFolder(folder: any) {
  mail.currentFolder = folder
  router.push({ name: 'folder', params: { name: folder.name } })
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="mail-layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2>PostOffice</h2>
        <button class="compose-btn" @click="router.push('/compose')">Compose</button>
      </div>
      <nav class="folder-list">
        <a
          v-for="folder in mail.folders"
          :key="folder.id"
          class="folder-item"
          :class="{ active: mail.currentFolder?.id === folder.id }"
          @click="selectFolder(folder)"
        >
          <span class="folder-name">{{ folder.name }}</span>
          <span v-if="folder.unseen_count > 0" class="badge">{{ folder.unseen_count }}</span>
        </a>
      </nav>
      <div class="sidebar-footer">
        <button class="settings-btn" @click="router.push('/settings')">Settings</button>
        <button class="logout-btn" @click="handleLogout">Logout</button>
      </div>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.mail-layout {
  display: flex;
  height: 100vh;
}
.sidebar {
  width: 240px;
  background: #f8f9fa;
  border-right: 1px solid #e0e0e0;
  display: flex;
  flex-direction: column;
}
.sidebar-header {
  padding: 1rem;
  border-bottom: 1px solid #e0e0e0;
}
.sidebar-header h2 {
  margin: 0 0 0.75rem;
  font-size: 1.1rem;
}
.compose-btn {
  width: 100%;
  padding: 0.5rem;
  background: #1a73e8;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
}
.folder-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}
.folder-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  cursor: pointer;
  font-size: 0.875rem;
  color: #333;
  text-decoration: none;
}
.folder-item:hover {
  background: #e8eaed;
}
.folder-item.active {
  background: #d2e3fc;
  font-weight: 500;
}
.badge {
  background: #1a73e8;
  color: #fff;
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: 10px;
}
.sidebar-footer {
  padding: 0.75rem 1rem;
  border-top: 1px solid #e0e0e0;
  display: flex;
  gap: 0.5rem;
}
.sidebar-footer button {
  flex: 1;
  padding: 0.4rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 0.75rem;
}
.logout-btn {
  color: #d93025;
}
.content {
  flex: 1;
  overflow-y: auto;
  background: #fff;
}
</style>
