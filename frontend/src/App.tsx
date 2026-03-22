import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'

type AuthTab = 'register' | 'login'
type HealthStatus = 'checking' | 'online' | 'offline'
type ThemeMode = 'dark' | 'light'

type UserPayload = {
  id: string
  username: string
  email: string
  created_at: string
}

type AuthApiResponse = {
  token: string
  access_token: string
  refresh_token: string
  user: UserPayload
}

type MeResponse = {
  user_id: string
  email: string
}

type EventPayload = {
  id: string
  organizer_id: string
  title: string
  description: string
  event_date: string
  location: string
  visibility: 'public' | 'private'
  participant_limit?: number | null
  category?: string
  tags: string[]
  created_at: string
  updated_at: string
}

type EventListResponse = {
  items: EventPayload[]
  count: number
}

type EventFormState = {
  title: string
  description: string
  eventDate: string
  location: string
  visibility: 'public' | 'private'
  participantLimit: string
  category: string
  tags: string
}

const STORAGE_KEYS = {
  accessToken: 'eventra_access_token',
  refreshToken: 'eventra_refresh_token',
  user: 'eventra_user',
  theme: 'eventra_theme',
} as const

const REQUEST_TIMEOUT_MS = 10000
const GET_RETRY_ATTEMPTS = 1
const HEALTH_RETRY_ATTEMPTS = 2
const RETRY_BACKOFF_MS = 350
const LOGIN_DELAY_BASE_MS = 1500
const LOGIN_DELAY_MAX_MS = 15000

type EndpointItem = {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  path: string
  icon: 'user-add' | 'login' | 'refresh' | 'logout' | 'profile' | 'health' | 'event'
}

const endpointItems: EndpointItem[] = [
  { method: 'POST', path: '/api/v1/auth/register', icon: 'user-add' },
  { method: 'POST', path: '/api/v1/auth/login', icon: 'login' },
  { method: 'POST', path: '/api/v1/auth/refresh', icon: 'refresh' },
  { method: 'POST', path: '/api/v1/auth/logout', icon: 'logout' },
  { method: 'GET', path: '/api/v1/auth/me', icon: 'profile' },
  { method: 'GET', path: '/api/v1/events', icon: 'event' },
  { method: 'POST', path: '/api/v1/events', icon: 'event' },
  { method: 'PUT', path: '/api/v1/events/{id}', icon: 'event' },
  { method: 'DELETE', path: '/api/v1/events/{id}', icon: 'event' },
  { method: 'GET', path: '/health', icon: 'health' },
]

type ApiErrorPayload = {
  error?: string
  message?: string
}

type RequestOptions = {
  retries?: number
}

const defaultEventForm = (): EventFormState => ({
  title: '',
  description: '',
  eventDate: '',
  location: '',
  visibility: 'public',
  participantLimit: '',
  category: '',
  tags: '',
})

const sleep = (ms: number) => new Promise((resolve) => window.setTimeout(resolve, ms))

function getJwtExpiryMs(token: string): number | null {
  const parts = token.split('.')
  if (parts.length < 2) return null

  try {
    const payloadBase64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const payloadJson = atob(payloadBase64)
    const payload = JSON.parse(payloadJson) as { exp?: number }
    if (!payload.exp || typeof payload.exp !== 'number') return null
    return payload.exp * 1000
  } catch {
    return null
  }
}

async function parseApiBody<T>(response: Response): Promise<T | ApiErrorPayload> {
  try {
    return (await response.json()) as T | ApiErrorPayload
  } catch {
    return {}
  }
}

