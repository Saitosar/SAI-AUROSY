# Production Deployment Guide

Руководство по переносу SAI AUROSY в production-среду.

## Обзор архитектуры

| Компонент | Технология | Назначение |
|-----------|------------|------------|
| Operator Console | React + Vite | UI для управления роботами |
| Control Plane | Go | REST API, auth, streaming, cognitive gateway |
| NATS | Event broker | Телеметрия, события, очереди |
| PostgreSQL | БД | Роботы, задачи, сценарии, audit, webhooks |
| Gemini Adapter | Python | STT, TTS, intent (речь) |
| Robot Adapters | Go | Связь с роботами (AGIBOT, Unitree) |

---

## Рекомендуемый стек

| Компонент | Рекомендация | Альтернатива |
|-----------|--------------|--------------|
| Frontend | Vercel | Netlify, Cloudflare Pages |
| База данных | Supabase (PostgreSQL) | Railway, Neon, собственный PostgreSQL |
| Backend | Hetzner VPS | AWS EC2, DigitalOcean, любой VPS |
| Домен + SSL | Cloudflare / Let's Encrypt | — |

---

## 1. Frontend (Vercel)

### 1.1. Настройка проекта

- **Root Directory:** `pkg/operator-console`
- **Build Command:** `npm run build`
- **Output Directory:** `dist`

### 1.2. Переменные окружения

| Переменная | Описание | Пример |
|------------|----------|--------|
| `VITE_API_BASE` | URL Control Plane API | `https://api.yourdomain.com/v1` |
| `VITE_API_KEY` | API key для авторизации (опционально) | `sk-...` |

### 1.3. Изменения в коде

В `pkg/operator-console/src/App.jsx` используется `API_BASE = '/api/v1'`. Для production необходимо поддержать внешний URL:

```javascript
const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'
```

При деплое на Vercel `VITE_API_BASE` будет указывать на ваш Control Plane.

### 1.4. SSE (EventSource)

Телеметрия и события используют Server-Sent Events (`/v1/telemetry/stream`, `/v1/events/stream`). При cross-origin запросах Control Plane должен возвращать корректные CORS-заголовки (см. раздел 3).

---

## 2. База данных (Supabase)

Supabase предоставляет managed PostgreSQL, совместимый с Control Plane.

### 2.1. Создание проекта

