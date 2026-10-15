export interface Track {
  id: number
  title: string
  artist: string
  album: string
  album_artist: string
  genre: string
  year: number
  track_number: number
  disc_number: number
  duration_seconds: number
  bitrate: number
  sample_rate: number
  artwork_s3_key: string
  date_added: string
  play_count: number
}

export interface Playlist {
  id: number
  name: string
  is_active: boolean
  kind: string
  rules: string
  playback_mode: string
  artist_separation: number
  track_repeat_hours: number
  track_count?: number
  total_duration?: number
  created_at?: string
}

export interface TrackQuery {
  q?: string
  artist?: string
  album?: string
  genre?: string
  year_min?: number | null
  year_max?: number | null
  duration_min?: number | null
  duration_max?: number | null
  added_from?: string
  added_to?: string
  sort?: string
  dir?: 'asc' | 'desc'
  limit?: number
  offset?: number
}

export interface ArtistRow {
  name: string
  track_count: number
}

export interface AlbumRow {
  album: string
  album_artist: string
  year: number
  track_count: number
  artwork_s3_key: string
}

export interface GenreRow {
  name: string
  track_count: number
}

export interface SmartRules {
  artist_in: string[]
  album_in: string[]
  genre_in: string[]
  year_min: number | null
  year_max: number | null
  duration_min: number | null
  duration_max: number | null
  added_within_days: number | null
}

export interface ScanProgress {
  running: boolean
  phase: string
  seen: number
  total: number
  imported: number
  updated: number
  removed: number
  skipped: number
  errors?: string[]
}

export function emptyRules(): SmartRules {
  return {
    artist_in: [],
    album_in: [],
    genre_in: [],
    year_min: null,
    year_max: null,
    duration_min: null,
    duration_max: null,
    added_within_days: null,
  }
}

export function parseRules(raw: string): SmartRules {
  if (!raw) return emptyRules()
  try {
    return { ...emptyRules(), ...JSON.parse(raw) }
  } catch {
    return emptyRules()
  }
}

export function formatDuration(totalSeconds: number): string {
  const seconds = Math.max(0, Math.floor(totalSeconds || 0))
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remain = seconds % 60
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(remain).padStart(2, '0')}`
  }
  return `${minutes}:${String(remain).padStart(2, '0')}`
}

export function artworkUrl(key: string): string {
  return `/api/artwork?key=${encodeURIComponent(key)}`
}
