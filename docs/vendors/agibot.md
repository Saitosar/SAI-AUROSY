# AGIBOT

## Overview

AGIBOT (AGIBOT Innovation Shanghai) — производитель open-source гуманоидных роботов. Робот **X1** — основная модель с полной документацией, inference pipeline и middleware runtime. См. [AGIBOT X1 Development Guide](https://www.agibot.com/DOCS/OS/X1-PDG).

---

## Ключевые ресурсы

| Ресурс | URL | Назначение |
|--------|-----|------------|
| **X1 Development Guide** | https://www.agibot.com/DOCS/OS/X1-PDG | Developer manual: архитектура, актуаторы, deploy, IMU |
| **X1 Infer** | https://github.com/AGIBOTTech/AGIBOT_x1_infer | Inference: модели, драйверы, симуляция |
| **X1 Train** | https://github.com/AGIBOTTech/AGIBOT_x1_train | Training code |
| **X1 Hardware** | https://github.com/AGIBOTTech/AGIBOT_x1_hardware | BOM, STEP, SolidWorks, SOP |
| **AimRT** | https://github.com/AimRT/AimRT | Robotics middleware runtime |
| **AimRT Docs** | https://aimrt.org | Документация AimRT |

---

## X1: Архитектура и железо

### Структура робота

- **29 суставов:** R86-2×9, R86-3×6, R52×10, L28×4
- **2 грейпера**
- **Голова:** расширяемая, 3 DOF

### Актуаторы

| Тип | Количество | Примечания |
|-----|------------|------------|
| R86-2 | 9 | PowerFlow R series |
| R86-3 | 6 | PowerFlow R series |
| R52 | 10 | PowerFlow R series |
| L28 | 4 | Отличается от R (4pin USB, авто zero) |

**Управление актуаторами:** REF-CLI (Windows), USB-C. Режим по умолчанию: `mode 6 = MIT hybrid control`.

### Сенсоры

- **IMU:** YIS320 (см. BOM)
- **Подключение:** только IMU нижних конечностей через DCU
- **Связь:** Serial, baud rate 921600

### LED статусы актуаторов

| LED | Значение |
|-----|----------|
| Зелёный мигает | Disabled |
| Зелёный горит | Enabled |
| Красный горит | Error (L28: синий мигает) |
| Синий горит | Calibration |

---

## X1 Product Development Guide (PDG)

**Источник:** [AGIBOT X1 Development Guide](https://www.agibot.com/DOCS/OS/X1-PDG)

### Deploy Process (кратко)

1. **4.1** Actuator Status Confirmation — REF-CLI, Enable/Disable, Set ID, Zero calibration
2. **4.2** Actuator desktop joint test — `xyber_dcu_test`, Ubuntu 22.04 RT, X86
3. **4.3** Assemble of X1 — SOP, видео
4. **4.4** IMU debugging
5. **4.5** Hardware deployment — проверка short-circuit, питание
6. **4.6** Zero calibration
7. **4.7** Software deployment — см. X1 Infer

### Команды REF-CLI (примеры)

```text
ref0.motor.request_state(0)   # disable
ref0.motor.request_state(1) # enable
ref0.can_node_id = 8         # CAN ID
ref0.motor.apply_user_offset()  # zero position
ref0.save_config()          # сохранить (только в disabled)
```

---

## AGIBOT X1 Infer

**Репозиторий:** [AGIBOT_x1_infer](https://github.com/AGIBOTTech/AGIBOT_x1_infer)

**Что это:** Inference-модуль для X1: model inference, platform driver, software simulation. Построен на **AimRT** как middleware, использует reinforcement learning для locomotion control.

### Структура репозитория

```text
src/
├── assistant         # ROS2 simulation, примеры
├── install           # конфигурация
├── module            # модули
├── pkg               # deployment
└── protocols         # протоколы
```

### Зависимости

- GCC-13
- CMake ≥ 3.26
- ONNX Runtime
- ROS2 Humble
- Linux realtime kernel (для реального робота)

### Запуск

| Режим | Команда |
|-------|---------|
| Симуляция | `cd build/ && ./run_sim.sh` |
| Реальный робот | `cd build/ && ./run.sh` |

**Управление:** Joystick (см. Joystick Control Module в doc).

---

## AimRT

**Репозиторий:** [AimRT](https://github.com/AimRT/AimRT) | **Документация:** [aimrt.org](https://aimrt.org)

**Что это:** High-performance runtime framework для робототехники. Modern C++, лёгкий, удобный для деплоя.

### Возможности

- Управление ресурсами и контролем
- Асинхронное программирование
- Конфигурация деплоя
- Интеграция: robot end-side, edge, cloud
- Совместимость с ROS2, HTTP, gRPC
- Plugin-интерфейс

### Схема (концептуально)

```text
robot modules
     ↓
  AimRT runtime
     ↓
message bus / communication
     ↓
robot hardware
```

---

## API & SDK

| Компонент | Описание |
|-----------|----------|
| REF-CLI | Конфигурация актуаторов (Windows) |
| AimRT | Runtime, message bus, модули |
| X1 Infer | Inference API, протоколы в `src/protocols` |
| ROS2 | Симуляция, assistant |

Детали — в [X1 Development Guide](https://www.agibot.com/DOCS/OS/X1-PDG) и [X1 Infer doc](https://github.com/AGIBOTTech/AGIBOT_x1_infer/tree/main/doc).

---

## Integration Notes (SAI-AUROSY)

При разработке адаптера AGIBOT:

1. **AimRT** — основная точка интеграции: message bus, runtime, модули.
2. **X1 Infer** — `src/protocols`, `src/module` для понимания команд и данных.
3. **PDG** — REF-CLI команды, CAN ID, IMU, ограничения железа.
4. **Режимы работы:** симуляция (run_sim) vs реальный робот (run) — разные пути деплоя.
5. **Актуаторы:** R86-2/3, R52, L28 — разные типы, L28 имеет отличия.

---

## SAI-AUROSY: Спецификация протоколов (X1 Infer)

X1 Infer использует **ROS2 Plugin** AimRT. Топики публикуются в ROS2, что позволяет SAI-AUROSY Adapter подключаться без модификации X1 Infer.

### ROS2 топики (телеметрия)

| Топик | Тип | Частота | Описание |
|-------|-----|---------|----------|
| `/joint_states` | `sensor_msgs::msg::JointState` | 1000 Hz | Позиции, скорости, усилия суставов |
| `/imu/data` | `sensor_msgs::msg::Imu` | 1000 Hz | IMU нижних конечностей (ориентация, угловая скорость) |

### ROS2 топики (команды)

| Топик | Тип | Описание |
|-------|-----|----------|
| `/start_control` | `std_msgs::msg::Empty` | Переход в **idle** (safe stop). Без крутящего момента. |
| `/zero_mode` | `std_msgs::msg::Empty` | Переход в zero — суставы в нулевую позицию |
| `/stand_mode` | `std_msgs::msg::Empty` | Стоячая поза |
| `/walk_mode` | `std_msgs::msg::Empty` | Режим ходьбы |
| `/cmd_vel` | `geometry_msgs::msg::Twist` | Команда движения (линейная/угловая скорость) |
| `/joint_cmd` | `my_ros2_proto::msg::JointCommand` | Прямое управление суставами (hybrid control) |

### Модули X1 Infer

| Модуль | Назначение |
|--------|------------|
| `DcuDriverModule` | Публикует `/joint_states`, `/imu/data`; подписан на `/joint_cmd` |
| `JoyStickModule` | Публикует команды перехода состояний по джойстику |
| `ControlModule` (RL) | Подписан на `/start_control`, `/zero_mode`, `/stand_mode`, `/walk_mode`, `/cmd_vel`, `/imu/data`, `/joint_states`; публикует `/joint_cmd` |
| `sim_module` | Симуляция в ROS2 |

### Safe Stop (SAI-AUROSY)

Для команды `safe_stop` адаптер публикует `std_msgs::msg::Empty` в топик `/start_control`. RL Control Module переводит робота в состояние **idle** (без крутящего момента).

### Маппинг телеметрии → SAI-AUROSY

| Поле SAI-AUROSY | Источник |
|-----------------|----------|
| `online` | Наличие сообщений от `/joint_states` (heartbeat) |
| `actuator_status` | Выводится из `joint_states.effort` или статуса подключения |
| `imu` | `/imu/data` (orientation, angular_velocity) |
| `current_task` | Состояние RL: idle / keep / zero / stand / walk_leg / walk_leg_arm |

**Примечание:** Точное состояние (current_task) не публикуется в ROS2. Для MVP можно определять по активности `/joint_cmd` или использовать заглушку.

### Требования для подключения

- ROS2 Humble
- `source /opt/ros/humble/setup.bash`
- Для X1 Infer: `source ./install/ros2_setup.sh` (из build-директории)
- Сеть: адаптер и X1 Infer должны быть в одной ROS2 domain (по умолчанию domain 0)

---

## Supported Models

| Модель | Документация | Infer | Hardware |
|--------|--------------|-------|----------|
| **AGIBOT X1** | [PDG](https://www.agibot.com/DOCS/OS/X1-PDG) | [x1_infer](https://github.com/AGIBOTTech/AGIBOT_x1_infer) | [x1_hardware](https://github.com/AGIBOTTech/AGIBOT_x1_hardware) |

---

## Related Documents

- [Adapter Layer](../architecture/adapter-layer.md)
- [Multi-Robot Architecture](../architecture/multi-robot-architecture.md)
- [Unitree](unitree.md)
