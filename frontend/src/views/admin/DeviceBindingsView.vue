<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-4 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.deviceBindings.title') }}</h2>
          <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.deviceBindings.description') }}</p>
        </div>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadBindings">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          {{ t('common.refresh') }}
        </button>
      </section>

      <div v-if="!state.enabled && !loading" class="border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
        {{ t('admin.deviceBindings.disabled') }}
      </div>

      <section class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="card border border-gray-200 p-5 dark:border-dark-700">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.deviceBindings.activeDevices') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ state.total }}</p>
        </div>
        <div class="card border border-gray-200 p-5 dark:border-dark-700">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.deviceBindings.upstreamAccounts') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ accountCount }}</p>
        </div>
        <div class="card border border-gray-200 p-5 dark:border-dark-700">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.deviceBindings.policy') }}</p>
          <p class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.deviceBindings.policyValue', { count: state.max_devices_per_account, days: state.idle_ttl_days }) }}
          </p>
        </div>
      </section>

      <section class="card overflow-hidden border border-gray-200 dark:border-dark-700">
        <div class="flex flex-col gap-3 border-b border-gray-200 p-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div class="relative w-full sm:max-w-md">
            <input v-model.trim="search" type="search" class="input w-full pl-9" :placeholder="t('admin.deviceBindings.searchPlaceholder')" />
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.deviceBindings.visibleCount', { visible: filteredBindings.length, total: state.total }) }}
          </p>
        </div>

        <div v-if="loading" class="flex min-h-56 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="filteredBindings.length === 0" class="flex min-h-56 flex-col items-center justify-center px-6 text-center">
          <Icon name="server" size="xl" class="text-gray-400" />
          <p class="mt-3 font-medium text-gray-800 dark:text-gray-200">
            {{ search ? t('admin.deviceBindings.noMatches') : t('admin.deviceBindings.empty') }}
          </p>
          <p class="mt-1 max-w-lg text-sm text-gray-500 dark:text-gray-400">{{ t('admin.deviceBindings.emptyHint') }}</p>
        </div>
        <div v-else class="overflow-x-auto">
          <div class="divide-y divide-gray-100 dark:divide-dark-700 md:hidden">
            <article v-for="binding in filteredBindings" :key="`mobile-${binding.device_hash}`" class="space-y-4 p-5">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <code class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ shortHash(binding.device_hash) }}</code>
                  <p class="mt-2 break-all text-sm font-medium text-gray-900 dark:text-white">
                    {{ binding.user_name || binding.user_email || `#${binding.user_id}` }}
                  </p>
                  <p class="mt-0.5 text-xs text-gray-500">{{ binding.api_key_name || `Key #${binding.api_key_id}` }}</p>
                </div>
                <button type="button" class="btn btn-danger btn-sm flex-shrink-0" :disabled="removingHash === binding.device_hash" @click="removeBinding(binding)">
                  <Icon name="trash" size="sm" />
                  {{ removingHash === binding.device_hash ? t('common.processing') : t('admin.deviceBindings.unbind') }}
                </button>
              </div>
              <dl class="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
                <div>
                  <dt class="text-gray-500">{{ t('admin.deviceBindings.account') }}</dt>
                  <dd class="mt-1 font-medium text-gray-800 dark:text-gray-200">{{ binding.account_name || `#${binding.account_id}` }} · {{ accountUsage(binding.account_id) }} / {{ binding.max_device_count }}</dd>
                </div>
                <div>
                  <dt class="text-gray-500">{{ t('admin.deviceBindings.firstSeen') }}</dt>
                  <dd class="mt-1 text-gray-800 dark:text-gray-200">{{ formatDateTime(binding.first_seen_at) }}</dd>
                </div>
                <div class="col-span-2">
                  <dt class="text-gray-500">{{ t('admin.deviceBindings.activity') }}</dt>
                  <dd class="mt-1 text-gray-800 dark:text-gray-200">{{ formatDateTime(binding.last_seen_at) }} · {{ remaining(binding.expires_at) }}</dd>
                </div>
              </dl>
            </article>
          </div>
          <table class="hidden min-w-full divide-y divide-gray-200 dark:divide-dark-700 md:table">
            <thead class="bg-gray-50 dark:bg-dark-800/70">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">{{ t('admin.deviceBindings.device') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">{{ t('admin.deviceBindings.userAndKey') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">{{ t('admin.deviceBindings.account') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">{{ t('admin.deviceBindings.activity') }}</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="binding in filteredBindings" :key="binding.device_hash" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="px-5 py-4 align-top">
                  <code class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ shortHash(binding.device_hash) }}</code>
                  <p class="mt-2 text-xs text-gray-500">{{ t('admin.deviceBindings.firstSeen') }}: {{ formatDateTime(binding.first_seen_at) }}</p>
                </td>
                <td class="px-5 py-4 align-top text-sm">
                  <p class="font-medium text-gray-900 dark:text-white">{{ binding.user_name || binding.user_email || `#${binding.user_id}` }}</p>
                  <p v-if="binding.user_name && binding.user_email" class="mt-0.5 text-xs text-gray-500">{{ binding.user_email }}</p>
                  <p class="mt-1 text-xs text-gray-500">{{ binding.api_key_name || `Key #${binding.api_key_id}` }}</p>
                </td>
                <td class="px-5 py-4 align-top text-sm">
                  <p class="font-medium text-gray-900 dark:text-white">{{ binding.account_name || `#${binding.account_id}` }}</p>
                  <p class="mt-1 text-xs text-gray-500">{{ accountUsage(binding.account_id) }} / {{ binding.max_device_count }}</p>
                </td>
                <td class="whitespace-nowrap px-5 py-4 align-top text-sm text-gray-600 dark:text-gray-300">
                  <p>{{ formatDateTime(binding.last_seen_at) }}</p>
                  <p class="mt-1 text-xs text-gray-500">{{ remaining(binding.expires_at) }}</p>
                </td>
                <td class="px-5 py-4 text-right align-top">
                  <button type="button" class="btn btn-danger btn-sm" :disabled="removingHash === binding.device_hash" @click="removeBinding(binding)">
                    <Icon name="trash" size="sm" />
                    {{ removingHash === binding.device_hash ? t('common.processing') : t('admin.deviceBindings.unbind') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { OpenAIDeviceBinding, OpenAIDeviceBindingList } from '@/api/admin/deviceBindings'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const removingHash = ref('')
const search = ref('')
const state = ref<OpenAIDeviceBindingList>({
  items: [],
  total: 0,
  enabled: false,
  max_devices_per_account: 5,
  idle_ttl_days: 30
})

const accountCounts = computed(() => {
  const counts = new Map<number, number>()
  for (const binding of state.value.items) counts.set(binding.account_id, (counts.get(binding.account_id) ?? 0) + 1)
  return counts
})
const accountCount = computed(() => accountCounts.value.size)
const accountUsage = (accountID: number) => accountCounts.value.get(accountID) ?? 0

const filteredBindings = computed(() => {
  const query = search.value.toLowerCase()
  if (!query) return state.value.items
  return state.value.items.filter((binding) =>
    [binding.device_hash, binding.account_name, binding.user_name, binding.user_email, binding.api_key_name]
      .some((value) => String(value ?? '').toLowerCase().includes(query))
  )
})

const shortHash = (hash: string) => `${hash.slice(0, 12)}…`
const remaining = (expiresAt: string) => {
  const milliseconds = new Date(expiresAt).getTime() - Date.now()
  if (milliseconds <= 0) return t('admin.deviceBindings.expired')
  const hours = Math.ceil(milliseconds / 3_600_000)
  if (hours < 24) return t('admin.deviceBindings.expiresInHours', { hours })
  return t('admin.deviceBindings.expiresInDays', { days: Math.ceil(hours / 24) })
}

async function loadBindings() {
  loading.value = true
  try {
    state.value = await adminAPI.deviceBindings.list()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.deviceBindings.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function removeBinding(binding: OpenAIDeviceBinding) {
  const owner = binding.user_name || binding.user_email || `#${binding.user_id}`
  if (!window.confirm(t('admin.deviceBindings.unbindConfirm', { device: shortHash(binding.device_hash), owner }))) return
  removingHash.value = binding.device_hash
  try {
    await adminAPI.deviceBindings.remove(binding.device_hash)
    appStore.showSuccess(t('admin.deviceBindings.unbindSuccess'))
    await loadBindings()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.deviceBindings.unbindFailed')))
  } finally {
    removingHash.value = ''
  }
}

onMounted(loadBindings)
</script>
