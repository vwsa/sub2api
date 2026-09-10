import { apiClient } from '../client'

export interface OpenAIDeviceBinding {
  device_hash: string
  account_id: number
  account_name: string
  user_id: number
  user_email: string
  user_name: string
  api_key_id: number
  api_key_name: string
  first_seen_at: string
  last_seen_at: string
  expires_at: string
  max_device_count: number
}

export interface OpenAIDeviceBindingList {
  items: OpenAIDeviceBinding[]
  total: number
  enabled: boolean
  max_devices_per_account: number
  idle_ttl_days: number
}

export async function list(): Promise<OpenAIDeviceBindingList> {
  const { data } = await apiClient.get<OpenAIDeviceBindingList>('/admin/openai-device-bindings')
  return data
}

export async function remove(deviceHash: string): Promise<void> {
  await apiClient.delete(`/admin/openai-device-bindings/${encodeURIComponent(deviceHash)}`)
}

export default { list, remove }
