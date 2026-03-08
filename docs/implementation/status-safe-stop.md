# Status + Safe Stop — Runbook

Минимальный сценарий: телеметрия AGIBOT X1 в Operator Console и команда safe_stop.

## Компоненты

- **Control Plane** — Fleet Registry, API, Command Arbiter, Safety Supervisor, SSE telemetry stream
- **NATS** — Event Broker / Telemetry Bus
- **AGIBOT Adapter** — мост ROS2 ↔ NATS (или mock-режим)
- **Operator Console** — React SPA, список роботов, real-time телеметрия, кнопка Safe Stop

## Запуск (Docker Compose)

```bash
docker compose up -d
```

Сервисы:
- NATS: `localhost:4222`
- Control Plane: `http://localhost:8080`
- Operator Console: `http://localhost:3000`
- AGIBOT Adapter (mock): публикует телеметрию x1-001 каждую секунду
- Unitree Adapter (mock): публикует телеметрию go2-001 каждую секунду

## Запуск (локально)

### 1. NATS

```bash
docker run -d -p 4222:4222 -p 8222:8222 nats:2-alpine -m 8222
```

### 2. Control Plane

```bash
export NATS_URL=nats://localhost:4222
go run ./cmd/control-plane
```

### 3. AGIBOT Adapter (mock)

```bash
pip install nats-py
export ROBOT_ID=x1-001
export NATS_URL=nats://localhost:4222
export AGIBOT_MOCK=1
python pkg/adapters/agibot/adapter.py
```

### 3b. Unitree Adapter (mock)

```bash
pip install nats-py
export ROBOT_ID=go2-001
export NATS_URL=nats://localhost:4222
export UNITREE_MOCK=1
python pkg/adapters/unitree/adapter.py
```

### 4. Operator Console

```bash
cd pkg/operator-console
npm install
npm run dev
```

Открыть http://localhost:3000. Vite проксирует `/api` на Control Plane.

## E2E: Safe Stop

1. Открыть http://localhost:3000
2. Убедиться, что робот `x1-001` в статусе Online (телеметрия от адаптера)
3. Нажать **Safe Stop**
4. Подтвердить в модальном окне
5. Команда уходит: Console → API POST /robots/x1-001/command → NATS commands.robots.x1-001 → Adapter
6. В режиме ROS2: Adapter публикует в `/start_control` → X1 Infer переводит в idle

## AGIBOT X1 (реальный робот / симуляция)

Для работы с X1 Infer (симуляция или реальный робот):

1. Установить ROS2 Humble, собрать X1 Infer
2. Запустить `./run_sim.sh` или `./run.sh`
3. Запустить адаптер **без** `AGIBOT_MOCK`:
   ```bash
   pip install nats-py rclpy
   source /opt/ros/humble/setup.bash
   source /path/to/AGIBOT_x1_infer/build/install/ros2_setup.sh
   export ROBOT_ID=x1-001
   export NATS_URL=nats://localhost:4222
   python pkg/adapters/agibot/adapter.py
   ```

## Unitree Go2 (реальный робот / unitree_ros2)

Для работы с Go2 (реальный робот или unitree_ros2):

1. Установить ROS2 Humble, собрать unitree_ros2 (cyclonedds_ws)
2. Подключить робот по Ethernet (192.168.123.x) или использовать `setup_local.sh` для loopback
3. Запустить адаптер **без** `UNITREE_MOCK`:
   ```bash
   pip install nats-py rclpy
   source /opt/ros/humble/setup.bash
   source /path/to/unitree_ros2/cyclonedds_ws/install/setup.bash
   source /path/to/unitree_ros2/setup.sh
   export ROBOT_ID=go2-001
   export NATS_URL=nats://localhost:4222
   python pkg/adapters/unitree/adapter.py
   ```

Альтернатива: unitree_sdk2_python для команд (без ROS2).
