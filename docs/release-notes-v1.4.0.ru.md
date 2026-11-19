# db-adm-tui v1.4.0

_Дата релиза: 2026-11-20_

_[English](release-notes-v1.4.0.md) | Русский_

Релиз завершает write-сценарии Story 9 для PostgreSQL, MySQL/MariaDB и
MongoDB: native-пользователи и credential-файлы, матрица доступа, вставка и
удаление строк, обновление ячейки и batch-операции.

## Главное

- **Обновление ячейки (9d):** `e` редактирует выбранную ячейку или поле Data.
  SQL поддерживает `val`, `NULL` и `DEFAULT`; MongoDB применяет типизированный
  `$set`, включая extended JSON. Primary key и `_id` доступны только для
  чтения.
- **Batch-операции (9d):** `Space` выделяет строки. `-` удаляет текущую или все
  выбранные строки; `e` применяет значение выбранного столбца ко всему
  выделению. SQL использует один `OR`-predicate по ключам, MongoDB — `$in` с
  `updateMany`/`deleteMany`.
- **Вставка/удаление строк (9a):** SQL-форма с bound values и Mongo
  extended-JSON редактор; удаление по primary key/`_id` с подтверждением.
- **Матрица доступа (9b):** native membership и object privileges.
- **Пользователи (9c):** создание/удаление, masked passwords, ротация
  `.pgpass`/`.mypass`/`.mongo.*`, опциональный save-as-profile.
- **Аудит:** успешные записи сохраняют mode, native statement, status и
  affected count в history.

## Движки

- PostgreSQL 18: insert, update ячейки, batch, роли/ACL, пользователи,
  credentials.
- MariaDB 11.4: insert, update ячейки, batch, hostless-роли, пользователи,
  `.mypass`.
- MongoDB 8.2: extended-JSON insert, `$set` update, batch delete/update,
  пользователи и роли.

## Безопасность

- Любая запись требует `r/w`; `r/o` показывает подсказку и не доходит до
  движка.
- Один native statement на batch; идентификаторы цитируются, значения
  передаются параметрами.
- Destructive-операции подтверждают профиль, объект и ключ(и).
- Строки без стабильного primary key/`_id` и с binary-ключами блокируются
  локально.
- Native-ошибки остаются видимыми; пароли редактируются из history и ошибок.

## Известные ограничения

- Редактор строк меняет одну ячейку/столбец за раз; несколько столбцов — через
  query bar.
- Multi-row insert недоступен; используйте query bar или несколько statements.
- Write-операции MongoDB в query bar остаются ограничены `$out`/`$merge`.
- Редактирование membership `WITH ADMIN OPTION`/`INHERIT` пока не реализовано.

## Проверка

Live-приёмка на одноразовом стенде `live-test/` пройдена для всех трёх
движков. См. `live-test/ACCEPTANCE-2026-11-20.md` и
`live-test/TUI-TEST-PLAN.md`.

```text
go test ./...
go test -race ./...
go vet ./...
make verify
```
