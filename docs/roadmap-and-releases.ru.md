# Следующие шаги и тегирование релизов

_[English](roadmap-and-releases.md) | Русский_

## Текущее состояние

В репозитории уже есть:

- v1 read-only сценарии, query bar, инспекция привилегий, history и release
  hardening;
- v2 Story 9b — матрица доступа (membership и object privileges);
- одноразовый live-стенд PostgreSQL 18, MariaDB 11.4 и MongoDB 8.2 в
  `live-test/`;
- live-совместимость для hostless-ролей MariaDB, обнаружения ролей MongoDB и
  PostgreSQL 18 `MAINTAIN`;
- импорт и connection-source `~/.mypass` для MySQL/MariaDB;
- Story 9c: создание/удаление native-пользователей, masked passwords, запись
  credential-файлов с ротацией и save-as-profile;
- Story 9a: вставка и удаление одной строки/документа из вкладки Data с
  идентификацией по primary key/`_id`, гейтом `r/w`, обновлением Data и
  аудитом в history.

Обновление строк и запись нескольких строк в 9a не входят; используйте query
bar.

## Рекомендуемый следующий шаг

### 1. Live TUI acceptance для 9b

Используйте новый стенд `live-test/` по SSH и проверьте именно TUI, а не только
engine-пробы:

- подключитесь как `db_admin`, `mdb_admin` и `mongo_admin`;
- откройте `Ctrl+X Ctrl+R`, переключите обе панели через `Tab`/`1`/`2`;
- переключите membership и table/schema/database privileges;
- откройте `e`-детали и примените multi-select diff привилегий;
- создайте и удалите роль;
- проверьте обновление матрицы, списков Security, панели Privileges и маркеров
  доступа;
- проверьте, что `r/o` блокирует матрицу и write-запросы, а `r/w` разрешает их;
- проверьте, что history содержит режим, native statement, статус и количество
  затронутых строк;
- повторите неуспешную операцию под non-admin пользователем и проверьте
  native-ошибку с подсказкой `Ctrl+D`;
- для MongoDB проверьте replica set, поведение `$out`/`$merge` в `r/w` и то,
  что встроенные роли доступны только для чтения.

Исправьте найденное до объявления Story 9b принятой.

### 2. Live TUI acceptance для 9c

Проверьте новый user flow на стенде `live-test/`:

- создайте пользователя в PostgreSQL, MariaDB и MongoDB; убедитесь, что
  native-аккаунт появился;
- проверьте credential-файлы: правильную строку/URI, закомментированную старую
  строку, новую строку ниже, `0600` на файле и `0700` на директории;
- создайте того же пользователя повторно и проверьте native duplicate error;
- удалите созданного пользователя и проверьте исчезновение из Security и
  native-каталога;
- убедитесь, что удаление текущего подключённого пользователя блокируется;
- проверьте save-as-profile: профиль сохраняется, активная сессия не
  переключается;
- убедитесь, что пароли не появляются в picker, history, статусах и ошибках.

Исправьте найденное до объявления 9c принятой.

### 3. Live TUI acceptance для 9a

Проверьте вставку/удаление одной строки на стенде `live-test/`:

- убедитесь, что `+`/`-` блокируются в `r/o` и доходят до движка только в `r/w`;
- вставьте строку в PostgreSQL и MariaDB с полями `val`, `NULL` и `DEFAULT`;
  проверьте native-строку, обновление Data и запись в history;
- вставьте extended-JSON документ в MongoDB и проверьте `insertOne`;
- удалите строку по primary key в SQL и по `_id` в MongoDB, проверьте текст
  подтверждения и исчезновение ровно одной строки/документа;
- убедитесь, что таблица без primary key блокирует удаление локально;
- вызовите native-ошибку (duplicate key, bad cast, missing privilege) и
  проверьте, что сообщение движка остаётся видимым, а audit-запись не получает
  статус `ok`.

