<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api/client'
import type { Domain } from '../types'

const domains = ref<Domain[]>([])
const newDomain = ref('')
const loading = ref(false)
const error = ref('')

onMounted(fetchDomains)

async function fetchDomains() {
  const { data } = await api.get('/domains')
  domains.value = data.domains || []
}

async function addDomain() {
  if (!newDomain.value.trim()) return
  loading.value = true
  error.value = ''
  try {
    await api.post('/domains', { name: newDomain.value.trim() })
    newDomain.value = ''
    await fetchDomains()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to add domain'
  } finally {
    loading.value = false
  }
}

async function verifyDomain(id: string) {
  await api.post(`/domains/${id}/verify`)
  await fetchDomains()
}
</script>

<template>
  <div class="settings-view">
    <h3>Settings</h3>

    <section class="section">
      <h4>Domains</h4>
      <div class="add-domain">
        <input v-model="newDomain" placeholder="example.com" />
        <button :disabled="loading" @click="addDomain">Add Domain</button>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div v-for="domain in domains" :key="domain.id" class="domain-item">
        <span class="domain-name">{{ domain.name }}</span>
        <span class="status" :class="{ verified: domain.is_verified }">
          {{ domain.is_verified ? 'Verified' : 'Pending' }}
        </span>
        <button v-if="!domain.is_verified" class="verify-btn" @click="verifyDomain(domain.id)">
          Verify
        </button>
      </div>
    </section>

    <section class="section">
      <h4>DNS Records</h4>
      <p class="hint">
        For each domain, add the following DNS records:
      </p>
      <ul class="dns-list">
        <li><strong>MX</strong>: Point to your server hostname (priority 10)</li>
        <li><strong>SPF</strong>: TXT record <code>v=spf1 a mx ~all</code></li>
        <li><strong>DKIM</strong>: TXT record at <code>postoffice._domainkey.yourdomain.com</code></li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.settings-view {
  padding: 1.5rem;
  max-width: 600px;
}
.settings-view h3 {
  margin: 0 0 1.5rem;
}
.section {
  margin-bottom: 2rem;
}
.section h4 {
  margin: 0 0 0.75rem;
  font-size: 0.95rem;
}
.add-domain {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}
.add-domain input {
  flex: 1;
  padding: 0.4rem 0.6rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.85rem;
}
.add-domain button {
  padding: 0.4rem 0.75rem;
  background: #1a73e8;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.85rem;
}
.error {
  color: #d93025;
  font-size: 0.8rem;
}
.domain-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid #f0f0f0;
  font-size: 0.85rem;
}
.domain-name {
  flex: 1;
}
.status {
  color: #f4b400;
  font-size: 0.75rem;
}
.status.verified {
  color: #0f9d58;
}
.verify-btn {
  padding: 0.25rem 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 0.75rem;
}
.hint {
  font-size: 0.85rem;
  color: #666;
}
.dns-list {
  font-size: 0.85rem;
  line-height: 1.8;
}
.dns-list code {
  background: #f0f0f0;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.8rem;
}
</style>
