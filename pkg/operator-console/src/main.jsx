import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

class ErrorBoundary extends React.Component {
  state = { hasError: false, error: null }
  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }
  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 24, color: '#fca5a5', fontFamily: 'monospace', fontSize: 14 }}>
          <h2 style={{ color: '#e2e8f0' }}>Ошибка приложения</h2>
          <pre style={{ overflow: 'auto', background: '#1e293b', padding: 12, borderRadius: 6 }}>
            {this.state.error?.toString?.() || 'Unknown error'}
          </pre>
        </div>
      )
    }
    return this.props.children
  }
}

const root = document.getElementById('root')
if (root) {
  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </React.StrictMode>,
  )
}
