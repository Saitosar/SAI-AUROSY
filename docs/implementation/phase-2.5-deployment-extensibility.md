# Phase 2.5 — Deployment и расширяемость (приоритет 5)

Phase 2.5 реализует Edge/Cloud Deployment, общий ROS1/ROS2 адаптер и Adapter SDK.

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Edge/Cloud** | Edge Agent (локальный NATS, safety, heartbeat), Cloud API /v1/edges, /v1/edges/:id/heartbeat, синхронизация команд |
| **ROS Adapter** | Универсальный адаптер ROS1/ROS2, конфигурация через ROS_DISTRO и топики |
| **Adapter SDK** | Документация контракта RobotAdapter, шаблон адаптера, конвенция robot_id prefix |

## Edge/Cloud Deployment

### Edge Agent

- `cmd/edge-agent` — сервис, подключающийся к локальному NATS
- Heartbeat в Cloud Control Plane каждые N секунд
- Получение pending_commands и публикация в локальный NATS
- Локальный Safety Supervisor (валидация команд перед relay)

Переменные: `EDGE_ID`, `NATS_URL`, `CLOUD_URL`, `EDGE_ROBOT_IDS`, `EDGE_HEARTBEAT_SEC`.

### Cloud API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/edges | Список edge-узлов |
| GET | /v1/edges/:id | Статус edge |
| POST | /v1/edges/:id/heartbeat | Heartbeat от edge (возвращает pending_commands) |

### Роботы с edge_id

При `PUT /v1/robots/:id` можно указать `edge_id`. Команды для таких роботов попадают в очередь и доставляются edge-agent при heartbeat.

## ROS Adapter

- `pkg/adapters/ros/` — универсальный адаптер
- `ROS_DISTRO=noetic` — ROS1, `ROS_DISTRO=humble` — ROS2
- Конфигурируемые топики через env
- Mock: `ROS_MOCK=1`

См. [docs/vendors/ros.md](../vendors/ros.md).

## Adapter SDK

- [RobotAdapter Contract](../adapters/robot-adapter-contract.md) — интерфейс, NATS-топики, JSON-форматы
- [Template adapter](../../pkg/adapters/template/README.md) — минимальный пример
- [Adapter Layer](../architecture/adapter-layer.md) — таблица robot_id prefix

## Запуск Edge

```bash
# 1. Cloud
docker compose up -d

# 2. Назначить edge_id роботам (через API)
curl -X PUT http://localhost:8080/v1/robots/x1-001 \
  -H "Content-Type: application/json" \
  -d '{"id":"x1-001","vendor":"agibot","model":"X1","adapter_endpoint":"nats://nats:4222","edge_id":"edge-001"}'

# 3. Edge
CLOUD_URL=http://host.docker.internal:8080 docker compose -f docker-compose.edge.yml up -d
```

## Ссылки

- [Deployment Model](../architecture/deployment-model.md)
- [Adapter Layer](../architecture/adapter-layer.md)
- [Roadmap](../product/roadmap.md)
