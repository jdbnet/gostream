import type { AlbumRow, ArtistRow, GenreRow, Playlist, ScanProgress, Track, TrackQuery } from '../types'

async function readError(res: Response): Promise<string> {
  try {
    const data = await res.json()
    if (data && typeof data.error === 'string') return data.error
  } catch {
    // The body is not JSON.
  }
  return res.statusText || 'Request failed'
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) throw new Error(await readError(res))
  return res.json() as Promise<T>
}

function queryString(filter: TrackQuery): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filter)) {
    if (value === undefined || value === null || value === '') continue
    params.set(key, String(value))
  }
  const text = params.toString()
  return text ? `?${text}` : ''
}

export function fetchTracks(filter: TrackQuery) {
  return api<{ tracks: Track[]; total: number }>(`/api/library/tracks${queryString(filter)}`)
}

export function fetchArtists(q = '', offset = 0) {
  return api<{ artists: ArtistRow[]; total: number }>(`/api/library/artists${queryString({ q, limit: 100, offset })}`)
}

export function fetchAlbums(q = '', offset = 0) {
  return api<{ albums: AlbumRow[]; total: number }>(`/api/library/albums${queryString({ q, limit: 100, offset })}`)
}

export function fetchGenres(q = '', offset = 0) {
  return api<{ genres: GenreRow[]; total: number }>(`/api/library/genres${queryString({ q, limit: 200, offset })}`)
}

export function fetchPlaylists() {
  return api<Playlist[]>('/api/playlists')
}

export function addTracksToPlaylist(playlistId: number, body: { track_ids?: number[]; filter?: TrackQuery }) {
  return api<{ added: number }>(`/api/playlists/${playlistId}/tracks/bulk`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export function createPlaylist(body: Partial<Playlist> & { name: string }) {
  return api<Playlist>('/api/playlists', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export function requestTrack(id: number) {
  return api<{ status: string }>(`/api/stream/request/${id}`, { method: 'POST' })
}

export async function startScan() {
  const res = await fetch('/api/storage/scan', { method: 'POST' })
  const data = await res.json()
  if (!res.ok && res.status !== 409) throw new Error(data.error || 'Scan failed')
  return data as ScanProgress
}

export async function startRefresh() {
  const res = await fetch('/api/storage/refresh', { method: 'POST' })
  const data = await res.json()
  if (!res.ok && res.status !== 409) throw new Error(data.error || 'Refresh failed')
  return data as ScanProgress
}

export function scanStatus() {
  return api<ScanProgress>('/api/storage/scan')
}
