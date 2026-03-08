import { useState, useEffect } from 'react'

const API_BASE = '/api'
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:4222'

const buttonBase = {
  padding: '8px 16px',
  border: 'none',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: 14,
}

function RobotCard({ robot, telemetry, onCommand, onSafeStop }) {
  const t = telemetry[robot.id] || {}
  const online = t.online ?? false
  const actuatorStatus = t.actuator_status || 'unknown'
  const currentTask = t.current_task || '-'

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
        </div>
        <span
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
      </div>
      <div style={{ marginTop: 12, fontSize: 14, color: '#94a3b8' }}>
        <div>Actuator: {actuatorStatus}</div>
        <div>Task: {currentTask}</div>
      </div>
      <div style={{ marginTop: 12, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        <button
          onClick={() => onSafeStop(robot.id)}
          style={{ ...buttonBase, background: '#dc2626', color: 'white', fontWeight: 600 }}
        >
          Safe Stop
        </button>
        <button
          onClick={() => onCommand(robot.id, 'release_control')}
          style={{ ...buttonBase, background: '#475569', color: 'white' }}
        >
          Release
        </button>
        <button
          onClick={() => onCommand(robot.id, 'zero_mode')}
          style={{ ...buttonBase, background: '#475569', color: 'white' }}
        >
          Zero
        </button>
        <button
          onClick={() => onCommand(robot.id, 'stand_mode')}
          style={{ ...buttonBase, background: '#475569', color: 'white' }}
        >
          Stand
        </button>
        <button
          onClick={() => onCommand(robot.id, 'walk_mode')}
          style={{ ...buttonBase, background: '#475569', color: 'white' }}
        >
          Walk
        </button>
      </div>
    </div>
  )
}

export default function App() {
  const [robots, setRobots] = useState([])
  const [telemetry, setTelemetry] = useState({})
  const [safeStopModal, setSafeStopModal] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch(`${API_BASE}/robots`)
      .then((r) => r.json())
      .then(setRobots)
      .finally(() => setLoading(false))
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

  const sendCommand = async (robotId, command) => {
    const res = await fetch(`${API_BASE}/robots/${robotId}/command`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ command, operator_id: 'console' }),
    })
    if (res.ok && command === 'safe_stop') {
      setSafeStopModal(null)
    } else if (!res.ok) {
      alert(`Failed to send ${command}`)
    }
  }

  const handleCommand = (robotId, command) => {
    sendCommand(robotId, command)
  }

  const handleSafeStopClick = (robotId) => {
    setSafeStopModal(robotId)
  }

  if (loading) return <div style={{ padding: 24 }}>Loading...</div>

  return (
    <div style={{ padding: 24, maxWidth: 600 }}>
      <h1 style={{ marginBottom: 24 }}>SAI AUROSY Operator Console</h1>
      {robots.map((r) => (
        <RobotCard
          key={r.id}
          robot={r}
          telemetry={telemetry}
          onCommand={handleCommand}
          onSafeStop={handleSafeStopClick}
        />
      ))}
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
                onClick={() => sendCommand(safeStopModal, 'safe_stop')}
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
