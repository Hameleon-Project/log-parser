# Log Parser (InfiniBand / ibdiagnet)

Микросервис на Go принимает путь к файлу экспорта ibdiagnet (`*db_csv*`) или к ZIP-архиву с таким файлом внутри, парсит секции `START_NODES` / `START_PORTS` / `START_SYSTEM_GENERAL_INFORMATION`, сохраняет топологию в PostgreSQL и отдаёт REST API.

## Требования

- Go 1.22+
- Docker / Docker Compose (для PostgreSQL и приложения)
- Для проверки линтером из ТЗ: `make lint` (скачивает и запускает `golangci-lint` через `go run`, нужен интернет при первом запуске) или установите [golangci-lint](https://golangci-lint.run/) в `PATH`

## Переменные окружения

| Переменная      | Описание |
|-----------------|----------|
| `DATABASE_URL`  | DSN PostgreSQL (приоритетнее устаревшего `DB_CONN`) |
| `PORT`          | Порт HTTP, по умолчанию `8080` |
| `LOG_LEVEL`     | `DEBUG`, `INFO`, `WARN`, `ERROR` — уровень структурированных логов (JSON) в stdout |

## Сборка, тесты и линт

```bash
make fmt      # gofmt
make test     # go test ./...
make vet      # go vet ./...
make lint     # gofmt + go vet + golangci-lint run ./...
```

Конфигурация линтеров: `.golangci.yml`.

## Локальный запуск

```bash
export DATABASE_URL='postgres://user:password@localhost:5432/log_db?sslmode=disable'
mkdir -p data
# положите выгрузку ibdiagnet в data/, например data/ibdiagnet2.db_csv
go run ./cmd/app
```

## Docker Compose

В корне репозитория каталог `data/` монтируется в контейнер как `/root/data` (только чтение). Логи для парсинга должны лежать у вас в `./data` на хосте.

```bash
docker compose up -d --build
```

Остановка: `docker compose down`.

Команда из формулировки задания: `docker-compose up -d` — используйте файл `docker-compose.yml`, например `docker-compose -f docker-compose.yml up -d`.

## Postman

Импортируйте в Postman файл `postman_collection.json`. Сначала выполните запрос **Parse log** (он сохраняет `log_id` в переменную коллекции), затем **Topology** / **Log meta** и т.д.

## API

Все пути в теле `POST /api/v1/parse/` должны быть **относительными** и находиться под префиксом `data/` (без `..`).

### Примеры `curl`

Разбор файла из смонтированной папки `data/`:

```bash
curl -sS -X POST http://localhost:8080/api/v1/parse/ \
  -H 'Content-Type: application/json' \
  -d '{"path":"data/ibdiagnet2.db_csv"}'
```

Ответ: `{"log_id":1}`.

Топология: узлы, вложенные порты, **рёбра графа** `edges` (связи между портами разных узлов) и **`topology_groups`** — компоненты связности по этим рёбрам (если рёбер нет — одна группа `all_nodes_default`):

```bash
curl -sS http://localhost:8080/api/v1/topology/1
```

Узел и порты:

```bash
curl -sS http://localhost:8080/api/v1/node/1
curl -sS http://localhost:8080/api/v1/port/1
```

Мета по логу:

```bash
curl -sS http://localhost:8080/api/v1/log/1
```

Проверка живости:

```bash
curl -sS http://localhost:8080/health
```

## Топология и логика связей (F-3)

1. **Узел → порты** — иерархия из `START_PORTS`.
2. **Рёбра (`edges`)** — выводятся из счётчиков ibdiagnet и сохраняются в таблице **`links`** (FK на `ports` и `logs`):
   - если среди портов в состоянии **ACTIVE** (`PortState == 4`) ровно **два** порта на **разных** узлах имеют одинаковый положительный **`LinkRoundTripLatency`**, считается физическая связь (типичный случай HCA↔switch на тестовом дампе, значение `270`);
   - для **свитчей**: порты **`PortNum == 65`**, **ACTIVE**, **`LinkRoundTripLatency == 0`** — кандидаты в ISL; по одному такому порту на узел строится **кольцо** в лексикографическом порядке `NodeGuid` (эвристика для кольца из четырёх свитчей на данном файле).
3. **`topology_groups`** — связные компоненты графа по узлам, построенному из `edges` (при отсутствии рёбер — одна группа со всеми узлами).

Документация отражает эвристики: в других дампах ibdiagnet могут понадобиться дополнительные правила или секции отчёта.

## Схема БД (F-4)

Обязательный минимум из ТЗ: `logs`, `nodes`, `ports`, `nodes_info` с внешними ключами. Дополнительно таблица **`links`** хранит выведенные связи между портами для API и воспроизводимости. Скрипт `schema.sql` применяется при старте приложения (в т.ч. `ALTER ... ADD COLUMN IF NOT EXISTS` для обновления старых volume).

## Ошибки парсинга (F-2)

При любой ошибке разбора **весь файл отклоняется**, в БД ничего не записывается; клиент получает `400` с текстом ошибки.