function EndpointIcon({ icon }: { icon: EndpointItem['icon'] }) {
  if (icon === 'user-add') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M15.6 7.8a3.6 3.6 0 1 1-7.2 0 3.6 3.6 0 0 1 7.2 0" />
        <path d="M5.6 18.1a6.4 6.4 0 0 1 8.3-4.9" />
        <path d="M18.4 10.2v5" />
        <path d="M15.9 12.7h5" />
      </svg>
    )
  }

  if (icon === 'login') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M5.5 5.5h5v13h-5" />
        <path d="M11 12h8" />
        <path d="M16 8.5 19.5 12 16 15.5" />
      </svg>
    )
  }

  if (icon === 'refresh') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M19 7v4h-4" />
        <path d="M5 17v-4h4" />
        <path d="M7.7 9.3A6 6 0 0 1 19 11" />
        <path d="M16.3 14.7A6 6 0 0 1 5 13" />
      </svg>
    )
  }

  if (icon === 'logout') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M18.5 5.5h-5v13h5" />
        <path d="M13 12H5" />
        <path d="M8 8.5 4.5 12 8 15.5" />
      </svg>
    )
  }

  if (icon === 'profile') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M8 7.8a2.8 2.8 0 1 0 5.6 0 2.8 2.8 0 0 0-5.6 0" />
        <path d="M5.2 17.8a5.5 5.5 0 0 1 11 0" />
        <path d="M4.5 4.5h15v15h-15z" />
      </svg>
    )
  }

  if (icon === 'event') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M6 4.8v3" />
        <path d="M18 4.8v3" />
        <path d="M4.8 8h14.4" />
        <rect x="4.8" y="6.8" width="14.4" height="12.4" rx="2" />
        <path d="M8.4 12.6h2.8" />
        <path d="M12.8 12.6h2.8" />
        <path d="M8.4 16h2.8" />
      </svg>
    )
  }

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3.8 12h2.8" />
      <path d="M17.4 12h2.8" />
      <path d="M12 3.8v2.8" />
      <path d="M12 17.4v2.8" />
      <path d="M7.6 12.1h2.5l1.6-2.9 2.1 5 1.3-2.1h1.3" />
      <path d="M12 17a5 5 0 1 0-5-5" />
    </svg>
  )
}

