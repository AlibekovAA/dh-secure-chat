# dh-secure-chat

> End-to-end зашифрованный real-time мессенджер 1-на-1 с использованием Diffie-Hellman для установления сессионного ключа и симметричного шифрования на клиенте.

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://golang.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-blue)](https://www.typescriptlang.org/)
[![React](https://img.shields.io/badge/React-18.3-blue)](https://react.dev/)

---

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Архитектура](#архитектура)
- [Основные возможности](#основные-возможности)
- [Технический стек](#технический-стек)
- [Криптография и безопасность](#криптография-и-безопасность)
- [Развёртывание](#развёртывание)
- [API](#api)
- [Метрики и мониторинг](#метрики-и-мониторинг)
- [Тестирование](#тестирование)
- [Планы развития](#планы-развития)

---

## Быстрый старт

```bash
# 1. Клонируйте репозиторий
git clone https://github.com/AlibekovAA/dh-secure-chat.git
cd dh-secure-chat

# 2. Настройте переменные окружения
cp infra/env.example .env
# Отредактируйте .env файл

# 3. Запустите проект (режим разработки)
make develop-up-build

# 4. Откройте в браузере
# http://localhost
```

**Режимы запуска:**

- `make develop-up-build` — минимальный набор (без мониторинга)
- `make prod-up-build` — полный стек (с Prometheus и Grafana)

---

## Архитектура

### Общая архитектура системы

```mermaid
graph TB
    subgraph "Client"
        FE[React Frontend<br/>TypeScript + Tailwind]
    end

    subgraph "Reverse Proxy"
        NGINX[Nginx<br/>HTTPS Termination]
    end

    subgraph "Backend Services"
        AUTH[Auth Service<br/>Go + JWT]
        CHAT[Chat Service<br/>Go + WebSocket]
        ID[Identity Service<br/>Go]
    end

    subgraph "Database"
        PG[(PostgreSQL<br/>Users & Keys)]
    end

    subgraph "Monitoring"
        PROM[Prometheus]
        GRAF[Grafana]
    end

    FE -->|HTTPS| NGINX
    NGINX -->|/api/auth| AUTH
    NGINX -->|/api/chat<br/>/ws/| CHAT
    NGINX -->|/api/identity| ID
    AUTH --> PG
    CHAT --> PG
    ID --> PG
    PROM -->|Scrape| AUTH
    PROM -->|Scrape| CHAT
    GRAF --> PROM
```

### Поток аутентификации

```mermaid
sequenceDiagram
    participant C as Client
    participant N as Nginx
    participant A as Auth Service
    participant DB as PostgreSQL

    C->>N: POST /api/auth/register
    N->>A: Forward request
    A->>DB: Create user + identity key
    DB-->>A: User created
    A-->>C: JWT + Refresh Token (cookie)

    C->>N: POST /api/auth/login
    N->>A: Forward request
    A->>DB: Verify credentials
    DB-->>A: User verified
    A-->>C: JWT + Refresh Token (cookie)

    Note over C: JWT в памяти,<br/>Refresh Token в httpOnly cookie
```

### Поток установки защищённой сессии

```mermaid
sequenceDiagram
    participant A as Client A
    participant WS as WebSocket Hub
    participant B as Client B

    A->>WS: Connect + JWT auth
    B->>WS: Connect + JWT auth

    A->>A: Generate ephemeral key pair
    A->>A: Sign ephemeral pub key<br/>with identity private key
    A->>WS: ephemeral_key + signature
    WS->>B: Forward ephemeral_key + signature

    B->>B: Verify signature
    B->>B: Generate ephemeral key pair
    B->>B: Sign ephemeral pub key
    B->>WS: ephemeral_key + signature + ack
    WS->>A: Forward ephemeral_key + signature + ack

    A->>A: Verify signature
    A->>A: Derive session key (ECDH + HKDF)
    B->>B: Derive session key (ECDH + HKDF)

    A->>WS: session_established
    WS->>B: Forward session_established
    B->>WS: ack
    WS->>A: Forward ack

    Note over A,B: Защищённая сессия установлена<br/>Можно отправлять зашифрованные сообщения
```

### Поток отправки сообщения

```mermaid
sequenceDiagram
    participant A as Client A
    participant WS as WebSocket Hub
    participant B as Client B

    A->>A: Encrypt message<br/>(AES-GCM + session key)
    A->>WS: message {ciphertext, nonce, metadata}
    WS->>WS: Validate + route
    WS->>B: Forward message

    B->>B: Decrypt message<br/>(AES-GCM + session key)
    B->>WS: ack {message_id}
    WS->>A: Forward ack

    Note over B: Message visible in viewport
    B->>WS: message_read {message_id}
    WS->>A: Forward message_read
```

### Поток передачи файла

```mermaid
sequenceDiagram
    participant A as Client A
    participant WS as WebSocket Hub
    participant B as Client B

    A->>A: Encrypt file chunk 1<br/>(1MB chunks)
    A->>WS: file_start {metadata, access_mode}
    WS->>B: Forward file_start

    loop For each chunk
        A->>A: Encrypt chunk
        A->>WS: file_chunk {chunk_id, ciphertext, nonce}
        WS->>B: Forward file_chunk
        B->>B: Decrypt chunk
    end

    A->>WS: file_complete {file_id}
    WS->>B: Forward file_complete
    B->>B: Reconstruct file
    B->>WS: ack {file_id}
    WS->>A: Forward ack
```

---

## Основные возможности

### Безопасность

- **E2E шифрование** — все сообщения шифруются на клиенте (AES-GCM)
- **Diffie-Hellman** — отдельный сессионный ключ для каждого чата
- **Подпись ключей** — защита от MITM-атак через подпись ephemeral-ключей
- **Fingerprint верификация** — визуальное сравнение ключей (TOFU)
- **JWT с инвалидацией** — возможность отзыва токенов через JTI

### Чат

- **Real-time обмен** — WebSocket для мгновенной доставки
- **1-на-1 диалоги** — только приватные чаты
- **Статусы доставки** — `sending` → `delivered` → `read`
- **Typing indicators** — индикатор набора текста
- **Реакции на сообщения** — эмодзи-реакции

### Файлы и медиа

- **Файлы до 50MB** — передача по частям (1MB chunks)
- **Голосовые сообщения** — запись через MediaRecorder API (до 5 мин, 10MB)
- **Видео сообщения** — запись с камеры (до 50MB)
- **Режимы доступа** — `both`, `view_only`, `download_only`
- **Защита просмотра** — Canvas рендеринг с водяным знаком для `view_only`
- **Видео-кружки** — круглые превью с воспроизведением в чате

### Надёжность

- **Circuit Breaker** — защита БД от перегрузки
- **Idempotency** — предотвращение дублирования сообщений
- **Graceful degradation** — продолжение работы при некритичных ошибках
- **Метрики Prometheus** — полный мониторинг системы

---

## Технический стек

### Frontend

- **React 18** + **TypeScript** — современный SPA
- **Vite** — сборщик и dev-сервер
- **Tailwind CSS** — стилизация
- **Web Crypto API** — криптография на клиенте
- **WebSocket** — real-time коммуникация
- **MediaRecorder API** — запись аудио/видео
- **ESLint** + **Prettier** — линтинг и форматирование кода
- **Path Aliases** (`@/`) — удобные импорты

### Backend

- **Go 1.24** — высокопроизводительный backend
- **net/http** — стандартная библиотека (без фреймворков)
- **PostgreSQL** — хранение пользователей и ключей
- **JWT** — аутентификация
- **WebSocket Hub** — управление соединениями

### Инфраструктура

- **Docker** + **docker-compose** — контейнеризация
- **Nginx** — reverse proxy и HTTPS
- **Prometheus** — сбор метрик
- **Grafana** — визуализация метрик

---

## Криптография и безопасность

### Ключевая архитектура

```mermaid
graph TB
    subgraph "Identity Keys"
        IKP[Identity Key Pair<br/>ECDSA P-256]
        IPUB[Public Key → Server]
        IPRIV[Private Key → Client Only]
        IKP --> IPUB
        IKP --> IPRIV
    end

    subgraph "Session Establishment"
        EKP1[Ephemeral Key Pair A<br/>ECDH P-256]
        EKP2[Ephemeral Key Pair B<br/>ECDH P-256]
        SIG1[Sign with Identity A]
        SIG2[Sign with Identity B]
        EKP1 --> SIG1
        EKP2 --> SIG2
        SIG1 --> VERIFY[Verify Signatures]
        SIG2 --> VERIFY
        VERIFY --> DH[ECDH Key Exchange]
        DH --> KDF[HKDF-SHA256]
        KDF --> SK[Session Key<br/>AES-256-GCM]
    end

    subgraph "Message Encryption"
        MSG[Plaintext Message]
        SK --> ENC[AES-GCM Encrypt]
        MSG --> ENC
        ENC --> CT[Ciphertext + Nonce]
    end
```

### Процесс установки сессии

1. **Генерация identity-ключей** (при регистрации)
   - Клиент генерирует долгоживущую пару ECDSA P-256
   - Публичный ключ отправляется на сервер
   - Приватный ключ остаётся только на клиенте

2. **Генерация ephemeral-ключей** (для каждой сессии)
   - Каждый клиент генерирует временную пару ECDH P-256
   - Публичный ephemeral-ключ подписывается приватным identity-ключом

3. **Обмен ключами**
   - Клиенты обмениваются публичными ephemeral-ключами и подписями
   - Подписи проверяются перед использованием ключей
   - Используется acknowledge mechanism для подтверждения

4. **Выработка сессионного ключа**
   - Общий секрет вычисляется через ECDH
   - Применяется HKDF-SHA256 для получения симметричного ключа AES-256-GCM

### Шифрование сообщений

- **Алгоритм**: AES-256-GCM
- **Ключ**: сессионный ключ (256 бит)
- **Nonce**: генерируется для каждого сообщения
- **Аутентификация**: встроенная в GCM

**На сервер уходят только:**

- Ciphertext (зашифрованный текст)
- Nonce (случайное число)
- Метаданные (отправитель, получатель, ID)

**Сервер не может расшифровать сообщения.**

### Fingerprint верификация

- **Fingerprint** = SHA-256 от публичного identity-ключа
- **Визуализация** через эмодзи для удобного сравнения
- **TOFU** (Trust On First Use) — автоматическое сохранение при первом контакте
- **Блокировка** чата при изменении fingerprint у verified peer

---

## Развёртывание

### Режим DEVELOP (минимальный)

```bash
make develop-up           # Запуск
make develop-up-build     # Запуск с пересборкой
make develop-down         # Остановка
make develop-down-volumes  # Остановка + удаление volumes
```

**Сервисы:**

- Frontend (React)
- Auth Service
- Chat Service
- PostgreSQL
- Nginx

### Режим PROD (полный стек)

```bash
make prod-up           # Запуск
make prod-up-build     # Запуск с пересборкой
make prod-down         # Остановка
make prod-down-volumes # Остановка + удаление volumes
```

**Дополнительно:**

- Prometheus (порт 9090)
- Grafana (порт 3000, логин: `admin`/`admin`)

### Утилиты

```bash
make help                      # Список всех команд
make clean                     # Полная очистка Docker
make backend                   # Запуск backend локально
make frontend                  # Запуск frontend локально
make format        # Форматирование и линтинг (Go + TypeScript/React)
make backend-test  # Запуск всех тестов бэкенда
```

---

## API

### Auth Service

| Метод  | Endpoint             | Описание                          |
| ------ | -------------------- | --------------------------------- |
| `POST` | `/api/auth/register` | Регистрация пользователя          |
| `POST` | `/api/auth/login`    | Вход в систему                    |
| `POST` | `/api/auth/refresh`  | Обновление access token           |
| `POST` | `/api/auth/logout`   | Выход (инвалидация токенов)       |
| `POST` | `/api/auth/revoke`   | Инвалидация текущего access token |

### Chat Service (REST)

| Метод | Endpoint                       | Описание                          |
| ----- | ------------------------------ | --------------------------------- |
| `GET` | `/api/chat/me`                 | Информация о текущем пользователе |
| `GET` | `/api/chat/users?username=...` | Поиск пользователя по username    |

### Identity Service

| Метод | Endpoint                               | Описание                            |
| ----- | -------------------------------------- | ----------------------------------- |
| `GET` | `/api/identity/users/{id}/key`         | Получение публичного identity-ключа |
| `GET` | `/api/identity/users/{id}/fingerprint` | Получение fingerprint               |

### WebSocket

| Endpoint  | Описание                                              |
| --------- | ----------------------------------------------------- |
| `WS /ws/` | WebSocket подключение (JWT в первом сообщении `auth`) |

**Типы сообщений:**

- `auth` — аутентификация
- `ephemeral_key` — обмен ephemeral-ключами
- `session_established` — подтверждение установки сессии
- `message` — текстовое сообщение
- `file_start`, `file_chunk`, `file_complete` — передача файла
- `ack` — подтверждение получения
- `message_read` — сообщение прочитано
- `typing` — индикатор набора текста
- `reaction` — реакция на сообщение

---

## Метрики и мониторинг

Все метрики экспортируются в формате **Prometheus** через `GET /metrics` на каждом сервисе.

### Auth Service (`:8081/metrics`)

- **HTTP метрики**: `http_requests_total`, `http_request_duration_seconds`, `http_errors_total`
- **Token метрики**: `access_tokens_issued_total`, `access_tokens_revoked_total`, `refresh_tokens_issued_total`, `refresh_tokens_revoked_total`
- **JWT метрики**: `jwt_validations_total`, `jwt_validations_failed_total`
- **Domain ошибки**: `domain_errors_total`

### Chat Service (`:8082/metrics`)

- **HTTP метрики**: `http_requests_total`, `http_request_duration_seconds`, `http_errors_total`
- **WebSocket метрики**:
  - `chat_websocket_connections_active` — активные соединения
  - `chat_websocket_connections_rejected_total` — отклонённые соединения
  - `chat_websocket_messages_total` — сообщения по типам
  - `chat_websocket_errors_total` — ошибки по типам
  - `chat_websocket_disconnections_total` — отключения по причинам
  - `chat_websocket_dropped_messages_total` — потерянные сообщения
  - `chat_websocket_message_send_duration_seconds` — длительность отправки (p95, p99)
  - `chat_websocket_message_processing_duration_seconds` — длительность обработки
  - `chat_websocket_message_processor_queue_size` — размер очереди обработки
- **Database метрики**:
  - `db_pool_acquired_connections`, `db_pool_idle_connections`, `db_pool_max_connections`, `db_pool_total_connections`
  - `db_query_duration_seconds` — длительность запросов (p95, p99)
  - `db_query_errors_total` — ошибки запросов
- **Circuit Breaker**: `circuit_breaker_state` — состояние (0=closed, 1=open, 2=half-open)
- **File Transfer**:
  - `chat_websocket_files_total` — количество файлов
  - `chat_websocket_files_chunks_total` — количество чанков
  - `chat_websocket_file_transfer_failures_total` — ошибки передачи
- **Idempotency**: `chat_websocket_idempotency_duplicates_total` — дубликаты сообщений
- **Cache метрики**:
  - `chat_websocket_user_existence_cache_hits_total`, `chat_websocket_user_existence_cache_misses_total`
  - `chat_websocket_user_existence_cache_size` — размер кэша

### Grafana Dashboard

Автоматически загружается дашборд **"DH Secure Chat - Comprehensive Monitoring"** с панелями:

- **WebSocket**: активные соединения, отклонённые соединения, размер очереди, сообщения по типам, ошибки, отключения
- **HTTP**: rate запросов, длительность (p50, p95, p99), ошибки по статус-кодам
- **Database**: пул соединений, длительность запросов (p95, p99)
- **Circuit Breaker**: состояние (closed/open/half-open)
- **JWT и Token**: валидации, выдача и отзыв токенов
- **File Transfers**: количество файлов, чанков, ошибки передачи
- **Idempotency**: дубликаты сообщений
- **Domain Errors**: ошибки по категориям и кодам

**Доступ:** http://localhost:3000 (логин: `admin` / пароль: `admin`)

---

## Тестирование

Тесты бэкенда находятся в `backend/test/`: пакеты `auth` (auth service, refresh token, validation, HTTP-хендлеры) и `chat` (chat service).

```bash
make backend-test   # Запуск всех тестов
```

---

## Планы развития

- [ ] Fallback библиотека для Web Crypto API

---