1. Создайте проект в [Supabase](https://supabase.com)
2. В **Settings → Database** скопируйте connection string
3. Используйте **Connection pooling** (порт 6543) для лучшей производительности

### 2.2. Connection string

Формат DSN для Control Plane:

```
postgres://postgres.[PROJECT_REF]:[PASSWORD]@aws-0-[REGION].pooler.supabase.com:6543/postgres?sslmode=require
```

Или прямой (без pooling):

```
postgres://postgres:[PASSWORD]@db.[PROJECT_REF].supabase.co:5432/postgres?sslmode=require
```

### 2.3. Миграции

Control Plane автоматически применяет миграции при старте. Таблицы создаются в схеме `public`. Дополнительных действий не требуется.

### 2.4. Безопасность

- Никогда не коммитьте DSN в репозиторий
- Храните пароль в секретах (Vercel, VPS env, Vault)
- Supabase поддерживает connection pooling — рекомендуется для production

---

## 3. Backend (Hetzner VPS)

### 3.1. Сервисы на VPS

| Сервис | Порт | Описание |
|--------|------|----------|
| NATS | 4222 (client), 8222 (monitoring) | Event broker |
| Control Plane | 8080 | REST API |
| Gemini Adapter | 8001 | Speech (STT, TTS, Intent) |
| Reverse proxy | 443 | HTTPS, маршрутизация |

### 3.2. Docker Compose (production)

Создайте `docker-compose.prod.yml`:

```yaml
services:
  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-m", "8222"]
    restart: unless-stopped

  control-plane:
    build:
      context: .
      dockerfile: Dockerfile.control-plane
    ports:
      - "8080:8080"
    environment:
      NATS_URL: nats://nats:4222
      CONTROL_PLANE_ADDR: ":8080"
      REGISTRY_DB_DRIVER: postgres
      REGISTRY_DB_DSN: ${REGISTRY_DB_DSN}
      JWT_SECRET: ${JWT_SECRET}
      JWT_ISSUER: sai-aurosy
      CORS_ORIGINS: ${CORS_ORIGINS}
      LOG_FORMAT: json
      AUTH_REQUIRED: "true"
      COGNITIVE_PROVIDER: http
      COGNITIVE_HTTP_TRANSCRIBE_URL: http://gemini-adapter:8001/transcribe
      COGNITIVE_HTTP_SYNTHESIZE_URL: http://gemini-adapter:8001/synthesize
      COGNITIVE_HTTP_INTENT_URL: http://gemini-adapter:8001/understand-intent
      COGNITIVE_HTTP_TRANSLATE_URL: http://gemini-adapter:8001/translate
    depends_on:
      - nats
      - gemini-adapter
    restart: unless-stopped

  gemini-adapter:
    build:
      context: ./examples/integration/gemini-adapter
      dockerfile: Dockerfile
    ports:
      - "8001:8001"
    environment:
      GEMINI_API_KEY: ${GEMINI_API_KEY}
      PORT: "8001"
    restart: unless-stopped
```

### 3.3. Переменные окружения (VPS)

Создайте `.env` на VPS (не коммитить в git):

```bash
# Database (Supabase)
REGISTRY_DB_DSN=postgres://postgres.[REF]:[PASSWORD]@aws-0-[REGION].pooler.supabase.com:6543/postgres?sslmode=require

# Auth
JWT_SECRET=<openssl rand -base64 32>

# CORS — домены фронтенда через запятую
CORS_ORIGINS=https://your-app.vercel.app,https://yourdomain.com

# Gemini (для speech)
GEMINI_API_KEY=AIza...
```

### 3.4. HTTPS (Reverse Proxy)

Используйте Caddy или nginx с Let's Encrypt.

**Caddy** (автоматический HTTPS):

```
api.yourdomain.com {
    reverse_proxy localhost:8080
}
```

**nginx** — настройте SSL через certbot:

```nginx
server {
    listen 443 ssl;
    server_name api.yourdomain.com;
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 4. Robot Adapters (опционально)

Адаптеры (agibot-adapter, unitree-adapter) нужны только при наличии физических роботов. Варианты:

- **На VPS** — если роботы в той же сети
- **На edge-устройстве** — рядом с роботами при распределённой архитектуре

Для mock-режима или без роботов адаптеры можно не запускать.

---

## 5. Секреты и безопасность

### 5.1. Обязательные секреты

| Секрет | Где хранить | Описание |
|--------|-------------|----------|
| `JWT_SECRET` | VPS env, Vault | HMAC для JWT |
| `REGISTRY_DB_DSN` | VPS env, Vault | Подключение к PostgreSQL |
| `GEMINI_API_KEY` | VPS env | Google Gemini API |

### 5.2. Генерация JWT_SECRET

```bash
openssl rand -base64 32
```

### 5.3. Secrets Management

Для production рекомендуется Vault или AWS Secrets Manager. См. [Secrets Management](implementation/secrets-management.md).

---

## 6. Чеклист деплоя

- [ ] **Supabase:** создать проект, получить DSN
- [ ] **VPS:** установить Docker, настроить firewall (22, 80, 443)
- [ ] **VPS:** настроить reverse proxy и HTTPS
- [ ] **VPS:** создать `.env` с секретами
- [ ] **VPS:** запустить `docker compose -f docker-compose.prod.yml up -d`
- [ ] **Код:** добавить поддержку `VITE_API_BASE` в Operator Console
- [ ] **Vercel:** добавить переменные `VITE_API_BASE`, `VITE_API_KEY`
- [ ] **Vercel:** задеплоить Operator Console (root: `pkg/operator-console`)
- [ ] **Control Plane:** проверить `CORS_ORIGINS` (домен Vercel)
- [ ] **API:** создать API key для Operator Console
- [ ] **Проверка:** телеметрия, SSE, команды, Speech Test

---

## 7. Схема развёртывания

```
                    [Пользователь]
                           │
                           ▼
              [Vercel] Operator Console (React)
                           │
                           │ VITE_API_BASE → https://api.yourdomain.com
                           │
                           ▼
                   [Hetzner VPS]
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
   nginx/Caddy      Control Plane      Gemini Adapter
   (HTTPS :443)     (Go :8080)        (Python :8001)
         │                 │                 │
         │                 ├── NATS :4222    └── Google Gemini API
         │                 │
         │                 └── Supabase PostgreSQL
         │
         └── /v1/* → Control Plane
```

---

## 8. Дополнительные рекомендации

| Потребность | Решение |
|-------------|---------|
| Мониторинг | Prometheus + Grafana (на VPS или облачный) |
| Логи | Loki, Papertrail, CloudWatch |
| Резервные копии БД | Supabase встроенные бэкапы; дополнительно pg_dump |
| CI/CD | GitHub Actions для билда и деплоя на VPS |
| Домен | Зарегистрировать домен, настроить DNS (A-запись на IP VPS) |

---

## Связанные документы

- [Production Runbook](operations/production-runbook.md) — мониторинг, алерты, восстановление
- [Secrets Management](implementation/secrets-management.md) — Vault, AWS Secrets Manager
- [Phase 2.1 Control Plane](implementation/phase-2.1-control-plane.md) — переменные окружения
- [Integration Guide](integration/README.md) — API, аутентификация
