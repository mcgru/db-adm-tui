# Live acceptance — 2026-11-20

Проверка Story 9 на одноразовом стенде `live-test/` (PostgreSQL 18, MariaDB
11.4, MongoDB 8.2) через SSH-туннель и настоящий TUI.

- Base: `21a4a7b`
- Feature commit: `144e217` (`feat: add cell updates and batch row writes`)
- Проверенный бинарник: локальная сборка `144e217` (перед релизом `v1.4.0`)

## Результат

Приёмка пройдена. Блокеров нет.

### Read-only gate

- `+`, `e`, `-` в `r/o` показывают `read-write mode` и `Ctrl+X Ctrl+W`.
- Native-запрос не выполняется.
- `Space` только выделяет строки и не пишет.

### PostgreSQL

- Insert с `DEFAULT` для `balance`/`created_at` создал строку с default values.
- `e` на `name` обновил одну ячейку; сообщение `row updated`, Data
  перезагружена.
- Две выделенные строки обновлены одним `UPDATE ... WHERE (id=$2) OR (id=$3)`;
  сообщение `2 rows updated`.
- Две временные строки удалены одним `DELETE ... WHERE (id=$1) OR (id=$2)`;
  сообщение `2 rows removed`.
- History содержит `mode:r/w`, `status:ok` и `affected:1/2`; statement
  содержит placeholders, не значения.
- Таблица без primary key блокирует delete локально.
- Binary key блокируется локально (unit test).

### MariaDB

- `e` на `name` обновил одну строку.
- Batch delete двух временных строк выполнен одним `DELETE ... WHERE
  (id=?) OR (id=?)`.
- History содержит native statement, `r/w`, `ok`, `affected:2`.
- Seed-строка восстановлена после проверки.

### MongoDB

- `e` на `name` выполнил `updateOne` с `$set`; остальные поля документа
  сохранены.
- Batch update двух документов выполнил `updateMany` с `_id:{$in:...}` и
  `$set`.
- Batch delete выполнил `deleteMany` с `_id:{$in:...}`.
- History содержит `affected:2` и native-имена `updateMany`/`deleteMany`.
- Временные документы `_id:9001` и `_id:9002` удалены.

## Очистка

- PostgreSQL: временные строки `9001/9002` удалены, `Alice`/`Bob`/`Carol`
  восстановлены, `app.no_pk` удалена.
- MariaDB: временные строки `9001/9002` удалены, `Alice` восстановлена.
- MongoDB: временные документы `9001/9002` удалены.
- SSH-туннель и tmux-сессии остановлены.

## Проверки кода

```text
go test ./...
go test -race ./...
go vet ./...
make verify
```

Все проходят. Unit tests покрывают r/o gate, single update, batch update/delete,
primary-key protection, missing/binary keys, Mongo typed values и history.
