<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

type Session = { id: string; game_date: string; duration_minutes: number; note: string }
type Statistics = { playingDays: number; totalMinutes: number; averageMinutes: number; longestMinutes: number }
type Tab = 'calendar' | 'stats'

const current = ref(new Date(new Date().getFullYear(), new Date().getMonth(), 1))
const entries = ref<Session[]>([])
const selected = ref<string | null>(null)
const hours = ref(0)
const minutes = ref(0)
const note = ref('')
const error = ref('')
const saved = ref('')
const tab = ref<Tab>('calendar')
const stats = ref<Statistics | null>(null)
const loading = ref(true)
const saving = ref(false)
const initData = window.Telegram?.WebApp?.initData ?? ''

const from = computed(() => formatDate(current.value))
const to = computed(() => formatDate(new Date(current.value.getFullYear(), current.value.getMonth() + 1, 0)))
const title = computed(() => new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric' }).format(current.value))
const monthMinutes = computed(() => entries.value.reduce((sum, item) => sum + item.duration_minutes, 0))
const monthDays = computed(() => entries.value.length)
const selectedEntry = computed(() => selected.value ? entry(selected.value) : undefined)
const selectedLabel = computed(() => selected.value
  ? new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'long', weekday: 'long' }).format(new Date(`${selected.value}T12:00:00`))
  : '')
const days = computed(() => {
  const start = new Date(current.value)
  const offset = (start.getDay() + 6) % 7
  const count = new Date(start.getFullYear(), start.getMonth() + 1, 0).getDate()
  return Array.from({ length: offset + count }, (_, index) => index < offset ? null : new Date(start.getFullYear(), start.getMonth(), index - offset + 1))
})

function formatDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function formatDuration(value: number) {
  if (!value) return '0 мин'
  const hour = Math.floor(value / 60)
  const minute = value % 60
  return hour ? `${hour} ч${minute ? ` ${minute} мин` : ''}` : `${minute} мин`
}

function sessionStyle(value?: number) {
  if (!value) return undefined
  // GitHub-style heatmaps use a small ordinal scale: it is much easier to
  // distinguish on a phone than a barely perceptible continuous ramp.
  const scale = value <= 30
    ? ['#c8ecdf', '#82cfb7', '#102f29']
    : value <= 60
      ? ['#78cfb0', '#32977d', '#ffffff']
      : value <= 120
        ? ['#3cae8d', '#176f5c', '#ffffff']
        : value <= 180
          ? ['#237e69', '#0d5548', '#ffffff']
          : ['#135a4d', '#062f2a', '#ffffff']
  return { '--session-start': scale[0], '--session-end': scale[1], '--session-text': scale[2] }
}

function entry(date: string) {
  return entries.value.find(item => item.game_date === date)
}

