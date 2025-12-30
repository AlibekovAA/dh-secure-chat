## dh-secure-chat

End‑to‑end зашифрованный real‑time мессенджер 1‑на‑1 на Go (backend) и TypeScript/React (frontend) с использованием Diffie‑Hellman для установления сессионного ключа и симметричного шифрования на клиенте. Репозиторий: [`dh-secure-chat`](https://github.com/AlibekovAA/dh-secure-chat).

### Цели проекта

- **Real‑time чат 1‑на‑1** без хранения сообщений на сервере.
- **E2E‑шифрование**: все сообщения шифруются и расшифровываются только на клиентах.
- **DH‑обмен ключами** для установления отдельного сессионного ключа на каждый чат.
- **Простое развёртывание** через Docker и `docker-compose` на один сервер.

---

## Основной стек

- **Frontend**

  - React + TypeScript (SPA).
  - Tailwind CSS.
  - WebSocket‑клиент для real‑time сообщений.
  - Web Crypto API для работы с ключами и симметричным шифрованием.
  - MediaRecorder API для записи голосовых и видео сообщений.
  - getUserMedia API для доступа к микрофону и камере.

- **Backend**

  - Go, стандартная библиотека `net/http`.
  - Отдельный **auth‑service** (HTTP + JWT).
  - Сервис чата с WebSocket‑hub'ом.
  - PostgreSQL (только пользователи и публичные identity‑ключи).

- **Инфраструктура**
  - Docker, `docker-compose`.
  - Reverse proxy (Nginx) для HTTPS и роутинга HTTP/WS.
  - Prometheus для сбора метрик.
  - Grafana для визуализации метрик.

---

## Развёртывание

Проект поддерживает два режима запуска через Makefile:

### Режим DEVELOP (минимальный)

Минимальный набор сервисов для разработки без мониторинга:

- Frontend (React)
- Backend (auth-service + chat-service)
- PostgreSQL
- Nginx (reverse proxy)

**Команды:**

```bash
make develop-up           # Запуск контейнеров
make develop-up-build     # Запуск с пересборкой
make develop-down         # Остановка контейнеров
make develop-down-volumes # Остановка с удалением volumes
make develop-restart      # Перезапуск
make develop-reup         # Остановка + пересборка + запуск
make develop-rebuild      # Полная пересборка без кеша
```

### Режим PROD (полный стек)

Полный набор сервисов с мониторингом:

- Все сервисы из режима develop
- Prometheus (сбор метрик)
- Grafana (визуализация метрик)

**Команды:**

```bash
make prod-up           # Запуск контейнеров
make prod-up-build     # Запуск с пересборкой
make prod-down         # Остановка контейнеров
make prod-down-volumes # Остановка с удалением volumes
make prod-restart      # Перезапуск
make prod-reup         # Остановка + пересборка + запуск
make prod-rebuild      # Полная пересборка без кеша
```

### Быстрый старт

1. Скопируйте `infra/env.example` в `.env` и настройте переменные окружения
2. Для разработки: `make develop-up-build`
3. Для продакшена с мониторингом: `make prod-up-build`

После запуска приложение будет доступно на `http://localhost` (через Nginx).

### Утилиты

```bash
make help                      # Список всех доступных команд
make clean                     # Удаление всех Docker контейнеров, образов, volumes и файлов покрытия
make backend                   # Запуск backend локально без Docker
make frontend                  # Запуск frontend локально без Docker
make format                    # Форматирование Go кода (go-vet, go-fmt, go-lint)
make go-test                   # Запуск всех тестов backend
make go-test-auth              # Запуск тестов auth-service
make go-test-auth-coverage     # Запуск тестов auth-service с HTML отчётом покрытия
```

### Тестирование

Проект включает комплексные unit-тесты для auth-service с покрытием 97.8% кода сервиса.

**Структура тестов:**

- Тесты находятся в `backend/test/auth/`
- Тесты разбиты по компонентам:
  - `auth_service_register_test.go` - тесты регистрации
  - `auth_service_login_test.go` - тесты входа
  - `auth_service_refresh_test.go` - тесты обновления токенов
  - `auth_service_revoke_test.go` - тесты отзыва токенов
  - `auth_service_error_test.go` - тесты обработки ошибок
  - `token_issuer_test.go` - тесты выдачи JWT токенов
  - `refresh_token_rotator_test.go` - тесты ротации refresh токенов
  - `validation_test.go` - тесты валидации учётных данных
  - `cleanup_test.go` - тесты очистки истёкших токенов
  - `mocks.go` - общие моки для тестов

**Запуск тестов:**

```bash
make go-test-auth              # Запуск всех тестов auth-service
make go-test-auth-coverage     # Запуск с HTML отчётом покрытия (coverage.html)
```

**Покрытие кода:**

- Все тесты обеспечивают 97.8% покрытие кода auth-service
- Отчёты покрытия генерируются в формате HTML и текстовом формате
- HTML отчёт сохраняется в `backend/coverage.html`

---

## Функциональные требования

### Пользователи и аутентификация

- **Регистрация**

  - username + пароль.
  - На клиенте генерируется долгоживущая **identity‑ключевая пара**.
  - На сервер уходит только публичный `identity`‑ключ, приватный ключ всегда остаётся на клиенте.

- **Логин**

  - Аутентификация по `username + password`.
  - Auth‑service выдаёт **JWT** (TTL 15 минут) с уникальным **JTI** (JWT ID) и **refresh token** (httpOnly cookie, TTL 7 дней).
  - Refresh token используется для автоматического обновления access token.
  - JWT используется для REST‑запросов (Authorization: Bearer …) и авторизации WebSocket‑подключений.
  - JTI позволяет инвалидировать access tokens до истечения их срока действия.

- **Инвалидация токенов**

  - `POST /api/auth/revoke` — инвалидация текущего access token (требует Authorization header).
  - При logout инвалидируются как access token (если передан), так и refresh token.
  - Инвалидированные токены проверяются при каждой валидации JWT.
  - Автоматическая очистка expired revoked tokens.

- **Профиль и поиск**
  - `GET /api/chat/me` — информация о текущем пользователе.
  - Поиск пользователя по `username` для старта диалога.
  - `GET /api/identity/users/{id}/key` — получение публичного identity‑ключа.
  - `GET /api/identity/users/{id}/fingerprint` — получение fingerprint для верификации.

### Real‑time чат (только при одновременном онлайне)

- Только **1‑на‑1** диалоги.
- Чат‑сессия существует, пока **оба участника подключены по WebSocket**.
- При отключении любого участника:

  - чат‑сессия завершается;
  - история сообщений больше не доступна (сообщения хранятся только в памяти клиента).

- **Сообщения**

  - отправляются и принимаются только через WebSocket;
  - не буферизуются и не доставляются позже;
  - не сохраняются в БД, не логируются и не кэшируются на сервере;
  - на сервере виден только шифротекст + nonce + минимальные метаданные (отправитель/получатель, technical ids);
  - **Статусы доставки/прочтения**:
    - для текстовых сообщений: `sending` → `delivered` (после получения `ack`) → `read` (после получения `message_read`);
    - для файлов: `sending` → `delivered` (после получения `ack` на `file_complete`) → `read` (после просмотра файла);
    - для голосовых сообщений: `sending` → `delivered` (после получения `ack` на `file_complete`) → `read` (после прослушивания 50%+ длительности);
    - статусы отображаются только для собственных сообщений;
    - `message_read` отправляется автоматически при видимости сообщения в viewport (для текстовых), при просмотре файла или при прослушивании голосового сообщения.

- **Отправка файлов**

  - поддержка отправки файлов до 50MB;
  - файлы шифруются на клиенте перед отправкой;
  - разбиение на чанки по 1MB для передачи через WebSocket;
  - каждый чанк шифруется отдельно с использованием сессионного ключа;
  - файлы передаются только при активной защищённой сессии;
  - файлы не сохраняются на сервере, передаются напрямую между клиентами;
  - **Режимы доступа к файлам**:
    - `both` — скачивание и просмотр (по умолчанию);
    - `view_only` — только просмотр (без возможности скачивания);
    - `download_only` — только скачивание (без предпросмотра);
  - отправитель всегда имеет полный доступ (и просмотр, и скачивание) к своим файлам;
  - выбор режима доступа при отправке файла через модальное окно;
  - модальное окно предпросмотра файлов с поддержкой изображений, PDF, текстовых файлов и видео;
  - для режима `view_only` применяются дополнительные меры защиты от копирования (Canvas рендеринг с водяным знаком, блокировка контекстного меню, блокировка клавиатурных сокращений, блокировка выделения текста и drag-and-drop);
  - **Видео-кружки**:
    - видео файлы (MP4, WebM, OGG, QuickTime, AVI, MKV) отображаются как круглые превью в чате;
    - автоматическая генерация превью из первого кадра видео на клиенте;
    - кэширование превью в localStorage для быстрой загрузки;
    - lazy loading превью (генерация только при появлении в viewport);
    - при клике на видео-кружку видео воспроизводится прямо в кружке без открытия модального окна;
    - автоматическое определение продолжительности видео для корректного отображения;
    - поддержка всех стандартных форматов видео через браузерный видеоплеер.

- **Голосовые сообщения**

  - запись голосовых сообщений через MediaRecorder API;
  - поддержка форматов: WebM (Opus), OGG (Opus), MP4, MPEG;
  - максимальная длительность записи: 5 минут;
  - максимальный размер: 10MB;
  - автоматическое определение голосовых сообщений по MIME-типу;
  - отображение с проигрывателем и прогресс-баром;
  - голосовые сообщения передаются как зашифрованные файлы через тот же механизм;
  - требуется поддержка браузером MediaRecorder API и getUserMedia API.

- **Видео сообщения**
  - запись видео сообщений через MediaRecorder API с использованием камеры и микрофона;
  - поддержка форматов: WebM (VP9/VP8 + Opus), MP4;
  - автоматический выбор оптимального формата в зависимости от поддержки браузером;
  - максимальная длительность записи: без ограничений (ограничение только по размеру файла);
  - максимальный размер: 50MB;
  - модальное окно записи с предпросмотром видео в реальном времени;
  - отображение таймера записи во время съёмки;
  - возможность отмены записи без сохранения;
  - видео сообщения передаются как зашифрованные файлы через тот же механизм;
  - отображаются как видео-кружки с возможностью воспроизведения прямо в кружке;
  - требуется поддержка браузером MediaRecorder API и getUserMedia API для доступа к камере и микрофону.

### UI‑состояния

- `peer offline` — собеседник не в сети, чат недоступен.
- `establishing secure session` — идёт DH‑обмен и установка сессионного ключа.
- `secure session active` — установлена защищённая сессия, можно переписываться.
- `peer disconnected` — один из участников отключился, сессия завершена, история очищена из UI.

---

## Криптография и безопасность

### Identity‑keys

- На клиенте генерируется долгоживущая ключевая пара `identity` при регистрации.
- Публичный ключ отправляется на сервер и хранится в БД.
- Приватный ключ не покидает устройство.
- **Примечание**: Identity-ключи используются для идентификации пользователя. Для установки сессионного ключа используются ephemeral-ключи (см. ниже).

### Сессионный ключ

- Для каждого чата создаётся отдельная **Diffie‑Hellman‑сессия**:
  - клиенты генерируют ephemeral‑ключи;
  - ephemeral‑ключи подписываются приватным identity‑ключом для защиты от MITM‑атак;
  - обмениваются публичными ephemeral‑ключами и подписями через служебные WebSocket‑сообщения;
  - подписи проверяются перед использованием ключей;
  - общий секрет прогоняется через KDF для получения симметричного ключа шифрования сообщений;
  - используется acknowledge mechanism для подтверждения получения критичных сообщений (ephemeral keys).

### Шифрование сообщений

- Все сообщения, файлы и голосовые сообщения шифруются на клиенте.
- Используется AES-GCM для симметричного шифрования.
- На сервер уходят только:
  - ciphertext;
  - nonce/iv;
  - минимальные служебные поля (id чата, id отправителя/получателя, MIME-тип для файлов).
- Сервер **никогда не может расшифровать** сообщение, файл или голосовое сообщение.
- Файлы и голосовые сообщения разбиваются на чанки и шифруются по частям для эффективной передачи.

### Fingerprint‑верификация

- Для каждого собеседника отображается **fingerprint** его публичного identity‑ключа (SHA‑256).
- Визуальное представление через эмодзи для упрощения сравнения по телефону.
- Автоматическое сохранение fingerprint при первом контакте (TOFU).
- Блокировка чата при изменении fingerprint у verified peer.
- История изменений fingerprint для детекции атак.
- Экспорт/импорт verified peers для многоустройственности.

---

## Нефункциональные требования

- **Отсутствие персистентности сообщений**

  - БД хранит только пользователей и их публичные identity‑ключи.
  - Сообщения никогда не пишутся в БД, не логируются и не кэшируются на сервере.

- **Безопасность**

  - HTTPS на уровне reverse‑proxy (Nginx).
  - Безопасное хранение паролей (bcrypt).
  - Refresh tokens в httpOnly cookies с `SameSite=Strict`.
  - Access tokens хранятся только в памяти клиента.
  - Лимит активных refresh tokens на пользователя (5).
  - Автоматическая очистка expired refresh tokens и revoked tokens.
  - JTI (JWT ID) в каждом access token для возможности инвалидации.
  - Проверка revoked tokens при каждой валидации JWT (в REST и WebSocket).
  - Trace ID для корреляции логов между сервисами (генерируется автоматически, передаётся в заголовке `X-Trace-ID`).
  - Rate limiting для защиты от брутфорса (5 req/s для login, 2 req/s для register, 100 req/s для остальных запросов).
  - **Circuit Breaker для БД**: все критические операции с базой данных в AuthService обёрнуты в circuit breaker для защиты от перегрузки БД (порог: 5 ошибок, таймаут: 5 секунд, сброс: 30 секунд).
  - Circuit breaker для защиты БД от перегрузки при обновлении `last_seen_at` в chat-service.
  - **Graceful Degradation для Identity Service**: при регистрации, если сохранение identity-ключа не удалось (некритичная ошибка), регистрация продолжается успешно, пользователь создаётся без identity-ключа.
  - Idempotency для предотвращения дублирования критичных операций (ephemeral keys, messages, file chunks).
  - Шифрование приватных identity‑ключей перед сохранением в IndexedDB (защита от XSS).
  - **Защита файлов в режиме "только просмотр"**:
    - Canvas рендеринг изображений с динамическим водяным знаком (время, пользователь);
    - блокировка контекстного меню (правый клик);
    - блокировка клавиатурных сокращений (F12, Ctrl+S, Ctrl+P, PrintScreen, DevTools);
    - блокировка выделения текста и drag-and-drop;
    - автоматическое закрытие окна предпросмотра при потере фокуса или переключении вкладки;
    - **Примечание**: полная защита от скриншотов на уровне ОС невозможна в веб-приложении, но применяются базовые меры для усложнения копирования.

- **Простое развёртывание**

  - Два режима запуска через Makefile:
    - **develop** — минимальный набор сервисов (frontend, backend, PostgreSQL, Nginx);
    - **prod** — полный стек с мониторингом (включая Prometheus и Grafana).
  - Все сервисы управляются через `docker-compose` и Makefile.

---

## Метрики

Все метрики экспортируются в формате **Prometheus** через HTTP‑эндпоинт `GET /metrics` на каждом сервисе. Метрики готовы для сбора Prometheus и визуализации в Grafana.

### Auth‑service

- **Endpoint**: `GET /metrics` на порту auth‑сервиса (`AUTH_HTTP_PORT`, по умолчанию `8081`).
- **Примеры**:

  - внутри контейнера: `curl -s localhost:8081/metrics`
  - с хоста (если порт проброшен): `curl -s http://localhost:8081/metrics`
  - `Invoke-RestMethod -Uri http://localhost:8081/metrics`

- **HTTP метрики**:

  - `auth_requests_total{method, path}` — общее число HTTP запросов (с labels по методу и пути);
  - `auth_requests_in_flight` — текущее число обрабатываемых запросов (Gauge);
  - `auth_request_duration_seconds{method, path, status}` — гистограмма длительности запросов (с labels по методу, пути и статусу).

- **Auth метрики**:

  - `refresh_tokens_issued_total` — количество выданных refresh tokens;
  - `refresh_tokens_used_total` — количество использованных refresh tokens;
  - `refresh_tokens_revoked_total` — количество отозванных refresh tokens;
  - `refresh_tokens_expired_total` — количество истёкших refresh tokens;
  - `refresh_tokens_cleanup_deleted_total` — количество удалённых expired refresh tokens при cleanup;
  - `access_tokens_issued_total` — количество выданных access tokens;
  - `access_tokens_revoked_total` — количество инвалидированных access tokens;
  - `revoked_tokens_cleanup_deleted_total` — количество удалённых expired revoked tokens при cleanup.

- **JWT метрики**:

  - `jwt_validations_total` — общее количество проверок JWT токенов;
  - `jwt_validations_failed_total` — количество неудачных проверок JWT;
  - `jwt_revoked_checks_total` — количество проверок revoked tokens.

- **Rate limiting метрики**:
  - `rate_limit_blocked_total{path, limiter_type}` — количество заблокированных запросов (с labels по пути и типу лимитера: `login`, `register`, `general`).

### Chat‑service

- **Endpoint**: `GET /metrics` на порту chat‑сервиса (`CHAT_HTTP_PORT`, по умолчанию `8082`).
- **Примеры**:

  - внутри контейнера: `curl -s localhost:8082/metrics`
  - с хоста (если порт проброшен): `curl -s http://localhost:8082/metrics`
  - `Invoke-RestMethod -Uri http://localhost:8082/metrics`

- **HTTP метрики**:

  - `chat_requests_total{method, path}` — общее число HTTP запросов (с labels по методу и пути);
  - `chat_requests_in_flight` — текущее число обрабатываемых запросов (Gauge);
  - `chat_request_duration_seconds{method, path, status}` — гистограмма длительности запросов (с labels по методу, пути и статусу).

- **WebSocket метрики**:

  - `chat_websocket_connections_active` — текущее число активных WebSocket подключений (Gauge);
  - `chat_websocket_connections_total` — общее число установленных WebSocket подключений;
  - `chat_websocket_disconnections_total{reason}` — количество отключений (с labels по причине: `read_error`, `read_error_unauthenticated`, `normal_close`, `unregister`);
  - `chat_websocket_errors_total{error_type}` — количество ошибок WebSocket по типам (invalid payload, validation failed и т.д.);
  - `chat_websocket_messages_total{message_type}` — количество сообщений WebSocket по типам (`message`, `ephemeral_key`, `session_established`, `ack`, `file_start`, `file_chunk`, `file_complete`, `typing`, `reaction`, `message_delete`, `message_edit`, `message_read`);
  - `chat_websocket_message_processing_duration_seconds{message_type}` — гистограмма времени обработки сообщений по типам;
  - `chat_websocket_message_processor_queue_size` — текущий размер очереди обработки сообщений (Gauge);
  - `chat_websocket_files_total` — количество отправленных файлов;
  - `chat_websocket_files_chunks_total` — количество переданных чанков файлов;
  - `chat_websocket_file_transfer_duration_seconds{status}` — гистограмма длительности передачи файлов (с labels по статусу: `success`, `failed`);
  - `chat_websocket_file_transfer_failures_total{reason}` — количество неудачных передач файлов (с labels по причине: `track_failed`, `complete_failed`, `timeout_or_disconnect`);
  - `chat_websocket_idempotency_duplicates_total{message_type}` — количество обнаруженных дубликатов сообщений по типам.

- **Database метрики**:

  - `db_pool_acquired_connections` — количество активных соединений с БД (Gauge);
  - `db_pool_idle_connections` — количество простаивающих соединений с БД (Gauge);
  - `db_pool_max_connections` — максимальное количество соединений в пуле (Gauge);
  - `db_pool_total_connections` — общее количество соединений в пуле (Gauge);
  - `db_query_duration_seconds{operation, table}` — гистограмма длительности запросов к БД (с labels по операции и таблице);
  - `db_query_errors_total{operation, table, error_type}` — количество ошибок запросов к БД (с labels по операции, таблице и типу ошибки).

- **Circuit breaker метрики**:
  - `circuit_breaker_state{name}` — состояние circuit breaker (0=closed, 1=open, Gauge);
  - `circuit_breaker_failures_total{name}` — количество failures в circuit breaker (используется для обновления `last_seen_at` и для всех операций БД в AuthService).

### Интеграция с Prometheus и Grafana

Prometheus и Grafana настроены и готовы к использованию. Метрики собираются автоматически с обоих сервисов и визуализируются в Grafana.

**Примечание**: Prometheus и Grafana доступны только в режиме **prod**. В режиме **develop** они не запускаются.

#### Запуск мониторинга

Для запуска с мониторингом используйте режим **prod**:

```bash
make prod-up
```

или с пересборкой:

```bash
make prod-up-build
```

После запуска:

- **Prometheus UI**: http://localhost:9090
- **Grafana UI**: http://localhost:3000

#### Доступ к Grafana

- **URL**: http://localhost:3000
- **Логин**: `admin`
- **Пароль**: `admin`

#### Конфигурация Prometheus

Конфигурация Prometheus находится в `infra/prometheus/prometheus.yml`:

- **Auth service**: `auth:8081/metrics`
- **Chat service**: `chat:8082/metrics`

Интервал сбора метрик: **15 секунд**

#### Дашборды Grafana

Автоматически загружается дашборд **"DH Secure Chat - Overview"** с визуализацией всех метрик:

- **WebSocket метрики**: активные соединения, сообщения, отключения
- **HTTP метрики**: запросы, длительность, ошибки (для auth и chat сервисов)
- **Database метрики**: пул соединений, длительность запросов, ошибки
- **Circuit Breaker**: состояние и количество failures
- **JWT метрики**: валидации, ошибки
- **Token метрики**: выдача, использование, отзыв токенов
- **Rate Limiting**: заблокированные запросы
- **File Transfers**: количество файлов, чанков, длительность, ошибки
- **Idempotency**: обнаруженные дубликаты сообщений

Дашборд обновляется каждые 10 секунд и содержит 24 панели с различными метриками.

#### Проверка работы Prometheus

1. Откройте http://localhost:9090
2. Перейдите в **Status → Targets**
3. Убедитесь, что оба target (auth и chat) имеют статус **UP**
4. В разделе **Graph** можно выполнять запросы к метрикам

#### Проверка работы Grafana

1. Откройте http://localhost:3000
2. Войдите с учетными данными `admin` / `admin`
3. Перейдите в **Dashboards → Browse**
4. Откройте дашборд **"DH Secure Chat - Overview"**
5. Убедитесь, что все панели отображают данные

#### Пример конфигурации Prometheus (для справки)

```yaml
scrape_configs:
  - job_name: 'auth-service'
    static_configs:
      - targets: ['auth:8081']
  - job_name: 'chat-service'
    static_configs:
      - targets: ['chat:8082']
```

---

## Архитектура (high‑level)

### Frontend (React + TS + Tailwind)

- SPA с основными модулями:
  - аутентификация (регистрация/логин, автоматический refresh токенов);
  - список пользователей и выбор собеседника;
  - управление ключами (identity + ephemeral);
  - DH‑обмен и установка сессионного ключа с подписью ephemeral‑ключей;
  - E2E‑шифрование/дешифрование сообщений, файлов и голосовых сообщений;
  - WebSocket‑клиент и отображение UI‑состояний чата;
  - отправка и получение файлов с автоматическим шифрованием;
  - выбор режима доступа к файлам (скачивание и просмотр, только просмотр, только скачивание);
  - модальное окно предпросмотра файлов (изображения, PDF, текстовые файлы, видео);
  - защита файлов в режиме "только просмотр" (Canvas рендеринг с водяным знаком, блокировка копирования);
  - видео-кружки для отображения видео файлов с автоматической генерацией превью, lazy loading и воспроизведением прямо в кружке;
  - запись и отправка голосовых сообщений (MediaRecorder API);
  - запись и отправка видео сообщений с использованием камеры и микрофона (MediaRecorder API);
  - воспроизведение голосовых сообщений с контролем прогресса;
  - fingerprint‑верификация с визуальным сравнением (эмодзи);
  - автоматическая блокировка чата при изменении fingerprint;
  - acknowledge mechanism для критичных сообщений;
  - проверка поддержки браузером необходимых API (Web Crypto, MediaRecorder, getUserMedia);
  - мемоизация компонентов для оптимизации производительности;
  - оптимизация создания blob URL через `requestIdleCallback`.

### Backend (Go, net/http)

- **Auth‑service**

  - Регистрация/логин.
  - Валидация учётных данных, хэширование паролей.
  - Генерация и валидация JWT (access token, TTL 15 минут) с JTI для инвалидации.
  - Управление refresh tokens (выдача, обновление, отзыв).
  - Инвалидация access tokens через JTI-based blacklist.
  - Автоматическая очистка expired refresh tokens и revoked tokens.
  - Создание identity‑ключей при регистрации с graceful degradation (регистрация продолжается даже при ошибке сохранения identity-ключа).
  - Все критические операции с БД обёрнуты в Circuit Breaker для защиты от перегрузки.
  - Endpoint `POST /api/auth/revoke` для инвалидации текущего access token.

- **Identity‑service**

  - Управление публичными identity‑ключами пользователей.
  - Генерация fingerprint (SHA‑256 от публичного ключа).
  - API для получения публичных ключей и fingerprint.

- **Chat‑service**

  - REST‑эндпоинты для:
    - `GET /api/chat/me` — информация о текущем пользователе;
    - `GET /api/chat/users?username=...` — поиск пользователя по username (с оптимизацией через GIN индекс для `ILIKE`);
    - `GET /api/identity/users/{id}/key` — получение публичного identity‑ключа пользователя.
  - WebSocket‑эндпоинт:
    - `WS /ws/` — подключение к WebSocket‑hub (JWT передаётся в первом сообщении типа `auth`);
    - Поддержка `permessage-deflate` компрессии для снижения трафика.
  - WebSocket‑hub для:
    - управления подключениями пользователей;
    - маршрутизации зашифрованных сообщений, файлов и голосовых сообщений 1‑на‑1;
    - служебных сообщений (ephemeral ключи, acknowledge, статусы онлайна);
    - передачи файлов и голосовых сообщений по частям (file_start, file_chunk, file_complete);
    - поддержки режимов доступа к файлам (`access_mode`: `both`, `view_only`, `download_only`);
    - валидации MIME-типов для аудио и видео файлов;
    - обновления `last_seen_at` с таймаутом (debounce, не чаще раза в минуту) через circuit breaker для защиты от перегрузки БД;
    - idempotency проверок для предотвращения дублирования сообщений и чанков файлов;
    - асинхронной обработки сообщений через worker pool с очередью;
    - трекинга передачи файлов с автоматической очисткой устаревших записей.

- **База данных (PostgreSQL)**

  - Таблица пользователей с полем `last_seen_at` для отслеживания активности.
  - Таблица публичных identity‑ключей.
  - Таблица refresh tokens с метаданными (user_agent, ip_address).
  - Таблица revoked tokens для инвалидации access tokens (JTI-based blacklist).
  - Расширение `pg_trgm` для оптимизации поиска по username через `ILIKE`.
  - GIN индекс на поле `username` для быстрого поиска пользователей.
  - Connection pooling с метриками использования пула.

- **Reverse‑proxy**

  - TLS‑терминация (HTTPS).
  - Роутинг HTTP/WS на backend‑сервисы.

- **Мониторинг (Prometheus + Grafana)**
  - Автоматический сбор метрик с auth-service и chat-service (только в режиме prod).
  - Prometheus Web UI доступен на порту 9090.
  - Grafana Web UI доступен на порту 3000.
  - Конфигурация в `infra/prometheus/prometheus.yml`.
  - Интервал сбора метрик: 15 секунд.

---

## Планируемые улучшения

- Fallback либа для Crypto Web API.
- Виртуализация списка сообщений для оптимизации производительности при большом количестве сообщений (1000+).
