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
  аудитом в history;
- Story 9d: обновление ячейки по `e`, выделение строк `Space` и batch
  update/delete для SQL и MongoDB с affected count в history.

Изменение нескольких столбцов за раз и multi-row insert остаются сценариями
query bar.

## Статус приёмки

Все срезы Story 9 реализованы и проверены на стенде `live-test/`:

- 9a insert/delete, 9b access matrix, 9c users, 9d cell update/batch writes;
- PostgreSQL 18, MariaDB 11.4, MongoDB 8.2;
- результаты, коммиты и очистка записаны в
  `live-test/ACCEPTANCE-2026-11-20.md`.

## Рекомендуемый следующий шаг: релиз v1.4.0

1. Прогнать полный набор проверок:

   ```sh
   go test ./...
   go test -race ./...
   go vet ./...
   make verify
   ```

2. Собрать архивы:

   ```sh
   make release VERSION=v1.4.0
   ./dist/db-adm-tui_v1.4.0_linux_amd64/db-adm-tui --version
   ./dist/db-adm-tui_v1.4.0_linux_arm64/db-adm-tui --version
   ```

3. Поставить тег и опубликовать release notes с итогами 9a–9d, результатами
   live-приёмки и известными ограничениями ниже.

4. Обновить бинарник на `jojo@uuu:~/db-adm-tui` и синхронизировать
   `~/live-test`.

Не ставьте тег, пока acceptance-файл и README/usage-документация не совпадают с
собранным бинарником.

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
| `v1.4.0` | 9d: update ячейки и batch writes | live update, batch update и batch delete на всех движках |

Текущий релиз можно выпустить как кумулятивный `v1.4.0`; если нужна отдельная
история тегов, сначала поставьте `v1.3.0` на коммит 9a, затем `v1.4.0`.

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
- native-пользователи с ротацией credential-файлов
- insert, update ячейки и batch delete/update строк/документов
- поведение r/o/r/w и native-ошибки доступа
- live fixture-стенд PostgreSQL 18, MariaDB 11.4, MongoDB 8.2

## Engines
- PostgreSQL 18: роли, ACL, MAINTAIN, bound row writes
- MariaDB 11.4: hostless-роли, schema/table grants, batch writes
- MongoDB 8.2: rs0, пользователи, custom/built-in роли, typed $set/batch

## Safety
- явное подтверждение read-write
- один native statement на batch
- history-аудит с mode/status/affected count
- пароли маскируются; primary key и _id защищены

## Known limitations
- редактор строк меняет одну ячейку/столбец за раз; несколько столбцов — через query bar
- multi-row insert недоступен; используйте query bar или несколько statements
- query bar MongoDB остаётся find/aggregate/count/distinct; запись через $out/$merge
- редактирование membership WITH ADMIN OPTION/INHERIT пока не реализовано
```
