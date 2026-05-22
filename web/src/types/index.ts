export interface User {
  id: string
  email: string
  display_name: string
  is_admin: boolean
}

export interface Domain {
  id: string
  name: string
  is_verified: boolean
  mx_verified: boolean
  spf_verified: boolean
  dkim_verified: boolean
}

export interface Mailbox {
  id: string
  address: string
  display_name: string
  quota_bytes: number
  used_bytes: number
}

export interface Folder {
  id: string
  name: string
  special_use: string
  message_count: number
  unseen_count: number
}

export interface Message {
  id: string
  uid: number
  message_id: string
  from_address: string
  from_name: string
  to_addresses: string[]
  cc_addresses: string[]
  subject: string
  date: string
  size_bytes: number
  has_attachments: boolean
  is_seen: boolean
  is_flagged: boolean
  is_answered: boolean
  is_deleted: boolean
  is_draft: boolean
  body_text?: string
  body_html?: string
  attachments?: Attachment[]
}

export interface Attachment {
  filename: string
  size: number
  content_type: string
  download_url: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  display_name: string
}

export interface AuthResponse {
  token: string
  user: User
}