Исправьте найденное до объявления 9a принятой.

### 4. Документация релиза

Обновите README, usage-документацию, статус story и release notes с финальным
scope пользователей/ролей/строк и ограничениями live-проверки.

## Политика тегирования

Используйте Semantic Versioning с префиксом `v`:

```text
vMAJOR.MINOR.PATCH
v1.1.0
v1.1.0-rc.1
```

- **MAJOR** — несовместимые изменения profile/state schema, CLI или поведения.
- **MINOR** — пользовательские фичи: матрица доступа, редактирование
  пользователей и т.п.
- **PATCH** — обратно совместимые исправления без новых возможностей.
- **Pre-release** — `-rc.N`, пока фича код-полна, но live-acceptance или
  совместимость ещё открыты.

### Предлагаемые вехи

| Тег | Содержание | Условие |
| --- | --- | --- |
| `v1.1.0-rc.1` | матрица 9b, live-стенд, live-совместимость | engine-пробы и локальный стенд проходят; live-smoke TUI ещё открыт |
| `v1.1.0` | принятая 9b | live-smoke TUI на всех трёх движках, документация и release notes обновлены |
| `v1.2.0` | 9c: пользователи и credential-файлы | live-проверка создания, ротации, удаления и save-as-profile |
| `v1.3.0` | 9a: строки/документы | live insert/delete одной строки на всех движках |

Если 9b ещё не принята live, не ставьте `v1.1.0`; используйте `v1.1.0-rc.1`
или подождите.

Существующие теги `1.0.1` и `v.1.0.0` указывают на один старый коммит и
названы непоследовательно. Не переписывайте уже опубликованные теги.
Дальше используйте `v1.1.0`. Если эти теги никогда не публиковались, их можно
отдельно привести к `v1.0.0`/`v1.0.1`.

### Когда создавать тег

Ставьте тег только когда выполнено всё:

- рабочее дерево чистое, `main` актуален;
- `go test ./...`, `go test -race ./...`, `go vet ./...` и `make verify`
  проходят;
- `make release VERSION=vX.Y.Z` собирает оба Linux-архива;
- бинарник сообщает ожидаемые version, commit и date;
- README/usage-документация и release notes описывают текущий scope и
  известные ограничения;
- live-smoke пройден для затронутых движков;
- нет известных блокеров и молча отложенных пунктов acceptance.

Не тегируйте каждый коммит или незавершённую story. Для промежуточных
checkpoint используйте SHA коммитов.

### Чек-лист релиза

```sh
git checkout main
git pull --ff-only

go test ./...
go test -race ./...
go vet ./...
make verify

make release VERSION=v1.1.0
./dist/db-adm-tui_v1.1.0_linux_amd64/db-adm-tui --version
./dist/db-adm-tui_v1.1.0_linux_arm64/db-adm-tui --version

git tag -a v1.1.0 -m "db-adm-tui v1.1.0"
git push origin main
git push origin v1.1.0
```

Перед тегом выполните live-acceptance на `live-test/`. После тега опубликуйте
оба архива из `dist/` и приложите release notes.

## Структура release notes

```text
## Highlights
- матрица доступа для membership и native object privileges
- поведение r/o/r/w и native-ошибки доступа
- live fixture-стенд PostgreSQL 18, MariaDB 11.4, MongoDB 8.2

## Engines
- PostgreSQL 18: роли, ACL, MAINTAIN
- MariaDB 11.4: hostless-роли, schema/table grants
- MongoDB 8.2: rs0, пользователи, custom/built-in роли

## Safety
- явное подтверждение read-write
- один логический statement за выполнение
- history-аудит с mode/status/affected count
- пароли маскируются

## Known limitations
- обновление строк и запись нескольких строк; используйте query bar
- query bar MongoDB остаётся find/aggregate/count/distinct; запись через $out/$merge
- редактирование membership WITH ADMIN OPTION/INHERIT пока не реализовано
```
