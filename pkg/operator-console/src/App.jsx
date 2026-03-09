import { useState, useEffect, useCallback } from 'react'

const API_BASE = '/api/v1'
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:4222'
const API_KEY = import.meta.env.VITE_API_KEY || ''

function apiHeaders(includeJson) {
  const h = {}
  if (API_KEY) h['X-API-Key'] = API_KEY
  if (includeJson) h['Content-Type'] = 'application/json'
  return h
}

function apiFetch(url, opts = {}) {
  const hasBody = opts.body !== undefined && opts.body !== null
  const headers = { ...apiHeaders(hasBody), ...opts.headers }
  return fetch(url, { ...opts, headers })
}

function streamUrl(path) {
  const u = API_BASE + path
  return API_KEY ? `${u}${path.includes('?') ? '&' : '?'}api_key=${encodeURIComponent(API_KEY)}` : u
}

const buttonBase = {
  padding: '8px 16px',
  border: 'none',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: 14,
}

const MODE_HINTS = {
  release_control: 'Передача управления джойстику оператора',
  zero_mode: 'Суставы в нулевую позицию',
  stand_mode: 'Стоячая поза',
  walk_mode: 'Режим ходьбы',
}

const COMMAND_LABELS = {
  release_control: 'Release',
  zero_mode: 'Zero',
  stand_mode: 'Stand',
  walk_mode: 'Walk',
  cmd_vel: 'Drive',
  safe_stop: 'Safe Stop',
}

const SCENARIO_COMMANDS = [
  { value: 'stand_mode', label: 'Stand' },
  { value: 'walk_mode', label: 'Walk' },
  { value: 'cmd_vel', label: 'Drive (cmd_vel)' },
  { value: 'release_control', label: 'Release' },
  { value: 'zero_mode', label: 'Zero' },
  { value: 'safe_stop', label: 'Safe Stop' },
]

const KNOWN_CAPABILITIES = [
  'walk',
  'stand',
  'safe_stop',
  'release_control',
  'cmd_vel',
  'zero_mode',
  'patrol',
  'navigation',
  'speech',
]

const MALL_ASSISTANT_STATUS = {
  idle: 'Idle',
  guiding_visitor: 'Guiding Visitor',
  returning_to_base: 'Returning to Base',
}

const STATUS_LABELS = {
  pending: 'Ожидание',
  running: 'Выполняется',
  completed: 'Завершена',
  failed: 'Ошибка',
  cancelled: 'Отменена',
}

function Toast({ toasts, onDismiss }) {
  if (!toasts || toasts.length === 0) return null
  return (
    <div
      style={{
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        maxWidth: 360,
      }}
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          onClick={() => onDismiss(t.id)}
          style={{
            padding: '12px 16px',
            borderRadius: 8,
            background: t.type === 'safe_stop' ? '#7f1d1d' : t.type === 'robot_online' ? '#065f46' : '#0369a1',
            color: '#e2e8f0',
            fontSize: 14,
            boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
            cursor: 'pointer',
          }}
        >
          {t.message}
        </div>
      ))}
    </div>
  )
}

function TaskRow({ task, scenarios, onCancel, readOnly }) {
  const scenario = scenarios.find((s) => s.id === task.scenario_id)
  const scenarioName = scenario?.name ?? task.scenario_id
  const canCancel = !readOnly && (task.status === 'pending' || task.status === 'running')
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '12px 16px',
        borderBottom: '1px solid #334155',
        background: '#1e293b',
      }}
    >
      <div>
        <div style={{ fontSize: 12, color: '#94a3b8' }}>{task.id.slice(0, 8)}...</div>
        <div style={{ fontSize: 14, color: '#e2e8f0' }}>
          {task.robot_id} — {scenarioName}
        </div>
        <span
          style={{
            fontSize: 11,
            padding: '2px 6px',
            borderRadius: 4,
            background: task.status === 'running' ? '#065f46' : task.status === 'completed' ? '#0369a1' : task.status === 'failed' || task.status === 'cancelled' ? '#7f1d1d' : '#475569',
            color: '#e2e8f0',
          }}
        >
          {STATUS_LABELS[task.status] ?? task.status}
        </span>
      </div>
      {canCancel && (
        <button
          onClick={() => onCancel(task.id)}
          style={{
            ...buttonBase,
            background: '#dc2626',
            color: 'white',
            fontSize: 12,
            padding: '6px 12px',
          }}
        >
          Отмена
        </button>
      )}
    </div>
  )
}