function App() {
  const location = useLocation()
  const navigate = useNavigate()
  const apiBase = useMemo(
    () => import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
    [],
  )

  const [health, setHealth] = useState<HealthStatus>('checking')
  const [theme, setTheme] = useState<ThemeMode>('dark')
  const [activeTab, setActiveTab] = useState<AuthTab>('register')
  const [message, setMessage] = useState<string>('Ready to connect with Eventra Auth API.')
  const [loading, setLoading] = useState(false)

  const [registerForm, setRegisterForm] = useState({
    username: '',
    email: '',
    password: '',
  })
  const [loginForm, setLoginForm] = useState({
    email: '',
    password: '',
  })

  const [accessToken, setAccessToken] = useState<string>('')
  const [refreshToken, setRefreshToken] = useState<string>('')
  const [user, setUser] = useState<UserPayload | null>(null)
  const [mePayload, setMePayload] = useState<MeResponse | null>(null)
  const [events, setEvents] = useState<EventPayload[]>([])
  const [eventForm, setEventForm] = useState<EventFormState>(defaultEventForm)
  const [eventsLoading, setEventsLoading] = useState(false)
  const [editingEventId, setEditingEventId] = useState<string | null>(null)
  const [tokenExpiry, setTokenExpiry] = useState<number | null>(null)
  const [loginFailures, setLoginFailures] = useState(0)
  const [loginLockedUntil, setLoginLockedUntil] = useState(0)
  const [nowMs, setNowMs] = useState(Date.now())

  const sessionState = accessToken ? 'Active' : 'Guest'
  const securityState = refreshToken ? 'Rotation enabled' : 'No refresh token'
  const isAuthenticated = Boolean(accessToken && user)
  const isGuestEventsPage = !isAuthenticated && location.pathname === '/events'
  const loginRetrySeconds = Math.max(0, Math.ceil((loginLockedUntil - nowMs) / 1000))
  const isLoginLocked = loginRetrySeconds > 0

  const createdDate = user?.created_at
    ? new Date(user.created_at).toLocaleDateString('tr-TR', {
        day: '2-digit',
        month: 'long',
        year: 'numeric',
      })
    : ''

  useEffect(() => {
    const storedAccess = localStorage.getItem(STORAGE_KEYS.accessToken)
    const storedRefresh = localStorage.getItem(STORAGE_KEYS.refreshToken)
    const storedUser = localStorage.getItem(STORAGE_KEYS.user)
    const storedTheme = localStorage.getItem(STORAGE_KEYS.theme)

    if (storedAccess) setAccessToken(storedAccess)
    if (storedRefresh) setRefreshToken(storedRefresh)
    if (storedUser) setUser(JSON.parse(storedUser) as UserPayload)

    if (storedTheme === 'light' || storedTheme === 'dark') {
      setTheme(storedTheme)
      return
    }

    if (window.matchMedia('(prefers-color-scheme: light)').matches) {
      setTheme('light')
    }
  }, [])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(STORAGE_KEYS.theme, theme)
  }, [theme])

  useEffect(() => {
    const ping = async () => {
      for (let attempt = 0; attempt <= HEALTH_RETRY_ATTEMPTS; attempt++) {
        try {
          const response = await fetch(`${apiBase}/health`)
          if (response.ok) {
            setHealth('online')
            return
          }

          if (response.status < 500 || attempt === HEALTH_RETRY_ATTEMPTS) {
            break
          }
        } catch {
          if (attempt === HEALTH_RETRY_ATTEMPTS) {
            break
          }
        }

        await sleep(RETRY_BACKOFF_MS * (attempt + 1))
      }

      setHealth('offline')
    }

    ping()
  }, [apiBase])

  useEffect(() => {
    if (!isLoginLocked) return

    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 500)

    return () => window.clearInterval(intervalId)
  }, [isLoginLocked])

  const persistSession = (payload: AuthApiResponse) => {
    const nextAccess = payload.access_token || payload.token
    setAccessToken(nextAccess)
    setRefreshToken(payload.refresh_token)
    setUser(payload.user)
    setTokenExpiry(getJwtExpiryMs(nextAccess))

    localStorage.setItem(STORAGE_KEYS.accessToken, nextAccess)
    localStorage.setItem(STORAGE_KEYS.refreshToken, payload.refresh_token)
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(payload.user))
  }

  const clearSession = () => {
    setAccessToken('')
    setRefreshToken('')
    setUser(null)
    setMePayload(null)
    setTokenExpiry(null)
    localStorage.removeItem(STORAGE_KEYS.accessToken)
    localStorage.removeItem(STORAGE_KEYS.refreshToken)
    localStorage.removeItem(STORAGE_KEYS.user)
  }

  useEffect(() => {
    if (!accessToken) {
      setTokenExpiry(null)
      return
    }

    setTokenExpiry(getJwtExpiryMs(accessToken))
  }, [accessToken])

  useEffect(() => {
    if (!tokenExpiry || !isAuthenticated) return

    const delay = tokenExpiry - Date.now()
    if (delay <= 0) {
      clearSession()
      setMessage('Session expired. Please sign in again.')
      navigate('/')
      return
    }

    const timeoutId = window.setTimeout(() => {
      clearSession()
      setMessage('Session expired. Please sign in again.')
      navigate('/')
    }, delay)

    return () => window.clearTimeout(timeoutId)
  }, [isAuthenticated, navigate, tokenExpiry])

  const callApi = async <T,>(
    path: string,
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    body?: unknown,
    withAuth?: boolean,
    options?: RequestOptions,
  ): Promise<T> => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    if (withAuth && accessToken) {
      headers.Authorization = `Bearer ${accessToken}`
    }

    const retryCount = options?.retries ?? (method === 'GET' ? GET_RETRY_ATTEMPTS : 0)

    for (let attempt = 0; attempt <= retryCount; attempt++) {
      const controller = new AbortController()
      const timeoutId = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)

      try {
        const response = await fetch(`${apiBase}${path}`, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
          signal: controller.signal,
        })

        if (response.status === 204) {
          return {} as T
        }

        const data = await parseApiBody<T>(response)
        if (!response.ok) {
          if (method === 'GET' && response.status >= 500 && attempt < retryCount) {
            await sleep(RETRY_BACKOFF_MS * (attempt + 1))
            continue
          }

          const apiError = data as ApiErrorPayload
          throw new Error(apiError.error ?? apiError.message ?? 'Request failed')
        }

        return data as T
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          if (attempt < retryCount) {
            await sleep(RETRY_BACKOFF_MS * (attempt + 1))
            continue
          }
          throw new Error('Request timed out. Please try again.')
        }

        if (error instanceof TypeError && method === 'GET' && attempt < retryCount) {
          await sleep(RETRY_BACKOFF_MS * (attempt + 1))
          continue
        }

        throw error
      } finally {
        window.clearTimeout(timeoutId)
      }
    }

    throw new Error('Request failed')
  }

  const onRegister = async (event: FormEvent) => {
    event.preventDefault()
    setLoading(true)
    try {
      const payload = {
        username: registerForm.username.trim(),
        email: registerForm.email.trim().toLowerCase(),
        password: registerForm.password,
      }

      const data = await callApi<AuthApiResponse>('/api/v1/auth/register', 'POST', payload)
      persistSession(data)
      setMessage('Account created. You are now signed in.')
      setActiveTab('login')
      navigate('/dashboard')
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const onLogin = async (event: FormEvent) => {
    event.preventDefault()

    if (isLoginLocked) {
      setMessage(`Too many attempts. Please wait ${loginRetrySeconds}s.`)
      return
    }

    setLoading(true)
    try {
      const payload = {
        email: loginForm.email.trim().toLowerCase(),
        password: loginForm.password,
      }

      const data = await callApi<AuthApiResponse>('/api/v1/auth/login', 'POST', payload)
      persistSession(data)
      setLoginFailures(0)
      setLoginLockedUntil(0)
      setMessage('Welcome back. Session established.')
      navigate('/dashboard')
    } catch (error) {
      const errMessage = (error as Error).message
      if (errMessage.toLowerCase().includes('invalid credentials')) {
        const nextFailures = loginFailures + 1
        const cooldownMs = Math.min(
          LOGIN_DELAY_BASE_MS * 2 ** Math.max(0, nextFailures - 1),
          LOGIN_DELAY_MAX_MS,
        )
        setLoginFailures(nextFailures)
        setLoginLockedUntil(Date.now() + cooldownMs)
        setNowMs(Date.now())
        setMessage(`Invalid credentials. Retry in ${Math.ceil(cooldownMs / 1000)}s.`)
      } else {
        setMessage(errMessage)
      }
    } finally {
      setLoading(false)
    }
  }

  const onRefresh = async () => {
    if (!refreshToken) {
      setMessage('No refresh token available.')
      return
    }

    setLoading(true)
    try {
      const data = await callApi<AuthApiResponse>('/api/v1/auth/refresh', 'POST', {
        refresh_token: refreshToken,
      })
      persistSession(data)
      setMessage('Token pair rotated successfully.')
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const onFetchMe = async () => {
    setLoading(true)
    try {
      const data = await callApi<MeResponse>('/api/v1/auth/me', 'GET', undefined, true)
      setMePayload(data)
      setMessage('Protected me endpoint responded successfully.')
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const onLogout = async () => {
    if (!refreshToken) {
      clearSession()
      setMessage('Session already cleared.')
      return
    }

    setLoading(true)
    try {
      await callApi('/api/v1/auth/logout', 'POST', { refresh_token: refreshToken })
      clearSession()
      setMessage('Logged out. Refresh token revoked.')
      navigate('/')
    } catch (error) {
      clearSession()
      setMessage(`Logged out locally. Server response: ${(error as Error).message}`)
      navigate('/')
    } finally {
      setLoading(false)
    }
  }

  const loadEvents = async (withAuth: boolean = isAuthenticated) => {
    setEventsLoading(true)
    try {
      const data = await callApi<EventListResponse>(
        '/api/v1/events?limit=50',
        'GET',
        undefined,
        withAuth,
      )
      setEvents(data.items)
      setMessage(`${data.count} event loaded successfully.`)
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setEventsLoading(false)
    }
  }

  const resetEventComposer = () => {
    setEventForm(defaultEventForm())
    setEditingEventId(null)
  }

  const onSaveEvent = async (event: FormEvent) => {
    event.preventDefault()
    setEventsLoading(true)

    try {
      const payload = {
        title: eventForm.title.trim(),
        description: eventForm.description.trim(),
        event_date: new Date(eventForm.eventDate).toISOString(),
        location: eventForm.location.trim(),
        visibility: eventForm.visibility,
        participant_limit:
          eventForm.participantLimit.trim() === ''
            ? null
            : Number.parseInt(eventForm.participantLimit, 10),
        category: eventForm.category.trim(),
        tags: eventForm.tags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      }

      if (Number.isNaN(payload.participant_limit as number)) {
        throw new Error('Participant limit must be a valid number.')
      }

      if (editingEventId) {
        await callApi<EventPayload>(`/api/v1/events/${editingEventId}`, 'PUT', payload, true)
        setMessage('Event updated successfully.')
      } else {
        await callApi<EventPayload>('/api/v1/events', 'POST', payload, true)
        setMessage('Event created successfully.')
      }

      resetEventComposer()
      await loadEvents()
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setEventsLoading(false)
    }
  }

  const onEditEvent = (selected: EventPayload) => {
    setEditingEventId(selected.id)
    setEventForm({
      title: selected.title,
      description: selected.description,
      eventDate: selected.event_date.slice(0, 16),
      location: selected.location,
      visibility: selected.visibility,
      participantLimit:
        selected.participant_limit === null || selected.participant_limit === undefined
          ? ''
          : String(selected.participant_limit),
      category: selected.category ?? '',
      tags: (selected.tags ?? []).join(', '),
    })
  }

  const onDeleteEvent = async (eventID: string) => {
    if (!window.confirm('Delete this event permanently?')) return

    setEventsLoading(true)
    try {
      await callApi(`/api/v1/events/${eventID}`, 'DELETE', undefined, true)
      if (editingEventId === eventID) {
        resetEventComposer()
      }
      setMessage('Event deleted successfully.')
      await loadEvents()
    } catch (error) {
      setMessage((error as Error).message)
    } finally {
      setEventsLoading(false)
    }
  }

  useEffect(() => {
    if (!isAuthenticated && location.pathname !== '/' && location.pathname !== '/events') {
      navigate('/')
      return
    }

    if (isAuthenticated && location.pathname === '/') {
      navigate('/dashboard')
    }
  }, [isAuthenticated, location.pathname, navigate])

  useEffect(() => {
    if (location.pathname === '/events') {
      void loadEvents(isAuthenticated)
    }
  }, [isAuthenticated, location.pathname])

  const DashboardPage = () => (
    <section className="panel dashboard-panel">
      <div className="panel-head">
        <h2>Welcome, {user?.username}</h2>
        <p>Kisisel kontrol merkezi: oturum, guvenlik ve kimlik dogrulamasi</p>
      </div>

      <div className="dashboard-grid">
        <article className="dash-card">
          <h3>Session Signal</h3>
          <p>{sessionState}</p>
          <span>{user?.email}</span>
        </article>
        <article className="dash-card">
          <h3>Security Layer</h3>
          <p>{securityState}</p>
          <span>Refresh token flow aktif</span>
        </article>
        <article className="dash-card">
          <h3>Account Since</h3>
          <p>{createdDate || 'N/A'}</p>
          <span>ID: {user?.id.slice(0, 12)}...</span>
        </article>
      </div>

      <div className="action-row">
        <button onClick={onRefresh} disabled={loading || !refreshToken}>
          Refresh Session
        </button>
        <button onClick={onFetchMe} disabled={loading || !accessToken}>
          Fetch Me
        </button>
        <button className="danger" onClick={onLogout} disabled={loading}>
          Logout
        </button>
      </div>

      {mePayload && (
        <div className="identity-card">
          <h3>Verified Payload</h3>
          <p className="me-result">
            {mePayload.user_id} · {mePayload.email}
          </p>
        </div>
      )}
    </section>
  )

  const ProfilePage = () => (
    <section className="panel profile-panel">
      <div className="panel-head">
        <h2>Profile</h2>
        <p>Kullaniciya ozel hesap ozet ve API tabanli kimlik bilgileri</p>
      </div>

      <div className="profile-top">
        <div className="avatar-badge">{user?.username.charAt(0).toUpperCase()}</div>
        <div>
          <h3>{user?.username}</h3>
          <p>{user?.email}</p>
          <p>Kayit tarihi: {createdDate || 'N/A'}</p>
        </div>
      </div>

      <div className="profile-grid">
        <article className="profile-card">
          <h3>User ID</h3>
          <code>{user?.id}</code>
        </article>
        <article className="profile-card">
          <h3>Access Token</h3>
          <code>{accessToken ? `${accessToken.slice(0, 56)}...` : 'No token'}</code>
        </article>
        <article className="profile-card">
          <h3>Refresh Token</h3>
          <code>{refreshToken ? `${refreshToken.slice(0, 56)}...` : 'No token'}</code>
        </article>
      </div>
    </section>
  )

  const EventsPage = () => (
    <section className="panel events-panel">
      <div className="panel-head">
        <h2>Events</h2>
        <p>
          {isAuthenticated
            ? 'Create, update, and manage event inventory directly from frontend.'
            : 'Public events are visible. Sign in to create, edit, and delete events.'}
        </p>
      </div>

      {isAuthenticated && (
        <form className="auth-form event-form" onSubmit={onSaveEvent}>
        <div className="event-form-grid">
          <label>
            Title
            <input
              value={eventForm.title}
              onChange={(event) => setEventForm((prev) => ({ ...prev, title: event.target.value }))}
              placeholder="Go Meetup Istanbul"
              required
            />
          </label>
          <label>
            Date
            <input
              type="datetime-local"
              value={eventForm.eventDate}
              onChange={(event) =>
                setEventForm((prev) => ({ ...prev, eventDate: event.target.value }))
              }
              required
            />
          </label>
          <label>
            Location
            <input
              value={eventForm.location}
              onChange={(event) =>
                setEventForm((prev) => ({ ...prev, location: event.target.value }))
              }
              placeholder="Kadikoy"
              required
            />
          </label>
          <label>
            Visibility
            <select
              value={eventForm.visibility}
              onChange={(event) =>
                setEventForm((prev) => ({
                  ...prev,
                  visibility: event.target.value as EventFormState['visibility'],
                }))
              }
            >
              <option value="public">public</option>
              <option value="private">private</option>
            </select>
          </label>
          <label>
            Participant Limit
            <input
              type="number"
              min="0"
              value={eventForm.participantLimit}
              onChange={(event) =>
                setEventForm((prev) => ({ ...prev, participantLimit: event.target.value }))
              }
              placeholder="150"
            />
          </label>
          <label>
            Category
            <input
              value={eventForm.category}
              onChange={(event) =>
                setEventForm((prev) => ({ ...prev, category: event.target.value }))
              }
              placeholder="technology"
            />
          </label>
          <label className="wide">
            Tags (comma separated)
            <input
              value={eventForm.tags}
              onChange={(event) => setEventForm((prev) => ({ ...prev, tags: event.target.value }))}
              placeholder="go, backend, meetup"
            />
          </label>
          <label className="wide">
            Description
            <textarea
              value={eventForm.description}
              onChange={(event) =>
                setEventForm((prev) => ({ ...prev, description: event.target.value }))
              }
              placeholder="Event details"
              rows={3}
            />
          </label>
        </div>

        <div className="action-row">
          <button className="cta" type="submit" disabled={eventsLoading}>
            {eventsLoading ? 'Saving...' : editingEventId ? 'Update event' : 'Create event'}
          </button>
          <button type="button" onClick={resetEventComposer} disabled={eventsLoading}>
            Reset
          </button>
          <button
            type="button"
            onClick={() => {
              void loadEvents(isAuthenticated)
            }}
            disabled={eventsLoading}
          >
            Reload list
          </button>
        </div>
        </form>
      )}

      <div className="events-list">
        {events.length === 0 ? (
          <p className="empty-state">No events yet.</p>
        ) : (
          events.map((item) => (
            <article key={item.id} className="event-item">
              <div>
                <h3>{item.title}</h3>
                <p>
                  {new Date(item.event_date).toLocaleString('tr-TR')} · {item.location} ·{' '}
                  {item.visibility}
                </p>
                <p className="event-meta">
                  Category: {item.category || 'N/A'} · Limit:{' '}
                  {item.participant_limit ?? 'unlimited'} · Tags: {item.tags.join(', ') || '-'}
                </p>
              </div>
              {isAuthenticated && (
                <div className="event-actions">
                  <button type="button" onClick={() => onEditEvent(item)} disabled={eventsLoading}>
                    Edit
                  </button>
                  <button
                    type="button"
                    className="danger"
                    onClick={() => onDeleteEvent(item.id)}
                    disabled={eventsLoading}
                  >
                    Delete
                  </button>
                </div>
              )}
            </article>
          ))
        )}
      </div>
    </section>
  )

  return (
    <div className="app-shell">
      <header className="hero">
        <div className="hero-glow" />
        <div className="hero-grid" />
        <p className="eyebrow">EVENTRA UI CONSOLE</p>
        <h1>Eventra backend, premium auth interface.</h1>
        <p className="subtitle">
          This frontend drives your Go Auth service in real-time: register, login, refresh,
          protected profile fetch, and secure logout.
        </p>
        <div className="status-row">
          <span className={`status-dot status-${health}`} />
          <span>
            Backend status:{' '}
            {health === 'checking'
              ? 'Checking connection...'
              : health === 'online'
                ? 'Online'
                : 'Offline'}
          </span>
          <span className="api-chip">{apiBase}</span>
          <button
            type="button"
            className="theme-toggle"
            onClick={() => setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'))}
            aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
          >
            {theme === 'dark' ? 'Light mode' : 'Dark mode'}
          </button>
        </div>
      </header>

      <section className="kpi-strip">
        <article className="kpi-card">
          <p className="kpi-label">Session mode</p>
          <h3>{sessionState}</h3>
          <span>{user ? `@${user.username}` : 'No authenticated identity yet'}</span>
        </article>
        <article className="kpi-card">
          <p className="kpi-label">Security posture</p>
          <h3>{securityState}</h3>
          <span>Refresh + revoke flow ready</span>
        </article>
        <article className="kpi-card">
          <p className="kpi-label">Auth coverage</p>
          <h3>{endpointItems.length} endpoints</h3>
          <span>Auth + Events endpoints integrated</span>
        </article>
      </section>

      {!isAuthenticated && !isGuestEventsPage ? (
        <main className="main-grid">
          <section className="panel auth-panel">
            <div className="panel-head">
              <h2>Authentication Studio</h2>
              <p>Production-oriented auth flow controller</p>
            </div>

            <div className="tab-row">
              <button
                className={activeTab === 'register' ? 'tab active' : 'tab'}
                onClick={() => setActiveTab('register')}
              >
                Register
              </button>
              <button
                className={activeTab === 'login' ? 'tab active' : 'tab'}
                onClick={() => setActiveTab('login')}
              >
                Login
              </button>
            </div>

            {activeTab === 'register' ? (
              <form className="auth-form" onSubmit={onRegister}>
                <label>
                  Username
                  <input
                    value={registerForm.username}
                    onChange={(event) =>
                      setRegisterForm((prev) => ({ ...prev, username: event.target.value }))
                    }
                    placeholder="nora_dev"
                    required
                  />
                </label>
                <label>
                  Email
                  <input
                    type="email"
                    value={registerForm.email}
                    onChange={(event) =>
                      setRegisterForm((prev) => ({ ...prev, email: event.target.value }))
                    }
                    placeholder="nora@eventra.app"
                    required
                  />
                </label>
                <label>
                  Password
                  <input
                    type="password"
                    value={registerForm.password}
                    onChange={(event) =>
                      setRegisterForm((prev) => ({ ...prev, password: event.target.value }))
                    }
                    placeholder="min 8 chars"
                    required
                  />
                </label>
                <button className="cta" disabled={loading} type="submit">
                  {loading ? 'Processing...' : 'Create account'}
                </button>
              </form>
            ) : (
              <form className="auth-form" onSubmit={onLogin}>
                <label>
                  Email
                  <input
                    type="email"
                    value={loginForm.email}
                    onChange={(event) =>
                      setLoginForm((prev) => ({ ...prev, email: event.target.value }))
                    }
                    placeholder="nora@eventra.app"
                    required
                  />
                </label>
                <label>
                  Password
                  <input
                    type="password"
                    value={loginForm.password}
                    onChange={(event) =>
                      setLoginForm((prev) => ({ ...prev, password: event.target.value }))
                    }
                    placeholder="your password"
                    required
                  />
                </label>
                <button className="cta" disabled={loading || isLoginLocked} type="submit">
                  {loading
                    ? 'Authenticating...'
                    : isLoginLocked
                      ? `Try again in ${loginRetrySeconds}s`
                      : 'Sign in'}
                </button>
                <Link className="cta" to="/events" style={{ textAlign: 'center' }}>
                  Browse public events
                </Link>
              </form>
            )}
          </section>

          <section className="panel identity-card">
            <h3>New: Event Module Connected</h3>
            <p>
              Frontend artik backend event endpointlerine bagli. Giris yaptiktan sonra Events
              sekmesi uzerinden event olusturma, listeleme, guncelleme ve silme islemlerini
              kullanabilirsin.
            </p>
            <p>
              Not: Events sayfasi guvenlik nedeniyle sadece authenticated kullanicilara
              gosterilir.
            </p>
          </section>
        </main>
      ) : isGuestEventsPage ? (
        <main className="main-grid">
          <EventsPage />
        </main>
      ) : (
        <section className="signed-shell">
          <nav className="signed-nav panel">
            <Link
              to="/dashboard"
              className={location.pathname === '/dashboard' ? 'nav-link active' : 'nav-link'}
            >
              Dashboard
            </Link>
            <Link
              to="/profile"
              className={location.pathname === '/profile' ? 'nav-link active' : 'nav-link'}
            >
              Profile
            </Link>
            <Link
              to="/events"
              className={location.pathname === '/events' ? 'nav-link active' : 'nav-link'}
            >
              Events
            </Link>
            <button className="nav-logout" onClick={onLogout} disabled={loading}>
              Logout
            </button>
          </nav>

          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/profile" element={<ProfilePage />} />
            <Route path="/events" element={<EventsPage />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </section>
      )}

      <section className="panel response-panel">
        <h2>Live API Feedback</h2>
        <p>{message}</p>
      </section>

      <section className="panel endpoint-panel">
        <h2>Integrated Endpoints</h2>
        <div className="endpoint-grid">
          {endpointItems.map((endpoint) => (
            <div key={endpoint.path} className="endpoint-card">
              <span className="endpoint-icon">
                <EndpointIcon icon={endpoint.icon} />
              </span>
              <div>
                <span>{endpoint.method}</span>
                <p>{endpoint.path}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <footer className="app-footer" aria-label="Developer credits">
        <span>2026</span>
        <span className="dot" aria-hidden="true">•</span>
        <span className="signature">CodeByPinar</span>
        <span className="dot" aria-hidden="true">•</span>
        <span>Pınar Topuz</span>
      </footer>
    </div>
  )
}

export default App
