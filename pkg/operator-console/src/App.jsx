import { useState, useEffect } from 'react'

const API_BASE = '/api/v1'
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:4222'

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

const STATUS_LABELS = {
  pending: 'Ожидание',
  running: 'Выполняется',
  completed: 'Завершена',
  failed: 'Ошибка',
  cancelled: 'Отменена',
}

function TaskRow({ task, scenarios, onCancel }) {
  const scenario = scenarios.find((s) => s.id === task.scenario_id)
  const scenarioName = scenario?.name ?? task.scenario_id
  const canCancel = task.status === 'pending' || task.status === 'running'
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

function CreateTaskModal({ robots, scenarios, onClose, onCreate }) {
  const [robotId, setRobotId] = useState('')
  const [scenarioId, setScenarioId] = useState('')
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
      const res = await fetch(`${API_BASE}/tasks`, {
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

function WorkflowsPanel({ workflows, workflowRuns, onRefresh }) {
  const [running, setRunning] = useState(null)
  const handleRun = async (wfId) => {
    setRunning(wfId)
    try {
      const res = await fetch(`${API_BASE}/workflows/${wfId}/run`, {
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
              <button
                onClick={() => handleRun(wf.id)}
                disabled={running === wf.id}
                style={{ ...buttonBase, background: '#0369a1', color: 'white' }}
              >
                {running === wf.id ? 'Запуск...' : 'Запустить'}
              </button>
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

function TasksPanel({ tasks, scenarios, onCreateClick, onCancel }) {
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
        <button onClick={onCreateClick} style={{ ...buttonBase, background: '#0369a1', color: 'white' }}>
          Создать задачу
        </button>
      </div>
      <div style={{ borderRadius: 6, overflow: 'hidden', border: '1px solid #334155' }}>
        {tasks.length === 0 ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#64748b', fontSize: 14 }}>
            Нет задач. Нажмите «Создать задачу» для запуска сценария.
          </div>
        ) : (
          tasks.map((task) => (
            <TaskRow key={task.id} task={task} scenarios={scenarios} onCancel={onCancel} />
          ))
        )}
      </div>
    </div>
  )
}

function RobotCard({ robot, telemetry, onCommand, onSafeStop, lastCommandSent }) {
  const t = telemetry[robot.id] || {}
  const online = t.online ?? false
  const mockMode = t.mock_mode ?? false
  const hasTelemetry = Object.keys(t).length > 0
  const actuatorStatus = t.actuator_status || 'unknown'
  const currentTask = t.current_task || '-'
  const jointStates = t.joint_states || []
  const [linearX, setLinearX] = useState(0)
  const [linearY, setLinearY] = useState(0)
  const [angularZ, setAngularZ] = useState(0)
  const [jointStatesExpanded, setJointStatesExpanded] = useState(false)

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
        </div>
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
      <div style={{ marginTop: 12 }}>
        <div style={{ fontSize: 11, color: '#64748b', marginBottom: 8, textTransform: 'uppercase' }}>Команды</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
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
  const [lastCommandSent, setLastCommandSent] = useState(null)
  const [tasks, setTasks] = useState([])
  const [scenarios, setScenarios] = useState([])
  const [createTaskModal, setCreateTaskModal] = useState(false)
  const [zones, setZones] = useState([])
  const [workflows, setWorkflows] = useState([])
  const [workflowRuns, setWorkflowRuns] = useState([])
  const [analyticsSummaries, setAnalyticsSummaries] = useState([])

  useEffect(() => {
    fetch(`${API_BASE}/robots`)
      .then((r) => r.json())
      .then(setRobots)
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetch(`${API_BASE}/scenarios`)
      .then((r) => r.json())
      .then(setScenarios)
      .catch(() => setScenarios([]))
  }, [])

  useEffect(() => {
    const fetchTasks = () => {
      fetch(`${API_BASE}/tasks`)
        .then((r) => r.json())
        .then(setTasks)
        .catch(() => setTasks([]))
    }
    fetchTasks()
    const interval = setInterval(fetchTasks, 5000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const fetchZones = () => {
      fetch(`${API_BASE}/zones`)
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
      fetch(`${API_BASE}/workflows`)
        .then((r) => r.json())
        .then(setWorkflows)
        .catch(() => setWorkflows([]))
    }
    const fetchWorkflowRuns = () => {
      fetch(`${API_BASE}/workflow-runs`)
        .then((r) => r.json())
        .then(setWorkflowRuns)
        .catch(() => setWorkflowRuns([]))
    }
    fetchWorkflows()
    fetchWorkflowRuns()
    const interval = setInterval(() => {
      fetchWorkflowRuns()
    }, 5000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const fetchAnalytics = () => {
      fetch(`${API_BASE}/analytics/robots`)
        .then((r) => r.json())
        .then(setAnalyticsSummaries)
        .catch(() => setAnalyticsSummaries([]))
    }
    fetchAnalytics()
    const interval = setInterval(fetchAnalytics, 30000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const es = new EventSource(`${API_BASE}/telemetry/stream`)
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        setTelemetry((prev) => ({ ...prev, [data.robot_id]: data }))
      } catch (_) {}
    }
    es.onerror = () => es.close()
    return () => es.close()
  }, [])

  const sendCommand = async (robotId, command, payload) => {
    const body = { command, operator_id: 'console' }
    if (payload && Object.keys(payload).length > 0) {
      body.payload = payload
    }
    const res = await fetch(`${API_BASE}/robots/${robotId}/command`, {
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
    const res = await fetch(`${API_BASE}/tasks/${taskId}/cancel`, { method: 'POST' })
    if (res.ok) {
      const task = await res.json()
      setTasks((prev) => prev.map((t) => (t.id === taskId ? task : t)))
    }
  }

  const handleTaskCreated = (task) => {
    setTasks((prev) => [task, ...prev])
  }

  if (loading) return <div style={{ padding: 24 }}>Loading...</div>

  return (
    <div style={{ padding: 24, maxWidth: 600 }}>
      <h1 style={{ marginBottom: 8 }}>SAI AUROSY Operator Console</h1>
      <p style={{ fontSize: 12, color: '#64748b', marginBottom: 24 }}>
        Online — подключение. Mock — симуляция без робота. Live — команды идут на робота. Батарея пока не поддерживается.
      </p>
      <AnalyticsPanel summaries={analyticsSummaries} />
      <ZonesPanel zones={zones} />
      <WorkflowsPanel
        workflows={workflows}
        workflowRuns={workflowRuns}
        onRefresh={() =>
          fetch(`${API_BASE}/workflow-runs`)
            .then((r) => r.json())
            .then(setWorkflowRuns)
            .catch(() => setWorkflowRuns([]))
        }
      />
      {robots.map((r) => (
        <RobotCard
          key={r.id}
          robot={r}
          telemetry={telemetry}
          onCommand={handleCommand}
          onSafeStop={handleSafeStopClick}
          lastCommandSent={lastCommandSent}
        />
      ))}
      <TasksPanel
        tasks={tasks}
        scenarios={scenarios}
        onCreateClick={() => setCreateTaskModal(true)}
        onCancel={handleCancelTask}
      />
      {createTaskModal && (
        <CreateTaskModal
          robots={robots}
          scenarios={scenarios}
          onClose={() => setCreateTaskModal(false)}
          onCreate={handleTaskCreated}
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