async function api(path: string, options: RequestInit = {}) {
  const response = await fetch(`/api/v1${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...(options.headers ?? {}) },
    ...options,
  })
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    throw new Error(body?.error?.message ?? 'Не удалось выполнить запрос')
  }
  return response.status === 204 ? null : response.json()
}

async function authenticate() {
  if (!initData) {
    error.value = 'Откройте трекер из Telegram.'
    return false
  }
  await api('/auth/telegram', { method: 'POST', body: JSON.stringify({ initData }) })
  return true
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    entries.value = await api(`/sessions?from=${from.value}&to=${to.value}`)
    if (tab.value === 'stats') stats.value = await api(`/statistics?from=${from.value}&to=${to.value}`)
  } catch (cause) {
    error.value = (cause as Error).message
  } finally {
    loading.value = false
  }
}

function choose(day: Date) {
  const date = formatDate(day)
  selected.value = date
  const value = entry(date)
  hours.value = Math.floor((value?.duration_minutes ?? 0) / 60)
  minutes.value = (value?.duration_minutes ?? 0) % 60
  note.value = value?.note ?? ''
  saved.value = ''
}

function closeEditor() {
  selected.value = null
  saved.value = ''
}

async function save() {
  if (!selected.value) return
  saving.value = true
  try {
    const existing = entry(selected.value)
    const body = JSON.stringify({ gameDate: selected.value, durationMinutes: hours.value * 60 + minutes.value, note: note.value })
    await api(existing ? `/sessions/${existing.id}` : '/sessions', { method: existing ? 'PATCH' : 'POST', body })
    saved.value = 'Сессия сохранена'
    await load()
  } catch (cause) {
    error.value = (cause as Error).message
  } finally {
    saving.value = false
  }
}

async function remove() {
  const existing = selected.value && entry(selected.value)
  if (!existing || !confirm('Удалить эту сессию?')) return
  try {
    await api(`/sessions/${existing.id}`, { method: 'DELETE' })
    closeEditor()
    await load()
  } catch (cause) {
    error.value = (cause as Error).message
  }
}

function move(delta: number) {
  current.value = new Date(current.value.getFullYear(), current.value.getMonth() + delta, 1)
  closeEditor()
  void load()
}

function setTab(value: Tab) {
  tab.value = value
  if (value === 'stats') void load()
}

function quick(value: number) {
  hours.value = Math.floor(value / 60)
  minutes.value = value % 60
}

onMounted(async () => {
  window.Telegram?.WebApp?.ready()
  window.Telegram?.WebApp?.expand()
  try {
    if (await authenticate()) await load()
  } catch (cause) {
    error.value = (cause as Error).message
    loading.value = false
  }
})
</script>

<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="brand"><span class="brand-mark">P</span><span>pool<span>tracker</span></span></div>
      <button class="avatar" type="button" aria-label="Настройки">•••</button>
    </header>

    <section class="intro">
      <p class="eyebrow">ЛИЧНЫЙ ТРЕКЕР</p>
      <h1>Твоя игра.<br><em>Твой ритм.</em></h1>
      <p>Отмечай сессии и замечай прогресс за месяц.</p>
    </section>

    <p v-if="error" class="notice error">{{ error }}</p>

    <template v-if="tab === 'calendar'">
      <section class="month-card">
        <div class="month-title"><div><p class="eyebrow">ОБЗОР МЕСЯЦА</p><h2>{{ title }}</h2></div><span class="cue-ball">●</span></div>
        <div class="month-metrics"><div><strong>{{ formatDuration(monthMinutes) }}</strong><span>за столом</span></div><div><strong>{{ monthDays }}</strong><span>игровых дней</span></div></div>
      </section>

      <section class="calendar-panel" aria-label="Календарь сессий">
        <div class="calendar-head"><button type="button" aria-label="Предыдущий месяц" @click="move(-1)">←</button><strong>{{ title }}</strong><button type="button" aria-label="Следующий месяц" @click="move(1)">→</button></div>
        <div class="weekdays"><span v-for="name in ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']" :key="name">{{ name }}</span></div>
        <div class="calendar-grid" :class="{ loading }"><template v-for="(day, index) in days" :key="index"><button v-if="day" type="button" :class="['calendar-day', { session: entry(formatDate(day)), chosen: selected === formatDate(day), today: formatDate(day) === formatDate(new Date()) }]" :style="sessionStyle(entry(formatDate(day))?.duration_minutes)" :title="entry(formatDate(day)) ? `${formatDuration(entry(formatDate(day))!.duration_minutes)}${entry(formatDate(day))!.note ? ` — ${entry(formatDate(day))!.note}` : ''}` : 'Добавить сессию'" :aria-label="entry(formatDate(day)) ? `${formatDate(day)}: ${formatDuration(entry(formatDate(day))!.duration_minutes)}. Открыть запись` : `${formatDate(day)}: добавить сессию`" @click="choose(day)"><b>{{ day.getDate() }}</b><small v-if="entry(formatDate(day))">{{ formatDuration(entry(formatDate(day))!.duration_minutes) }}</small><i v-if="entry(formatDate(day))?.note" aria-hidden="true">●</i></button><span v-else class="calendar-empty"></span></template></div>
        <div class="legend"><span><i class="level-1"></i>меньше</span><span class="legend-gradient"></span><span>больше времени</span></div>
      </section>

      <button class="add-session" type="button" @click="choose(new Date())"><span>+</span> Добавить сессию</button>
    </template>

    <section v-else class="stats-screen">
      <p class="eyebrow">ТВОЯ ДИНАМИКА</p><h2>{{ title }}</h2>
      <div v-if="loading" class="loading-card">Считаем статистику…</div>
      <template v-else-if="stats"><article class="feature-stat"><span>Время за месяц</span><strong>{{ formatDuration(stats.totalMinutes) }}</strong><p>{{ stats.playingDays }} игровых дней</p></article><div class="stat-grid"><article><span>Средняя сессия</span><strong>{{ formatDuration(stats.averageMinutes) }}</strong></article><article><span>Самая длинная</span><strong>{{ formatDuration(stats.longestMinutes) }}</strong></article></div><p class="stats-note">Регулярность важнее идеальной серии. Отмечай даже короткие подходы к столу.</p></template>
    </section>

    <nav class="bottom-nav" aria-label="Навигация"><button :class="{ active: tab === 'calendar' }" type="button" @click="setTab('calendar')"><span>▦</span>Календарь</button><button :class="{ active: tab === 'stats' }" type="button" @click="setTab('stats')"><span>◔</span>Статистика</button></nav>

    <div v-if="selected" class="sheet-backdrop" @click.self="closeEditor">
      <section class="session-sheet" aria-modal="true" role="dialog">
        <div class="sheet-handle"></div><div class="sheet-header"><div><p class="eyebrow">НОВАЯ СЕССИЯ</p><h2>{{ selectedLabel }}</h2></div><button type="button" aria-label="Закрыть" @click="closeEditor">×</button></div>
        <div v-if="selectedEntry" class="saved-session"><div><span>Сыграно</span><strong>{{ formatDuration(selectedEntry.duration_minutes) }}</strong></div><p v-if="selectedEntry.note">{{ selectedEntry.note }}</p><p v-else class="muted">Без комментария</p></div>
        <div class="duration"><label>Часы<input v-model.number="hours" min="0" max="24" type="number"></label><span>:</span><label>Минуты<input v-model.number="minutes" min="0" max="59" type="number"></label></div>
        <div class="quick"><button v-for="value in [30, 60, 90, 120, 180]" :key="value" type="button" :class="{ picked: hours * 60 + minutes === value }" @click="quick(value)">{{ formatDuration(value) }}</button></div>
        <label class="note-label">Заметка <textarea v-model="note" maxlength="1000" placeholder="Что сегодня получилось?"></textarea></label>
        <p v-if="saved" class="notice success">{{ saved }}</p><button class="save-button" type="button" :disabled="saving" @click="save">{{ saving ? 'Сохраняем…' : selectedEntry ? 'Сохранить изменения' : 'Сохранить сессию' }}</button><button v-if="selectedEntry" class="delete-button" type="button" @click="remove">Удалить запись</button>
      </section>
    </div>
  </main>
</template>
