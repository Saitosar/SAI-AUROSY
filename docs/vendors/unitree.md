# Unitree

## Overview

Unitree Robotics — производитель четвероногих и гуманоидных роботов. Линейка продуктов: **Go2**, **Go2W**, **B2**, **B2W**, **H1**, **H1-2**, **G1**, **A1**, **A2**, **B1**, **Z1**, **R1**, **R1 Air**, **Laikago**, **Aliengo**, **AliengoZ1**, **Dexterous Hand**. Основные SDK и ROS-пакеты — open-source, документация на [Unitree Document Center](https://support.unitree.com/home/zh/developer).

---

## Ключевые ресурсы

| Ресурс | URL | Назначение |
|--------|-----|------------|
| **unitree_sdk2** | https://github.com/unitreerobotics/unitree_sdk2 | SDK v2: CycloneDDS, Go2/B2/H1/G1, C++ |
| **unitree_ros** | https://github.com/unitreerobotics/unitree_ros | ROS1: симуляция Gazebo, описания роботов |
| **unitree_ros2** | https://github.com/unitreerobotics/unitree_ros2 | ROS2: DDS-совместимость, Go2/B2/H1 |
| **unitree_ros_to_real** | https://github.com/unitreerobotics/unitree_ros_to_real | ROS1 → реальный робот, unitree_legged_msgs |
| **Unitree Document Center** | https://support.unitree.com/home/zh/developer | Документация API, Sports Services, Basic Services |

---

## unitree_sdk2

**Репозиторий:** [unitree_sdk2](https://github.com/unitreerobotics/unitree_sdk2) | ~926 stars

**Что это:** SDK v2 для Unitree-роботов. Коммуникация на базе **CycloneDDS**. Поддерживает Go2, B2, H1, G1 и др.

### Среда сборки

- **ОС:** Ubuntu 20.04 LTS
- **Архитектура:** aarch64, x86_64
- **Компилятор:** GCC 9.4.0
- **Зависимости:** CMake ≥3.10, libyaml-cpp-dev, libeigen3-dev, libboost-all-dev, libspdlog-dev, libfmt-dev

### Структура репозитория

```text
unitree_sdk2/
├── include/unitree/     # Заголовки API
├── lib/                 # Библиотеки
├── example/             # Примеры по моделям
│   ├── go2, go2w        # Go2
│   ├── b2, b2w          # B2
│   ├── h1               # H1
│   ├── g1               # G1
│   ├── a2               # A2
│   ├── helloworld       # Базовый пример
│   ├── jsonize          # JSON-сериализация
│   ├── state_machine    # State machine
│   └── wireless_controller
├── cmake/               # CMake-модули
└── thirdparty/          # Зависимости
```

### Установка

```bash
mkdir build && cd build
cmake .. -DCMAKE_INSTALL_PREFIX=/opt/unitree_robotics
sudo make install
```

---

## unitree_ros (ROS1)

**Репозиторий:** [unitree_ros](https://github.com/unitreerobotics/unitree_ros) | ~1264 stars

**Что это:** ROS1-пакеты для симуляции в Gazebo и управления реальными роботами. **Не поддерживает high-level walking** в Gazebo — только low-level control (torque, position, velocity). Реальные роботы — через [unitree_ros_to_real](https://github.com/unitreerobotics/unitree_ros_to_real).

### Пакеты

| Пакет | Описание |
|-------|----------|
| `robots/*_description` | URDF/Xacro, mesh для моделей |
| `unitree_controller` | Контроллеры суставов (position, velocity, torque) |
| `z1_controller` | Контроллер Z1 |
| `unitree_gazebo` | Миры Gazebo |
| `unitree_legged_control` | Legged control для симуляции |

### Поддерживаемые модели (описания)

a1, a2, aliengo, aliengoZ1, b1, b2, b2w, dexterous_hand, g1, go1, go2, go2w, h1, h1_2, h2, laikago, r1, r1_air, z1

### Зависимости

- ROS Melodic или Kinetic
- Gazebo 8
- [unitree_legged_msgs](https://github.com/unitreerobotics/unitree_ros_to_real) (submodule unitree_ros_to_real)

### Запуск симуляции

```bash
roslaunch unitree_gazebo normal.launch rname:=go1 wname:=stairs
# rname: laikago, aliengo, a1, go1
# wname: earth, space, stairs
```

---

## unitree_ros2

**Репозиторий:** [unitree_ros2](https://github.com/unitreerobotics/unitree_ros2) | ~602 stars

**Что это:** ROS2-поддержка для Go2, B2, H1. Использует тот же DDS (CycloneDDS), что и SDK2 — ROS2-сообщения работают напрямую без обёртки SDK.

### Системные требования

| ОС | ROS2 |
|----|------|
| Ubuntu 20.04 | Foxy |
| Ubuntu 22.04 | Humble (рекомендуется) |

### Структура

```text
unitree_ros2/
├── cyclonedds_ws/       # unitree_go, unitree_api (msg definitions)
├── example/             # Примеры (read_motion_state, low_level_ctrl, sport_mode_ctrl)
├── docs/                # Документация
├── setup.sh             # Настройка сети для робота
├── setup_local.sh       # Локальный loopback (симуляция)
└── setup_default.sh     # Без указания интерфейса
```

### Сетевая конфигурация

- Робот и ПК — Ethernet
- IP ПК: 192.168.123.99, маска 255.255.255.0
- В `setup.sh` указать сетевой интерфейс (например, `enp3s0`)

### Ключевые топики и сообщения

| Топик | Тип | Описание |
|-------|-----|----------|
| `sportmodestate` / `lf/sportmodestate` | SportModeState | Позиция, скорость, gait, foot position |
| `lowstate` / `lf/lowstate` | LowState | MotorState[20], IMU, BMS, foot_force |
| `wirelesscontroller` | WirelessController | Joystick (lx, ly, rx, ry, keys) |
| `/api/sport/request` | unitree_api::msg::Request | Sportmode-команды (Euler, Move, Stand и т.д.) |
| `lowcmd` | LowCmd | MotorCmd[20] — torque, position, velocity |

### Sport mode (примеры)

| Mode | Описание |
|------|----------|
| 0 | idle (default stand) |
| 1 | balanceStand |
| 2 | pose |
| 3 | locomotion |
| 5 | lieDown |
| 6 | jointLock |
| 7 | damping |
| 10 | sit |
| 11–13 | frontFlip, frontJump, frontPounc |

### Gait type

0: idle, 1: trot, 2: run, 3: climb stair, 4: forwardDownStair, 9: adjust

---

## API & Protocols

| Компонент | Описание |
|-----------|----------|
| **CycloneDDS** | DDS-реализация, версия 0.10.x для Foxy |
| **unitree_go** | ROS2-пакет с msg для low-level (LowCmd, LowState) |
| **unitree_api** | ROS2-пакет с msg для sport mode (Request, SportModeState) |
| **SportClient** | Класс для формирования sportmode-запросов (Euler, Move, Stand) |
| **RMW** | `rmw_cyclonedds_cpp` |

### Low-level MotorCmd

```cpp
uint8 mode;   // Foc mode -> 0x01, stop -> 0x00
float q;      // Target position (rad)
float dq;     // Target velocity (rad/s)
float tau;    // Target torque (N·m)
float kp, kd;
```

### Документация API

- [Sports Services](https://support.unitree.com/home/en/developer/sports_services)
- [Basic Services](https://support.unitree.com/home/en/developer/Basic_services)
- [Get remote control status](https://support.unitree.com/home/en/developer/Get_remote_control_status)

---

## Integration Notes (SAI-AUROSY)

При разработке адаптера Unitree:

1. **unitree_ros2** — основной путь интеграции для Go2/B2/H1: ROS2-топики, DDS, совместимость с платформой.
2. **unitree_sdk2** — для нативного C++ без ROS2 или для моделей G1, A2 и др.
3. **unitree_ros** — для legacy ROS1 и симуляции в Gazebo; для реальных роботов нужен unitree_ros_to_real.
4. **Сеть:** Ethernet 192.168.123.x, настройка CYCLONEDDS_URI с указанием интерфейса.
5. **Sport mode vs Low-level:** Sport mode — high-level (ходьба, позы); Low-level — прямой контроль моторов.
6. **Модели:** Go2/B2/H1 — полная поддержка в ROS2; G1, A2 — примеры в SDK2; Z1 — отдельная документация [dev-z1.unitree.com](https://dev-z1.unitree.com).

---

## SAI-AUROSY: Спецификация протоколов (Go2 Adapter)

Адаптер Unitree Go2 использует **unitree_ros2** (ROS2) или **unitree_sdk2_python** (DDS). Публикует телеметрию в NATS, подписывается на команды.

### ROS2 топики (телеметрия)

| Топик | Тип | Описание |
|-------|-----|----------|
| `sportmodestate` | SportModeState | Позиция, скорость, gait, IMU |
| `lowstate` | LowState | MotorState[20], IMU, BMS |

### ROS2 топики (команды)

| Топик | Тип | Описание |
|-------|-----|----------|
| `/api/sport/request` | unitree_api::msg::Request | Sport mode команды (Damp, BalanceStand, Move) |

### Маппинг команд SAI-AUROSY → Unitree

| SAI-AUROSY | Unitree Sport API |
|------------|-------------------|
| `safe_stop` | Damp (mode 7) — отключение крутящего момента |
| `stand_mode` | BalanceStand |
| `walk_mode` | Move(0, 0, 0) |
| `zero_mode` | StandDown |
| `cmd_vel` | Move(linear_x, linear_y, angular_z) |
| `release_control` | Логирование; оператор берёт джойстик |

### Маппинг телеметрии → SAI-AUROSY

| Поле SAI-AUROSY | Источник |
|-----------------|----------|
| `online` | Heartbeat по sportmodestate/lowstate (timeout 2s) |
| `actuator_status` | enabled / unknown |
| `imu` | LowState.imu_state или SportModeState.imu_state |
| `joint_states` | LowState.motor_state[] (q, dq, tau_est) |
| `current_task` | Эвристика: safe_stop→idle, stand_mode→stand, walk_mode→walk, cmd_vel→walk |

### Режимы работы адаптера

- **UNITREE_MOCK=1** — без ROS2/SDK, публикация телеметрии каждую секунду
- **unitree_ros2** — при наличии unitree_go, unitree_api (source cyclonedds_ws/install/setup.bash)
- **unitree_sdk2_python** — альтернатива для команд (Damp, BalanceStand, Move)

### Требования для подключения

- ROS2 Humble (для unitree_ros2) или unitree_sdk2_python
- Сеть: `source ~/unitree_ros2/setup.sh` или `setup_local.sh` для loopback
- NATS: `NATS_URL`, `ROBOT_ID=go2-001`

---

## Supported Models

| Модель | unitree_sdk2 | unitree_ros | unitree_ros2 |
|--------|--------------|-------------|--------------|
| **Go2** | ✓ | ✓ (description, sim) | ✓ |
| **Go2W** | ✓ | ✓ | — |
| **B2** | ✓ | ✓ | ✓ |
| **B2W** | ✓ | ✓ | — |
| **H1** | ✓ | ✓ | ✓ |
| **H1-2** | ✓ | ✓ | ✓ |
| **G1** | ✓ | ✓ | ✓ (low-level) |
| **A1, A2** | ✓ (A2) | ✓ | — |
| **B1** | — | ✓ | — |
| **Z1** | — | ✓ (z1_controller) | — |
| **R1, R1 Air** | — | ✓ | — |
| **Laikago, Aliengo** | — | ✓ | — |

---

## Related Documents

- [Adapter Layer](../architecture/adapter-layer.md)
- [Multi-Robot Architecture](../architecture/multi-robot-architecture.md)
- [Agibot](agibot.md)