function CreateTaskModal({ robots, scenarios, onClose, onCreate, initialScenarioId }) {
  const [robotId, setRobotId] = useState('')
  const [scenarioId, setScenarioId] = useState(initialScenarioId || '')
  const [payload, setPayload] = useState('{}')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState(null)

  const handleSubmit = async () => {
    if (!robotId || !scenarioId) {
      setError('Выберите робота и сценарий')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const body = { robot_id: robotId, scenario_id: scenarioId, operator_id: 'console' }
      if (payload && payload.trim() !== '{}') {
        try {
          body.payload = JSON.parse(payload)
        } catch (_) {
          setError('Неверный JSON в payload')
          setSubmitting(false)
          return
        }
      }
      const res = await apiFetch(`${API_BASE}/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (res.ok) {
        const task = await res.json()
        onCreate(task)
        onClose()
      } else {
        const text = await res.text()
        setError(text || 'Ошибка создания задачи')
      }
    } catch (e) {
      setError(e.message || 'Ошибка')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: '#1e293b',
          padding: 24,
          borderRadius: 8,
          maxWidth: 400,
          width: '100%',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3 style={{ margin: '0 0 16px 0', color: '#e2e8f0' }}>Создать задачу</h3>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Робот</label>
          <select
            value={robotId}
            onChange={(e) => setRobotId(e.target.value)}
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
            }}
          >
            <option value="">—</option>
            {robots.map((r) => (
              <option key={r.id} value={r.id}>
                {r.id} ({r.vendor}/{r.model})
              </option>
            ))}
          </select>
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Сценарий</label>
          <select
            value={scenarioId}
            onChange={(e) => setScenarioId(e.target.value)}
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
            }}
          >
            <option value="">—</option>
            {scenarios.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} — {s.description}
              </option>
            ))}
          </select>
        </div>
        <div style={{ marginBottom: 16 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>
            Payload (JSON, опционально)
          </label>
          <textarea
            value={payload}
            onChange={(e) => setPayload(e.target.value)}
            placeholder='{"duration_sec": 30}'
            rows={3}
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
              fontFamily: 'monospace',
              fontSize: 12,
            }}
          />
        </div>
        {error && (
          <div style={{ marginBottom: 12, fontSize: 12, color: '#fca5a5' }}>{error}</div>
        )}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{ ...buttonBase, background: '#475569', color: 'white' }}>
            Отмена
          </button>
          <button
            onClick={handleSubmit}
            disabled={submitting}
            style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
          >
            {submitting ? 'Создание...' : 'Создать'}
          </button>
        </div>
      </div>
    </div>
  )
}

function AnalyticsPanel({ summaries }) {
  if (!summaries || summaries.length === 0) return null
  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginBottom: 24,
        background: '#1e293b',
      }}
    >
      <h2 style={{ margin: '0 0 12px 0', color: '#e2e8f0', fontSize: 18 }}>Analytics (24h)</h2>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12 }}>
        {summaries.map((s) => (
          <div
            key={s.robot_id}
            style={{
              padding: 12,
              borderRadius: 6,
              background: '#0f172a',
              border: '1px solid #334155',
            }}
          >
            <div style={{ fontWeight: 600, color: '#e2e8f0', marginBottom: 8 }}>{s.robot_id}</div>
            <div style={{ fontSize: 12, color: '#94a3b8', display: 'flex', flexDirection: 'column', gap: 4 }}>
              <span>Uptime: {Math.round(s.uptime_sec / 60)} min</span>
              <span>Commands: {s.commands_count}</span>
              <span>Errors: {s.errors_count}</span>
              <span>Tasks: {s.tasks_completed} done, {s.tasks_failed} failed</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function ZonesPanel({ zones }) {
  if (!zones || zones.length === 0) return null
  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginTop: 24,
        background: '#1e293b',
      }}
    >
      <h2 style={{ margin: '0 0 12px 0', color: '#e2e8f0', fontSize: 18 }}>Зоны</h2>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        {zones.map((z) => (
          <div
            key={z.zone_id}
            style={{
              padding: '12px 16px',
              borderRadius: 6,
              background: z.occupied ? '#7f1d1d' : '#065f46',
              color: z.occupied ? '#fca5a5' : '#6ee7b7',
              fontSize: 14,
              minWidth: 100,
            }}
          >
            <div style={{ fontWeight: 600 }}>Зона {z.zone_id}</div>
            <div style={{ fontSize: 12, marginTop: 4 }}>
              {z.occupied ? `Занята: ${z.robot_id}` : 'Свободна'}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function WorkflowsPanel({ workflows, workflowRuns, onRefresh, readOnly }) {
  const [running, setRunning] = useState(null)
  const handleRun = async (wfId) => {
    setRunning(wfId)
    try {
      const res = await apiFetch(`${API_BASE}/workflows/${wfId}/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ operator_id: 'console' }),
      })
      if (res.ok) {
        onRefresh?.()
      }
    } finally {
      setRunning(null)
    }
  }
  if (!workflows || workflows.length === 0) return null
  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginTop: 24,
        background: '#1e293b',
      }}
    >
      <h2 style={{ margin: '0 0 12px 0', color: '#e2e8f0', fontSize: 18 }}>Workflows</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {workflows.map((wf) => (
          <div
            key={wf.id}
            style={{
              padding: 12,
              borderRadius: 6,
              background: '#0f172a',
              border: '1px solid #334155',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontWeight: 600, color: '#e2e8f0' }}>{wf.name}</div>
                <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 4 }}>{wf.description}</div>
              </div>
              {!readOnly && (
                <button
                  onClick={() => handleRun(wf.id)}
                  disabled={running === wf.id}
                  style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
                >
                  {running === wf.id ? 'Запуск...' : 'Запустить'}
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
      {workflowRuns && workflowRuns.length > 0 && (
        <div style={{ marginTop: 16 }}>
          <div style={{ fontSize: 12, color: '#64748b', marginBottom: 8 }}>Активные runs</div>
          {workflowRuns.slice(0, 5).map((run) => (
            <div
              key={run.id}
              style={{
                padding: 8,
                fontSize: 12,
                color: '#94a3b8',
                borderBottom: '1px solid #334155',
              }}
            >
              {run.id.slice(0, 8)}... — {run.status}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function Stars({ value, max = 5, onRate }) {
  const stars = []
  for (let i = 1; i <= max; i++) {
    stars.push(
      <span
        key={i}
        onClick={() => onRate?.(i)}
        style={{
          cursor: onRate ? 'pointer' : 'default',
          color: i <= value ? '#fbbf24' : '#475569',
          fontSize: 14,
        }}
      >
        ★
      </span>
    )
  }
  return <span style={{ display: 'inline-flex', gap: 2 }}>{stars}</span>
}

function MarketplacePanel({ onUseScenario, readOnly }) {
  const [categories, setCategories] = useState([])
  const [scenarios, setScenarios] = useState([])
  const [category, setCategory] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState('newest')
  const [loading, setLoading] = useState(true)

  const fetchCategories = () =>
    apiFetch(`${API_BASE}/marketplace/categories`)
      .then((r) => r.json())
      .then(setCategories)
      .catch(() => setCategories([]))

  const fetchScenarios = () => {
    setLoading(true)
    const params = new URLSearchParams()
    if (category) params.set('category', category)
    if (search) params.set('search', search)
    if (sort) params.set('sort', sort)
    apiFetch(`${API_BASE}/marketplace/scenarios?${params}`)
      .then((r) => r.json())
      .then(setScenarios)
      .catch(() => setScenarios([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchCategories()
  }, [])

  useEffect(() => {
    fetchScenarios()
  }, [category, search, sort])

  const handleRate = async (scenarioId, rating) => {
    const res = await apiFetch(`${API_BASE}/marketplace/scenarios/${scenarioId}/rate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rating }),
    })
    if (res.ok) fetchScenarios()
  }

  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginTop: 24,
        background: '#1e293b',
      }}
    >
      <h2 style={{ margin: '0 0 16px 0', color: '#e2e8f0', fontSize: 18 }}>Каталог приложений</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
        <select
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          style={{
            padding: '6px 12px',
            borderRadius: 4,
            border: '1px solid #334155',
            background: '#0f172a',
            color: '#e2e8f0',
            fontSize: 13,
          }}
        >
          <option value="">Все категории</option>
          {categories.map((c) => (
            <option key={c.id} value={c.slug}>
              {c.name}
            </option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Поиск..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{
            padding: '6px 12px',
            borderRadius: 4,
            border: '1px solid #334155',
            background: '#0f172a',
            color: '#e2e8f0',
            fontSize: 13,
            minWidth: 120,
          }}
        />
        <select
          value={sort}
          onChange={(e) => setSort(e.target.value)}
          style={{
            padding: '6px 12px',
            borderRadius: 4,
            border: '1px solid #334155',
            background: '#0f172a',
            color: '#e2e8f0',
            fontSize: 13,
          }}
        >
          <option value="newest">Сначала новые</option>
          <option value="rating">По рейтингу</option>
        </select>
      </div>
      {loading ? (
        <div style={{ padding: 24, textAlign: 'center', color: '#64748b' }}>Загрузка...</div>
      ) : scenarios.length === 0 ? (
        <div style={{ padding: 24, textAlign: 'center', color: '#64748b', fontSize: 14 }}>
          Нет опубликованных сценариев.
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 12 }}>
          {scenarios.map((s) => (
            <div
              key={s.id}
              style={{
                padding: 16,
                borderRadius: 8,
                border: '1px solid #334155',
                background: '#0f172a',
              }}
            >
              <div style={{ fontWeight: 600, color: '#e2e8f0', marginBottom: 4 }}>{s.name}</div>
              <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>{s.description || s.id}</div>
              <div style={{ fontSize: 11, color: '#64748b', marginBottom: 8 }}>
                {s.category_name && <span>{s.category_name}</span>}
                {s.author && <span style={{ marginLeft: 8 }}>· {s.author}</span>}
              </div>
              <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
                <Stars value={Math.round(s.avg_rating || 0)} onRate={(r) => handleRate(s.id, r)} />
                <span style={{ fontSize: 12, color: '#64748b' }}>
                  ({s.rating_count || 0})
                </span>
              </div>
              {!readOnly && (
                <button
                  onClick={() => onUseScenario(s)}
                  style={{ ...buttonBase, background: '#0369a1', color: 'white', width: '100%' }}
                >
                  Использовать
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function ScenarioBuilderPanel({ scenarios, onRefresh, onCreateClick, onEditClick, onDeleteClick, readOnly }) {
  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginTop: 24,
        background: '#1e293b',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <h2 style={{ margin: 0, color: '#e2e8f0', fontSize: 18 }}>Scenario Builder</h2>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={onRefresh} style={{ ...buttonBase, background: '#475569', color: 'white' }}>
            Обновить
          </button>
          {!readOnly && (
            <button onClick={onCreateClick} style={{ ...buttonBase, background: '#0369a1', color: 'white' }}>
              Создать сценарий
            </button>
          )}
        </div>
      </div>
      <div style={{ borderRadius: 6, overflow: 'hidden', border: '1px solid #334155' }}>
        {!scenarios || scenarios.length === 0 ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#64748b', fontSize: 14 }}>
            Нет сценариев. Нажмите «Создать сценарий» для добавления.
          </div>
        ) : (
          scenarios.map((s) => (
            <div
              key={s.id}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '12px 16px',
                borderBottom: '1px solid #334155',
                background: '#1e293b',
              }}
            >
              <div>
                <div style={{ fontWeight: 600, color: '#e2e8f0' }}>{s.name}</div>
                <div style={{ fontSize: 12, color: '#94a3b8' }}>{s.id} — {s.description || '-'}</div>
                <div style={{ fontSize: 11, color: '#64748b', marginTop: 4 }}>
                  Steps: {s.steps?.length || 0} · Capabilities: {s.required_capabilities?.join(', ') || '-'}
                </div>
              </div>
              {!readOnly && (
                <div style={{ display: 'flex', gap: 8 }}>
                  <button
                    onClick={() => onEditClick(s)}
                    style={{ ...buttonBase, background: '#475569', color: 'white', fontSize: 12 }}
                  >
                    Изменить
                  </button>
                  <button
                    onClick={() => onDeleteClick(s)}
                    style={{ ...buttonBase, background: '#dc2626', color: 'white', fontSize: 12 }}
                  >
                    Удалить
                  </button>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}

function StepEditor({ steps, onChange }) {
  const inputStyle = {
    width: '100%',
    padding: 8,
    borderRadius: 4,
    border: '1px solid #334155',
    background: '#0f172a',
    color: '#e2e8f0',
  }
  const addStep = () => {
    onChange([...steps, { command: 'stand_mode', payload: null, duration_sec: 0 }])
  }
  const updateStep = (idx, patch) => {
    const next = [...steps]
    next[idx] = { ...next[idx], ...patch }
    onChange(next)
  }
  const removeStep = (idx) => {
    onChange(steps.filter((_, i) => i !== idx))
  }
  const parsePayload = (step) => {
    if (!step.payload) return { linear_x: 0, linear_y: 0, angular_z: 0 }
    try {
      const p = typeof step.payload === 'string' ? JSON.parse(step.payload) : step.payload
      return { linear_x: p.linear_x ?? 0, linear_y: p.linear_y ?? 0, angular_z: p.angular_z ?? 0 }
    } catch {
      return { linear_x: 0, linear_y: 0, angular_z: 0 }
    }
  }
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <label style={{ display: 'block', fontSize: 12, color: '#94a3b8' }}>Steps</label>
        <button type="button" onClick={addStep} style={{ ...buttonBase, background: '#0369a1', color: 'white', fontSize: 12 }}>
          + Add step
        </button>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {steps.map((step, idx) => (
          <div
            key={idx}
            style={{
              padding: 12,
              borderRadius: 6,
              border: '1px solid #334155',
              background: '#0f172a',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <span style={{ fontSize: 12, color: '#64748b' }}>Step {idx + 1}</span>
              <button
                type="button"
                onClick={() => removeStep(idx)}
                style={{ ...buttonBase, background: '#7f1d1d', color: 'white', fontSize: 11, padding: '4px 8px' }}
              >
                Remove
              </button>
            </div>
            <div style={{ marginBottom: 8 }}>
              <label style={{ display: 'block', fontSize: 11, color: '#64748b', marginBottom: 4 }}>Command</label>
              <select
                value={step.command || 'stand_mode'}
                onChange={(e) => updateStep(idx, { command: e.target.value })}
                style={inputStyle}
              >
                {SCENARIO_COMMANDS.map((c) => (
                  <option key={c.value} value={c.value}>
                    {c.label}
                  </option>
                ))}
              </select>
            </div>
            {step.command === 'cmd_vel' && (
              <div style={{ marginBottom: 8, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                <div>
                  <label style={{ display: 'block', fontSize: 11, color: '#64748b', marginBottom: 4 }}>linear_x</label>
                  <input
                    type="number"
                    step="0.1"
                    value={parsePayload(step).linear_x}
                    onChange={(e) => {
                      const p = parsePayload(step)
                      p.linear_x = parseFloat(e.target.value) || 0
                      updateStep(idx, { payload: p })
                    }}
                    style={{ ...inputStyle, width: 70 }}
                  />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: 11, color: '#64748b', marginBottom: 4 }}>linear_y</label>
                  <input
                    type="number"
                    step="0.1"
                    value={parsePayload(step).linear_y}
                    onChange={(e) => {
                      const p = parsePayload(step)
                      p.linear_y = parseFloat(e.target.value) || 0
                      updateStep(idx, { payload: p })
                    }}
                    style={{ ...inputStyle, width: 70 }}
                  />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: 11, color: '#64748b', marginBottom: 4 }}>angular_z</label>
                  <input
                    type="number"
                    step="0.1"
                    value={parsePayload(step).angular_z}
                    onChange={(e) => {
                      const p = parsePayload(step)
                      p.angular_z = parseFloat(e.target.value) || 0
                      updateStep(idx, { payload: p })
                    }}
                    style={{ ...inputStyle, width: 70 }}
                  />
                </div>
              </div>
            )}
            <div>
              <label style={{ display: 'block', fontSize: 11, color: '#64748b', marginBottom: 4 }}>
                Duration (sec) — 0=instant, -1=from task payload
              </label>
              <input
                type="number"
                value={step.duration_sec ?? 0}
                onChange={(e) => updateStep(idx, { duration_sec: parseInt(e.target.value, 10) || 0 })}
                style={inputStyle}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function CapabilitySelector({ selected, onChange }) {
  const toggle = (cap) => {
    if (selected.includes(cap)) {
      onChange(selected.filter((c) => c !== cap))
    } else {
      onChange([...selected, cap])
    }
  }
  return (
    <div>
      <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>
        Required capabilities (select at least one)
      </label>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        {KNOWN_CAPABILITIES.map((cap) => (
          <label
            key={cap}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              borderRadius: 6,
              background: selected.includes(cap) ? '#0369a1' : '#0f172a',
              border: '1px solid #334155',
              cursor: 'pointer',
              fontSize: 13,
              color: '#e2e8f0',
            }}
          >
            <input
              type="checkbox"
              checked={selected.includes(cap)}
              onChange={() => toggle(cap)}
              style={{ accentColor: '#0369a1' }}
            />
            {cap}
          </label>
        ))}
      </div>
    </div>
  )
}

function ScenarioModal({ scenario, onClose, onSaved }) {
  const isEdit = !!scenario
  const [id, setId] = useState(scenario?.id || '')
  const [name, setName] = useState(scenario?.name || '')
  const [description, setDescription] = useState(scenario?.description || '')
  const [steps, setSteps] = useState(() => {
    const s = scenario?.steps
    if (s && Array.isArray(s) && s.length > 0) {
      return s.map((st) => ({
        command: st.command || 'stand_mode',
        payload: st.payload ?? null,
        duration_sec: typeof st.duration_sec === 'number' ? st.duration_sec : 0,
      }))
    }
    return [{ command: 'stand_mode', payload: null, duration_sec: 0 }]
  })
  const [requiredCapabilities, setRequiredCapabilities] = useState(() => {
    const caps = scenario?.required_capabilities
    if (caps && Array.isArray(caps) && caps.length > 0) {
      return caps.filter((c) => KNOWN_CAPABILITIES.includes(c))
    }
    return ['stand']
  })
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState(null)

  const validate = () => {
    if (!id.trim() || !name.trim()) {
      setError('ID и название обязательны')
      return false
    }
    if (!steps.length) {
      setError('Добавьте хотя бы один шаг')
      return false
    }
    for (let i = 0; i < steps.length; i++) {
      if (!steps[i].command) {
        setError(`Шаг ${i + 1}: выберите команду`)
        return false
      }
      const d = steps[i].duration_sec
      if (typeof d !== 'number' || !Number.isInteger(d)) {
        setError(`Шаг ${i + 1}: duration_sec должен быть целым числом`)
        return false
      }
    }
    if (!requiredCapabilities.length) {
      setError('Выберите хотя бы одну capability')
      return false
    }
    setError(null)
    return true
  }

  const handleSubmit = async () => {
    if (!validate()) return
    const stepsForApi = steps.map((st) => {
      const payload = st.command === 'cmd_vel' && st.payload ? st.payload : null
      return {
        command: st.command,
        payload,
        duration_sec: st.duration_sec,
      }
    })
    setSubmitting(true)
    setError(null)
    try {
      const body = {
        id,
        name,
        description,
        steps: stepsForApi,
        required_capabilities: requiredCapabilities,
      }
      const url = isEdit ? `${API_BASE}/scenarios/${id}` : `${API_BASE}/scenarios`
      const method = isEdit ? 'PUT' : 'POST'
      const res = await apiFetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (res.ok) {
        onSaved?.()
        onClose()
      } else {
        const text = await res.text()
        setError(text || 'Ошибка')
      }
    } catch (e) {
      setError(e.message || 'Ошибка')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: '#1e293b',
          padding: 24,
          borderRadius: 8,
          maxWidth: 500,
          width: '100%',
          maxHeight: '90vh',
          overflow: 'auto',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3 style={{ margin: '0 0 16px 0', color: '#e2e8f0' }}>{isEdit ? 'Сценарий' : 'Создать сценарий'}</h3>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>ID</label>
          <input
            value={id}
            onChange={(e) => setId(e.target.value)}
            disabled={isEdit}
            placeholder="custom_patrol"
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
            }}
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Название</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Мой патруль"
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
            }}
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Описание</label>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Описание сценария"
            style={{
              width: '100%',
              padding: 8,
              borderRadius: 4,
              border: '1px solid #334155',
              background: '#0f172a',
              color: '#e2e8f0',
            }}
          />
        </div>
        <div style={{ marginBottom: 16 }}>
          <StepEditor steps={steps} onChange={setSteps} />
        </div>
        <div style={{ marginBottom: 16 }}>
          <CapabilitySelector selected={requiredCapabilities} onChange={setRequiredCapabilities} />
        </div>
        {error && <div style={{ marginBottom: 12, fontSize: 12, color: '#fca5a5' }}>{error}</div>}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{ ...buttonBase, background: '#475569', color: 'white' }}>
            Отмена
          </button>
          <button
            onClick={handleSubmit}
            disabled={submitting}
            style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
          >
            {submitting ? 'Сохранение...' : isEdit ? 'Сохранить' : 'Создать'}
          </button>
        </div>
      </div>
    </div>
  )
}

function TasksPanel({ tasks, scenarios, onCreateClick, onCancel, readOnly }) {
  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginTop: 24,
        background: '#1e293b',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <h2 style={{ margin: 0, color: '#e2e8f0', fontSize: 18 }}>Задачи</h2>
        {!readOnly && (
          <button onClick={onCreateClick} style={{ ...buttonBase, background: '#0369a1', color: 'white' }}>
            Создать задачу
          </button>
        )}
      </div>
      <div style={{ borderRadius: 6, overflow: 'hidden', border: '1px solid #334155' }}>
        {tasks.length === 0 ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#64748b', fontSize: 14 }}>
            {readOnly ? 'Нет задач.' : 'Нет задач. Нажмите «Создать задачу» для запуска сценария.'}
          </div>
        ) : (
          tasks.map((task) => (
            <TaskRow key={task.id} task={task} scenarios={scenarios} onCancel={onCancel} readOnly={readOnly} />
          ))
        )}
      </div>
    </div>
  )
}

function formatRelativeTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  const now = Date.now()
  const diff = Math.floor((now - d) / 1000)
  if (diff < 60) return `${diff} сек назад`
  if (diff < 3600) return `${Math.floor(diff / 60)} мин назад`
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч назад`
  return d.toLocaleString()
}

function RobotCard({ robot, telemetry, onCommand, onSafeStop, lastCommandSent, showTenant, readOnly, mallAssistantTask, mallAssistantStatus, onStartMallAssistant, onVisitorRequest }) {
  const t = telemetry[robot.id] || {}
  const online = t.online ?? false
  const mockMode = t.mock_mode ?? false
  const hasTelemetry = Object.keys(t).length > 0
  const actuatorStatus = t.actuator_status || 'unknown'
  const currentTask = t.current_task || '-'
  const [visitorText, setVisitorText] = useState('')
  const [visitorSubmitting, setVisitorSubmitting] = useState(false)
  const jointStates = t.joint_states || []
  const [linearX, setLinearX] = useState(0)
  const [linearY, setLinearY] = useState(0)
  const [angularZ, setAngularZ] = useState(0)
  const [jointStatesExpanded, setJointStatesExpanded] = useState(false)
  const [commandHistory, setCommandHistory] = useState([])
  const [historyExpanded, setHistoryExpanded] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)

  useEffect(() => {
    if (!historyExpanded) return
    setHistoryLoading(true)
    const params = new URLSearchParams({ robot_id: robot.id, action: 'command', limit: '10' })
    apiFetch(`${API_BASE}/audit?${params}`)
      .then((r) => (r.ok ? r.json() : []))
      .then(setCommandHistory)
      .catch(() => setCommandHistory([]))
      .finally(() => setHistoryLoading(false))
  }, [historyExpanded, robot.id])

  return (
    <div
      style={{
        border: '1px solid #334155',
        borderRadius: 8,
        padding: 16,
        marginBottom: 12,
        background: '#1e293b',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h3 style={{ margin: '0 0 4px 0' }}>{robot.id}</h3>
          <span style={{ fontSize: 12, color: '#94a3b8' }}>
            {robot.vendor} / {robot.model}
            {robot.location && (
              <span style={{ marginLeft: 8, color: '#94a3b8' }}>· {robot.location}</span>
            )}
            {showTenant && robot.tenant_id && (
              <span style={{ marginLeft: 8, color: '#64748b' }}>· {robot.tenant_id}</span>
            )}
          </span>
          {robot.capabilities && robot.capabilities.length > 0 && (
            <div style={{ marginTop: 6, fontSize: 11, color: '#64748b' }}>
              Capabilities: {robot.capabilities.join(', ')}
            </div>
          )}
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span
            title={online ? 'Робот подключён, телеметрия поступает' : 'Робот отключён'}
            style={{
              padding: '4px 8px',
              borderRadius: 4,
              fontSize: 12,
              background: online ? '#065f46' : '#7f1d1d',
              color: online ? '#6ee7b7' : '#fca5a5',
            }}
          >
            {online ? 'Online' : 'Offline'}
          </span>
          {hasTelemetry && (
            <span
              title={mockMode ? 'Симуляция: команды не идут на реального робота' : 'Режим Live: команды идут на робота'}
              style={{
                padding: '4px 8px',
                borderRadius: 4,
                fontSize: 12,
                background: mockMode ? '#854d0e' : '#065f46',
                color: mockMode ? '#fde047' : '#6ee7b7',
              }}
            >
              {mockMode ? 'Mock' : 'Live'}
            </span>
          )}
          {mallAssistantTask && (
            <span
              title="Mall Assistant scenario"
              style={{
                padding: '4px 8px',
                borderRadius: 4,
                fontSize: 12,
                background: '#7c3aed',
                color: '#e9d5ff',
              }}
            >
              {MALL_ASSISTANT_STATUS[mallAssistantStatus] || 'Idle'}
            </span>
          )}
        </div>
      </div>
      {mallAssistantTask && onVisitorRequest && !readOnly && (
        <div style={{ marginTop: 12, padding: 12, background: '#0f172a', borderRadius: 6 }}>
          <div style={{ fontSize: 11, color: '#64748b', marginBottom: 6, textTransform: 'uppercase' }}>Simulate visitor request</div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <input
              type="text"
              placeholder="e.g. Where is Nike?"
              value={visitorText}
              onChange={(e) => setVisitorText(e.target.value)}
              style={{
                flex: 1,
                minWidth: 140,
                padding: '6px 10px',
                borderRadius: 4,
                border: '1px solid #334155',
                background: '#1e293b',
                color: '#e2e8f0',
                fontSize: 13,
              }}
            />
            <button
              onClick={async () => {
                if (!visitorText.trim()) return
                setVisitorSubmitting(true)
                try {
                  await onVisitorRequest(robot.id, visitorText.trim())
                  setVisitorText('')
                } finally {
                  setVisitorSubmitting(false)
                }
              }}
              disabled={visitorSubmitting || !visitorText.trim()}
              style={{ ...buttonBase, background: '#7c3aed', color: 'white' }}
            >
              {visitorSubmitting ? 'Sending...' : 'Send'}
            </button>
          </div>
        </div>
      )}
      <div style={{ marginTop: 12, padding: 12, background: '#0f172a', borderRadius: 6 }}>
        <button
          onClick={() => setHistoryExpanded(!historyExpanded)}
          style={{
            ...buttonBase,
            background: 'transparent',
            color: '#94a3b8',
            padding: 0,
            fontSize: 12,
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            marginBottom: 8,
          }}
        >
          {historyExpanded ? '▼' : '▶'} История команд
        </button>
        {historyExpanded && (
          <div style={{ fontSize: 12, color: '#94a3b8' }}>
            {historyLoading ? (
              <div style={{ color: '#64748b' }}>Загрузка...</div>
            ) : commandHistory.length === 0 ? (
              <div style={{ color: '#64748b' }}>Нет команд</div>
            ) : (
              commandHistory.map((e) => {
                let cmd = '-'
                try {
                  const d = e.details ? JSON.parse(e.details) : {}
                  cmd = d.command || '-'
                } catch (_) {}
                return (
                  <div key={e.id} style={{ marginBottom: 4 }}>
                    {e.actor || 'system'}: {COMMAND_LABELS[cmd] ?? cmd} — {formatRelativeTime(e.timestamp)}
                  </div>
                )
              })
            )}
          </div>
        )}
      </div>
      <div style={{ marginTop: 12, padding: 12, background: '#0f172a', borderRadius: 6 }}>
        <div style={{ fontSize: 11, color: '#64748b', marginBottom: 6, textTransform: 'uppercase' }}>Состояние</div>
        <div style={{ fontSize: 14, color: '#94a3b8' }}>
          <div>Актуаторы: {actuatorStatus}</div>
          <div>Режим: {currentTask}</div>
          {lastCommandSent?.robotId === robot.id && (
            <div style={{ marginTop: 6, fontSize: 12, color: '#22c55e' }}
                 title="Команда отправлена на робота">
              ✓ Отправлено: {COMMAND_LABELS[lastCommandSent.command] ?? lastCommandSent.command}
            </div>
          )}
        </div>
      </div>
      {!readOnly && (
        <>
          <div style={{ marginTop: 12 }}>
            <div style={{ fontSize: 11, color: '#64748b', marginBottom: 8, textTransform: 'uppercase' }}>Команды</div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {!mallAssistantTask && onStartMallAssistant && (
                <button
                  onClick={() => onStartMallAssistant(robot.id)}
                  style={{ ...buttonBase, background: '#7c3aed', color: 'white' }}
                >
                  Start Mall Assistant
                </button>
              )}
              <button
                onClick={() => onSafeStop(robot.id)}
                style={{ ...buttonBase, background: '#dc2626', color: 'white', fontWeight: 600 }}
              >
                Safe Stop
              </button>
              <button
                onClick={() => onCommand(robot.id, 'release_control')}
                title={MODE_HINTS.release_control}
                style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
              >
                Release
              </button>
              <button
                onClick={() => onCommand(robot.id, 'zero_mode')}
                title={MODE_HINTS.zero_mode}
                style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
              >
                Zero
              </button>
              <button
                onClick={() => onCommand(robot.id, 'stand_mode')}
                title={MODE_HINTS.stand_mode}
                style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
              >
                Stand
              </button>
              <button
                onClick={() => onCommand(robot.id, 'walk_mode')}
                title={MODE_HINTS.walk_mode}
                style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
              >
                Walk
              </button>
            </div>
          </div>
          <div style={{ marginTop: 12, padding: 12, background: '#0f172a', borderRadius: 6 }}>
            <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>cmd_vel (m/s, rad/s)</div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            type="number"
            step="0.1"
            value={linearX}
            onChange={(e) => setLinearX(parseFloat(e.target.value) || 0)}
            placeholder="linear_x"
            style={{ width: 70, padding: 6, borderRadius: 4, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0' }}
          />
          <input
            type="number"
            step="0.1"
            value={linearY}
            onChange={(e) => setLinearY(parseFloat(e.target.value) || 0)}
            placeholder="linear_y"
            style={{ width: 70, padding: 6, borderRadius: 4, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0' }}
          />
          <input
            type="number"
            step="0.1"
            value={angularZ}
            onChange={(e) => setAngularZ(parseFloat(e.target.value) || 0)}
            placeholder="angular_z"
            style={{ width: 70, padding: 6, borderRadius: 4, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0' }}
          />
          <button
            onClick={() => onCommand(robot.id, 'cmd_vel', { linear_x: linearX, linear_y: linearY, angular_z: angularZ })}
            style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
          >
            Drive
          </button>
        </div>
      </div>
        </>
      )}
      {jointStates.length > 0 && (
        <div style={{ marginTop: 12, padding: 12, background: '#0f172a', borderRadius: 6 }}>
          <button
            onClick={() => setJointStatesExpanded(!jointStatesExpanded)}
            style={{
              ...buttonBase,
              background: 'transparent',
              color: '#94a3b8',
              padding: 0,
              fontSize: 12,
              display: 'flex',
              alignItems: 'center',
              gap: 6,
            }}
          >
            {jointStatesExpanded ? '▼' : '▶'} Joint States ({jointStates.length})
          </button>
          {jointStatesExpanded && (
            <div style={{ marginTop: 8, maxHeight: 200, overflow: 'auto', fontSize: 11 }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ color: '#64748b', borderBottom: '1px solid #334155' }}>
                    <th style={{ textAlign: 'left', padding: 4 }}>name</th>
                    <th style={{ textAlign: 'right', padding: 4 }}>pos</th>
                    <th style={{ textAlign: 'right', padding: 4 }}>vel</th>
                    <th style={{ textAlign: 'right', padding: 4 }}>eff</th>
                  </tr>
                </thead>
                <tbody>
                  {jointStates.slice(0, 15).map((js, i) => (
                    <tr key={i} style={{ borderBottom: '1px solid #1e293b' }}>
                      <td style={{ padding: 4, color: '#e2e8f0' }}>{js.name}</td>
                      <td style={{ padding: 4, textAlign: 'right', color: '#94a3b8' }}>{js.position?.toFixed(3) ?? '-'}</td>
                      <td style={{ padding: 4, textAlign: 'right', color: '#94a3b8' }}>{js.velocity?.toFixed(3) ?? '-'}</td>
                      <td style={{ padding: 4, textAlign: 'right', color: '#94a3b8' }}>{js.effort?.toFixed(3) ?? '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {jointStates.length > 15 && (
                <div style={{ marginTop: 4, color: '#64748b', fontSize: 10 }}>
                  ... and {jointStates.length - 15} more
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function App() {
  const [robots, setRobots] = useState([])
  const [telemetry, setTelemetry] = useState({})
  const [safeStopModal, setSafeStopModal] = useState(null)
  const [loading, setLoading] = useState(true)
  const [readOnly, setReadOnly] = useState(false)
  const [lastCommandSent, setLastCommandSent] = useState(null)
  const [tasks, setTasks] = useState([])
  const [scenarios, setScenarios] = useState([])
  const [createTaskModal, setCreateTaskModal] = useState(false)
  const [createTaskInitialScenarioId, setCreateTaskInitialScenarioId] = useState(null)
  const [scenarioModal, setScenarioModal] = useState(null)
  const [zones, setZones] = useState([])
  const [workflows, setWorkflows] = useState([])
  const [workflowRuns, setWorkflowRuns] = useState([])
  const [analyticsSummaries, setAnalyticsSummaries] = useState([])
  const [tenants, setTenants] = useState([])
  const [selectedTenantId, setSelectedTenantId] = useState(() => {
    try {
      return localStorage.getItem('operator-console-tenant') || ''
    } catch {
      return ''
    }
  })
  const [toasts, setToasts] = useState([])
  const [mallAssistantStatus, setMallAssistantStatus] = useState({})

  const addToast = useCallback((type, message) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2)}`
    setToasts((prev) => [...prev, { id, type, message }])
    setTimeout(() => {
      setToasts((p) => p.filter((t) => t.id !== id))
    }, 5000)
  }, [])
  const dismissToast = useCallback((id) => setToasts((p) => p.filter((t) => t.id !== id)), [])

  useEffect(() => {
    try {
      if (selectedTenantId) {
        localStorage.setItem('operator-console-tenant', selectedTenantId)
      } else {
        localStorage.removeItem('operator-console-tenant')
      }
    } catch (_) {}
  }, [selectedTenantId])

  useEffect(() => {
    apiFetch(`${API_BASE}/tenants`)
      .then((r) => r.json())
      .then(setTenants)
      .catch(() => setTenants([]))
  }, [])

  useEffect(() => {
    apiFetch(`${API_BASE}/me`)
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((me) => {
        const roles = me?.roles || []
        const hasWriteRole = roles.some((r) => ['operator', 'administrator'].includes((r || '').toLowerCase()))
        setReadOnly(!hasWriteRole)
      })
      .catch(() => setReadOnly(false))
  }, [])

  useEffect(() => {
    const url = selectedTenantId ? `${API_BASE}/robots?tenant_id=${selectedTenantId}` : `${API_BASE}/robots`
    apiFetch(url)
      .then((r) => r.json())
      .then(setRobots)
      .finally(() => setLoading(false))
  }, [selectedTenantId])

  const refreshScenarios = () => {
    apiFetch(`${API_BASE}/scenarios`)
      .then((r) => r.json())
      .then(setScenarios)
      .catch(() => setScenarios([]))
  }
  useEffect(() => {
    refreshScenarios()
  }, [])

  useEffect(() => {
    const fetchTasks = () => {
      const url = selectedTenantId ? `${API_BASE}/tasks?tenant_id=${selectedTenantId}` : `${API_BASE}/tasks`
      apiFetch(url)
        .then((r) => r.json())
        .then(setTasks)
        .catch(() => setTasks([]))
    }
    fetchTasks()
    const interval = setInterval(fetchTasks, 5000)
    return () => clearInterval(interval)
  }, [selectedTenantId])

  useEffect(() => {
    const fetchZones = () => {
      apiFetch(`${API_BASE}/zones`)
        .then((r) => r.json())
        .then(setZones)
        .catch(() => setZones([]))
    }
    fetchZones()
    const interval = setInterval(fetchZones, 3000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const fetchWorkflows = () => {
      apiFetch(`${API_BASE}/workflows`)
        .then((r) => r.json())
        .then(setWorkflows)
        .catch(() => setWorkflows([]))
    }
    const fetchWorkflowRuns = () => {
      const url = selectedTenantId ? `${API_BASE}/workflow-runs?tenant_id=${selectedTenantId}` : `${API_BASE}/workflow-runs`
      apiFetch(url)
        .then((r) => r.json())
        .then(setWorkflowRuns)
        .catch(() => setWorkflowRuns([]))
    }
    fetchWorkflows()
    fetchWorkflowRuns()
    const interval = setInterval(fetchWorkflowRuns, 5000)
    return () => clearInterval(interval)
  }, [selectedTenantId])

  useEffect(() => {
    const fetchAnalytics = () => {
      const url = selectedTenantId ? `${API_BASE}/analytics/robots?tenant_id=${selectedTenantId}` : `${API_BASE}/analytics/robots`
      apiFetch(url)
        .then((r) => r.json())
        .then(setAnalyticsSummaries)
        .catch(() => setAnalyticsSummaries([]))
    }
    fetchAnalytics()
    const interval = setInterval(fetchAnalytics, 30000)
    return () => clearInterval(interval)
  }, [selectedTenantId])

  useEffect(() => {
    let retryDelay = 1000
    let timeoutId
    let es
    const connect = () => {
      const params = new URLSearchParams()
      if (selectedTenantId) params.set('tenant_id', selectedTenantId)
      const url = streamUrl(`/telemetry/stream${params.toString() ? '?' + params.toString() : ''}`)
      es = new EventSource(url)
      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data)
          setTelemetry((prev) => ({ ...prev, [data.robot_id]: data }))
        } catch (_) {}
      }
      es.onerror = () => {
        es.close()
        timeoutId = setTimeout(() => {
          retryDelay = Math.min(retryDelay * 1.5, 30000)
          connect()
        }, retryDelay)
      }
    }
    connect()
    return () => {
      clearTimeout(timeoutId)
      if (es) es.close()
    }
  }, [selectedTenantId])

  useEffect(() => {
    const url = streamUrl('/events/stream')
    const es = new EventSource(url)
    es.onmessage = (e) => {
      try {
        const ev = JSON.parse(e.data)
        const type = ev.event || ev.type
        const data = ev.data || {}
        if (type === 'safe_stop') {
          addToast('safe_stop', `Safe Stop: ${data.robot_id || 'робот'}`)
        } else if (type === 'robot_online') {
          addToast('robot_online', `Робот онлайн: ${data.robot_id || 'робот'}`)
        } else         if (type === 'task_completed') {
          addToast('task_completed', `Задача завершена: ${data.robot_id || ''} — ${data.status || ''}`)
          if (data.robot_id) setMallAssistantStatus((prev) => ({ ...prev, [data.robot_id]: null }))
        }
        if (type === 'visitor_interaction_started' && data.robot_id) {
          setMallAssistantStatus((prev) => ({ ...prev, [data.robot_id]: 'idle' }))
        }
        if (type === 'navigation_started' && data.robot_id) {
          setMallAssistantStatus((prev) => ({ ...prev, [data.robot_id]: 'guiding_visitor' }))
        }
        if (type === 'navigation_completed' && data.robot_id) {
          setMallAssistantStatus((prev) => ({ ...prev, [data.robot_id]: 'guiding_visitor' }))
        }
        if (type === 'visitor_interaction_finished' && data.robot_id) {
          setMallAssistantStatus((prev) => ({ ...prev, [data.robot_id]: 'returning_to_base' }))
        }
      } catch (_) {}
    }
    es.onerror = () => es.close()
    return () => es.close()
  }, [addToast])

  const sendCommand = async (robotId, command, payload) => {
    const body = { command, operator_id: 'console' }
    if (payload && Object.keys(payload).length > 0) {
      body.payload = payload
    }
    const res = await apiFetch(`${API_BASE}/robots/${robotId}/command`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (res.ok) {
      if (command === 'safe_stop') {
        setSafeStopModal(null)
      }
      setLastCommandSent({ robotId, command })
      setTimeout(() => setLastCommandSent(null), 2500)
    } else {
      alert(`Failed to send ${command}`)
    }
  }

  const handleCommand = (robotId, command, payload) => {
    sendCommand(robotId, command, payload)
  }

  const handleSafeStopClick = (robotId) => {
    setSafeStopModal(robotId)
  }

  const handleCancelTask = async (taskId) => {
    const res = await apiFetch(`${API_BASE}/tasks/${taskId}/cancel`, { method: 'POST' })
    if (res.ok) {
      const task = await res.json()
      setTasks((prev) => prev.map((t) => (t.id === taskId ? task : t)))
    }
  }

  const handleTaskCreated = (task) => {
    setTasks((prev) => [task, ...prev])
    if (task.scenario_id === 'mall_assistant' && task.status === 'running') {
      setMallAssistantStatus((prev) => ({ ...prev, [task.robot_id]: 'idle' }))
    }
  }

  const handleStartMallAssistant = async (robotId) => {
    const res = await apiFetch(`${API_BASE}/scenarios/mall_assistant/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ robot_id: robotId, operator_id: 'console' }),
    })
    if (res.ok) {
      const task = await res.json()
      setTasks((prev) => [task, ...prev])
      setMallAssistantStatus((prev) => ({ ...prev, [robotId]: 'idle' }))
    } else {
      const text = await res.text()
      alert(text || 'Failed to start Mall Assistant')
    }
  }

  const handleVisitorRequest = async (robotId, text) => {
    const res = await apiFetch(`${API_BASE}/scenarios/mall_assistant/visitor-request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ robot_id: robotId, text }),
    })
    if (!res.ok) {
      const text = await res.text()
      alert(text || 'Failed to send visitor request')
    }
  }

  if (loading) return <div style={{ padding: 24 }}>Loading...</div>

  return (
    <div style={{ padding: 24, maxWidth: 600 }}>
      <Toast toasts={toasts} onDismiss={dismissToast} />
      <h1 style={{ marginBottom: 8 }}>
        SAI AUROSY Operator Console
        {readOnly && (
          <span style={{ marginLeft: 12, fontSize: 14, fontWeight: 400, color: '#94a3b8', background: '#334155', padding: '4px 10px', borderRadius: 6 }}>
            Только чтение
          </span>
        )}
      </h1>
      <p style={{ fontSize: 12, color: '#64748b', marginBottom: 16 }}>
        Online — подключение. Mock — симуляция без робота. Live — команды идут на робота. Батарея пока не поддерживается.
      </p>
      <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center', gap: 8 }}>
        <label style={{ fontSize: 12, color: '#94a3b8' }}>Tenant:</label>
        <select
          value={selectedTenantId}
          onChange={(e) => setSelectedTenantId(e.target.value)}
          style={{
            padding: '6px 12px',
            borderRadius: 4,
            border: '1px solid #334155',
            background: '#0f172a',
            color: '#e2e8f0',
            fontSize: 14,
          }}
        >
          <option value="">Все</option>
          {tenants.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name || t.id}
            </option>
          ))}
        </select>
      </div>
      <AnalyticsPanel summaries={analyticsSummaries} />
      <MarketplacePanel
        onUseScenario={(s) => {
          setCreateTaskInitialScenarioId(s.id)
          setCreateTaskModal(true)
        }}
        readOnly={readOnly}
      />
      <ZonesPanel zones={zones} />
      <WorkflowsPanel
        workflows={workflows}
        workflowRuns={workflowRuns}
        onRefresh={() => {
          const url = selectedTenantId ? `${API_BASE}/workflow-runs?tenant_id=${selectedTenantId}` : `${API_BASE}/workflow-runs`
          apiFetch(url)
            .then((r) => r.json())
            .then(setWorkflowRuns)
            .catch(() => setWorkflowRuns([]))
        }}
        readOnly={readOnly}
      />
      {(() => {
        const byLocation = {}
        for (const r of robots) {
          const loc = r.location?.trim() || 'Unassigned'
          if (!byLocation[loc]) byLocation[loc] = []
          byLocation[loc].push(r)
        }
        const locs = Object.keys(byLocation).sort((a, b) => (a === 'Unassigned' ? 1 : a.localeCompare(b)))
        return locs.map((loc) => (
          <div key={loc}>
            <div style={{ fontSize: 12, color: '#64748b', marginBottom: 8, textTransform: 'uppercase', fontWeight: 600 }}>
              {loc}
            </div>
            {byLocation[loc].map((r) => {
              const mallAssistantTask = tasks.find((t) => t.robot_id === r.id && t.scenario_id === 'mall_assistant' && (t.status === 'pending' || t.status === 'running'))
              return (
                <RobotCard
                  key={r.id}
                  robot={r}
                  telemetry={telemetry}
                  onCommand={handleCommand}
                  onSafeStop={handleSafeStopClick}
                  lastCommandSent={lastCommandSent}
                  showTenant={!selectedTenantId}
                  readOnly={readOnly}
                  mallAssistantTask={mallAssistantTask}
                  mallAssistantStatus={mallAssistantStatus[r.id]}
                  onStartMallAssistant={handleStartMallAssistant}
                  onVisitorRequest={handleVisitorRequest}
                />
              )
            })}
          </div>
        ))
      })()}
      <ScenarioBuilderPanel
        scenarios={scenarios}
        onRefresh={refreshScenarios}
        onCreateClick={() => setScenarioModal('create')}
        onEditClick={(s) => setScenarioModal(s)}
        onDeleteClick={async (s) => {
          if (!confirm(`Удалить сценарий "${s.name}"?`)) return
          const res = await apiFetch(`${API_BASE}/scenarios/${s.id}`, { method: 'DELETE' })
          if (res.ok) {
            refreshScenarios()
          } else {
            alert('Ошибка удаления')
          }
        }}
        readOnly={readOnly}
      />
      <TasksPanel
        tasks={tasks}
        scenarios={scenarios}
        onCreateClick={() => setCreateTaskModal(true)}
        onCancel={handleCancelTask}
        readOnly={readOnly}
      />
      {createTaskModal && (
        <CreateTaskModal
          robots={robots}
          scenarios={scenarios}
          onClose={() => {
            setCreateTaskModal(false)
            setCreateTaskInitialScenarioId(null)
          }}
          onCreate={handleTaskCreated}
          initialScenarioId={createTaskInitialScenarioId}
        />
      )}
      {scenarioModal && (
        <ScenarioModal
          scenario={scenarioModal === 'create' ? null : scenarioModal}
          onClose={() => setScenarioModal(null)}
          onSaved={refreshScenarios}
        />
      )}
      {safeStopModal && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.6)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          onClick={() => setSafeStopModal(null)}
        >
          <div
            style={{
              background: '#1e293b',
              padding: 24,
              borderRadius: 8,
              maxWidth: 360,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <p>Вы уверены? Отправить safe_stop для {safeStopModal}?</p>
            <div style={{ display: 'flex', gap: 12, marginTop: 16 }}>
              <button
                onClick={() => sendCommand(safeStopModal, 'safe_stop', null)}
                style={{
                  padding: '8px 16px',
                  background: '#dc2626',
                  color: 'white',
                  border: 'none',
                  borderRadius: 6,
                  cursor: 'pointer',
                }}
              >
                Да, Safe Stop
              </button>
              <button
                onClick={() => setSafeStopModal(null)}
                style={{
                  padding: '8px 16px',
                  background: '#475569',
                  color: 'white',
                  border: 'none',
                  borderRadius: 6,
                  cursor: 'pointer',
                }}
              >
                Отмена
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
