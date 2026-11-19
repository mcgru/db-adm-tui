# TUI test plan для live-test

Практический чек-лист ручной проверки `db-adm-tui` на стенде `live-test`.
Каждый тест содержит профиль, шаги, ожидаемый результат и очистку.

Стенд одноразовый. Все destructive-тесты выполняются только на `live-test`.

## 1. Подготовка

### 1.1. SSH-туннель

На локальной машине:

```bash
ssh -N \
  -L 5432:127.0.0.1:5432 \
  -L 3306:127.0.0.1:3306 \
  -L 57017:127.0.0.1:57017 \
  jojo@uuu
```

MongoDB в стенде объявляет себя как `localhost:57017`, поэтому туннель должен
пробрасывать именно `57017`.

### 1.2. Стенд на сервере

```bash
ssh jojo@uuu
cd ~/live-test
./up.sh
./status.sh
```

Все три контейнера должны быть `healthy`. Если стенд уже поднят, `./up.sh`
идемпотентно обновит пользователей, роли, гранты и seed-данные.

Проверка native-состояния (выполняется на сервере):

```bash
# PostgreSQL
cd ~/live-test/postgres
docker compose --env-file .env exec -T postgres psql -U postgres -d appdb

# MariaDB
cd ~/live-test/mariadb
docker compose --env-file .env exec -T mariadb \
  mariadb -uroot -p"$(grep '^MARIADB_ROOT_PASSWORD=' .env | cut -d= -f2-)" appdb

# MongoDB
cd ~/live-test/mongo
docker compose --env-file .env exec -T mongo \
  mongosh --port 57017 \
  -u root -p"$(grep '^MONGO_ROOT_PASSWORD=' .env | cut -d= -f2-)" \
  --authenticationDatabase admin appdb
```

### 1.3. Профили TUI

Пароли по умолчанию находятся в `live-test/*/.env.example`. Если на сервере
`.env` уже изменён, берите значения из `~/live-test/*/.env`.

#### Готовый пакет (рекомендуется)

В `live-test/home/` уже лежат:

```text
.pgpass
.mypass
.mongo.root-in-admin
.mongo.mongo_admin-in-admin
.mongo.app_user-in-appdb
.mongo.analyst-in-analytics
.config/db-adm-tui/profiles.json
LIVE-TEST-CREDENTIALS.md
```

`profiles.json` содержит все 12 профилей из таблицы ниже. PostgreSQL-профили
уже с `allow_no_tls`, MariaDB-профили используют `~/.mypass`, MongoDB-профили
используют `.mongo.*` с `replicaSet=rs0` и правильным `authSource`.

На машине, где будет запускаться TUI:

```bash
cd live-test
./install-home.sh
```

Если TUI запускается на `jojo@uuu`, сначала обновите `live-test` на сервере,
затем:

```bash
ssh jojo@uuu 'cd ~/live-test && ./install-home.sh'
```

Скрипт делает backup существующих файлов и выставляет права `0600/0700`.
После этого профили уже видны в TUI, включая MariaDB через `~/.mypass`.
Полный список профилей и паролей — в
[`home/LIVE-TEST-CREDENTIALS.md`](home/LIVE-TEST-CREDENTIALS.md).

Ручная настройка ниже нужна только если вы не используете готовый пакет или
хотите собрать профили выборочно.

#### Ручная настройка (fallback)

#### PostgreSQL

Стенд PostgreSQL работает без TLS. Новое подключение TUI по умолчанию пытается
TLS, поэтому профили PostgreSQL удобнее импортировать из `PGPASSFILE`, а затем
включить в каждом профиле `no TLS`.

Создайте локальный файл `$HOME/.config/db-adm-tui/live-test.pgpass`:

```text
127.0.0.1:5432:appdb:postgres:pg_root_test_password
127.0.0.1:5432:appdb:db_admin:db_admin_test_password
127.0.0.1:5432:appdb:app_user:pg_app_test_password
127.0.0.1:5432:analytics:analyst:pg_analyst_test_password
```

```bash
mkdir -p "$HOME/.config/db-adm-tui"
chmod 700 "$HOME/.config/db-adm-tui"
chmod 600 "$HOME/.config/db-adm-tui/live-test.pgpass"
PGPASSFILE="$HOME/.config/db-adm-tui/live-test.pgpass" ./bin/db-adm-tui
```

В TUI:

```text
i                 импорт из PGPASSFILE
Space             выбрать нужные строки
Enter             импортировать
e                 открыть профиль
Tab x5            дойти до переключателя "no TLS"
Space             включить [x] no TLS
Enter             сохранить
```

Повторите `e` → `no TLS` для всех четырёх PostgreSQL-профилей. После этого
подключение к `127.0.0.1:5432` будет проходить в plaintext. Если auto-label
отличается от таблицы ниже, переименуйте профиль через `e`.

`PGPASSFILE` указывает на постоянный файл в конфиге, поэтому профили смогут
переподключаться без повторного экспорта. Если файл удалён, подключение
PostgreSQL-профиля потребует новый источник пароля.

#### MariaDB

TUI поддерживает `~/.mypass` в формате
`HOST:PORT:DBNAME:USER:PASSWORD`. Готовый bundle уже содержит нужные строки;
`install-home.sh` устанавливает файл с правами `0600`.

Если создаёте профили вручную через `n`, URI:

```text
mysql://root@127.0.0.1:3306/appdb
mysql://mdb_admin@127.0.0.1:3306/appdb
mysql://app_user@127.0.0.1:3306/appdb
mysql://analyst@127.0.0.1:3306/analytics
```

При наличии подходящей строки в `~/.mypass` профиль автоматически получит
`source: ~/.mypass` и не будет запрашивать пароль. Без файла или при
несовпадении строки TUI запросит пароль вручную.

#### MongoDB

Создайте профили через `n`. База в URI одновременно задаёт `authSource`.

URI:

```text
mongodb://root@127.0.0.1:57017/admin
mongodb://mongo_admin@127.0.0.1:57017/admin
mongodb://app_user@127.0.0.1:57017/appdb
mongodb://analyst@127.0.0.1:57017/analytics
```

Пароли — из `live-test/mongo/.env`. После подключения сохраните профиль.
Если для профиля нет локального `~/.mongo.*` source, пароль будет
запрашиваться при каждом новом подключении.

#### Итоговая таблица профилей

| Предлагаемый label | Движок | URI / способ | База | Пользователь | Пароль |
| --- | --- | --- | --- | --- | --- |
| `live-pg-root` | PostgreSQL | импорт `PGPASSFILE` | `appdb` | `postgres` | `pg_root_test_password` |
| `live-pg-admin` | PostgreSQL | импорт `PGPASSFILE` | `appdb` | `db_admin` | `db_admin_test_password` |
| `live-pg-user` | PostgreSQL | импорт `PGPASSFILE` | `appdb` | `app_user` | `pg_app_test_password` |
| `live-pg-analyst` | PostgreSQL | импорт `PGPASSFILE` | `analytics` | `analyst` | `pg_analyst_test_password` |
| `live-maria-root` | MySQL/MariaDB | `~/.mypass` | `appdb` | `root` | `mariadb_root_test_password` |
| `live-maria-admin` | MySQL/MariaDB | `~/.mypass` | `appdb` | `mdb_admin` | `mdb_admin_test_password` |
| `live-maria-user` | MySQL/MariaDB | `~/.mypass` | `appdb` | `app_user` | `mdb_app_test_password` |
| `live-maria-analyst` | MySQL/MariaDB | `~/.mypass` | `analytics` | `analyst` | `mdb_analyst_test_password` |
| `live-mongo-root` | MongoDB | `n` | `admin` | `root` | `mongo_root_test_password` |
| `live-mongo-admin` | MongoDB | `n` | `admin` | `mongo_admin` | `mongo_admin_test_password` |
| `live-mongo-user` | MongoDB | `n` | `appdb` | `app_user` | `mongo_app_test_password` |
| `live-mongo-analyst` | MongoDB | `n` | `analytics` | `analyst` | `mongo_analyst_test_password` |

Если профили PostgreSQL созданы вручную через `n` и подключение падает с
`tls error: server refused TLS connection`, удалите такой профиль и
используйте импорт + `no TLS`, как описано выше.

## 2. Сводка тестов

| ID | Режим | Профиль | Что проверяем |
| --- | --- | --- | --- |
| RO-01 | `r/o` | все admin-профили | подключение, badge `r/o`, дерево |
| RO-02 | `r/o` | `live-pg-admin` | поиск и навигация по дереву |
| RO-03 | `r/o` | `live-pg-admin` | Details, индексы, exact count |
| RO-04 | `r/o` | `live-pg-admin` | Data, пагинация, фильтры, JSON |
| RO-05 | `r/o` | все admin-профили | Privileges tab |
| RO-06 | `r/o` | `live-pg-admin`, `live-maria-admin` | read SQL |
| RO-07 | `r/o` | `live-pg-admin`, `live-maria-admin` | write SQL заблокирован |
| RO-08 | `r/o` | `live-mongo-admin` | read MQL и блокировка `$out`/JS |
| RO-09 | `r/o` | `live-pg-admin` | подтверждение и сброс режима |
| RO-10 | `r/o` | `live-pg-admin` | блокировка смены режима при открытых Results |
| RO-11 | `r/o` | `live-pg-root`, `live-maria-root`, `live-mongo-root` | предупреждение elevated/root |
| RW-01 | `r/w` | `live-pg-admin` | UPDATE + чтение + восстановление |
| RW-02 | `r/w` | `live-maria-admin` | UPDATE + чтение + восстановление |
| RW-03 | `r/w` | `live-mongo-admin` | `$out` в временную коллекцию |
| RW-04 | `r/w` | `live-pg-admin` | multi-statement и битые кавычки |
| RW-05 | `r/w` | `live-pg-admin` | history/audit |
| MX-01 | `r/o` | `live-pg-admin` | матрица заблокирована |
| MX-02 | `r/w` | `live-pg-admin` | навигация, фильтр, панели матрицы |
| MX-03 | `r/w` | `live-pg-admin` | membership PG |
| MX-04 | `r/w` | `live-maria-admin` | membership MariaDB |
| MX-05 | `r/w` | `live-mongo-admin` | membership Mongo |
| MX-06 | `r/w` | `live-pg-admin` | DB/schema/table privileges + `e` |
| MX-07 | `r/w` | `live-maria-admin` | DB/table privileges + `e` |
| MX-08 | `r/w` | `live-mongo-admin` | DB/collection privileges + `e` |
| MX-09 | `r/w` | все admin-профили | create/drop role |
| MX-10 | `r/w` | `live-pg-user`, `live-maria-user`, `live-mongo-user` | native not permitted + `Ctrl+D` |
| MX-11 | `r/w` | `live-pg-admin` | scope и refresh после изменения |
| MX-12 | `r/w` | `live-mongo-admin` | built-in роли только для чтения |
| US-01 | `r/w` | `live-pg-admin` | создание пользователя PostgreSQL + credential |
| US-02 | `r/w` | `live-maria-admin` | создание пользователя MariaDB + credential |
| US-03 | `r/w` | `live-mongo-admin` | создание пользователя MongoDB + credential |
| US-04 | `r/w` | `live-pg-admin` | ротация `.pgpass` при повторном создании |
| US-05 | `r/w` | `live-pg-user` | нельзя удалить текущего пользователя |
| US-06 | `r/w` | `live-pg-admin` | save-as-profile без переключения сессии |
| RS-01 | `r/o` | `live-pg-admin` | Data `+`/`-` заблокированы |
| RS-02 | `r/w` | `live-pg-admin` | insert PostgreSQL + refresh |
| RS-03 | `r/w` | `live-pg-admin` | delete PostgreSQL + confirmation |
| RS-04 | `r/w` | `live-maria-admin` | insert/delete MariaDB |
| RS-05 | `r/w` | `live-mongo-admin` | insert MongoDB extended JSON |
| RS-06 | `r/w` | `live-mongo-admin` | delete MongoDB по `_id` |
| RS-07 | `r/w` | `live-pg-admin` | native error и history |
| CL-01 | — | — | reset и повторный `up.sh` |

### Минимальный smoke перед релизом

Если времени мало, пройдите:

`RO-01`, `RO-06`, `RO-07`, `RO-09`, `RW-01`, `RW-02`, `RW-03`, `MX-01`,
`MX-03`, `MX-04`, `MX-05`, `MX-06`, `MX-08`, `MX-09`, `MX-10`,
`US-01`, `US-02`, `US-03`, `US-05`, `US-06`, `RS-01`, `RS-02`, `RS-03`,
`RS-05`, `RS-06`, `CL-01`.

Полный план ниже покрывает остальные edge cases.

## 3. Read-only тесты

### RO-01. Подключение и badge

**Профиль:** `live-pg-admin`, `live-maria-admin`, `live-mongo-admin`.

**Шаги:**
1. Подключитесь к каждому профилю.
2. Посмотрите верхнюю строку.
3. Раскройте базу и схему/коллекцию.

**Ожидаемо:**
- badge `r/o`;
- видны базы, схемы, таблицы/коллекции;
- admin-профили имеют тег `privileged`;
- `live-pg-user` и `live-maria-user` тега `privileged` не имеют.

**Очистка:** не требуется.

### RO-02. Дерево, поиск, навигация

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Раскройте `appdb` → `app`.
2. Нажмите `/`, введите `customers`.
3. Нажмите `Enter`.
4. Нажмите `n`, затем `N`.
5. Нажмите `Esc`, затем `Right`/`Left` на схеме.

**Ожидаемо:**
- поиск подсвечивает совпадения;
- `n`/`N` ходят по совпадениям;
- `Enter` оставляет поиск и переходит к найденному объекту;
- `Esc` очищает поиск;
- дерево раскрывается и сворачивается без ошибок.

### RO-03. Details, индексы, exact count

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Выберите `app.customers`.
2. Откройте `Details`.
3. Перейдите на `Indexes`.
4. Нажмите `Ctrl+X s`.
5. Повторите для `live-mongo-admin` на `appdb.customers`.

**Ожидаемо:**
- Details показывает объект, базу, схему, статистику и структуру;
- Indexes показывает `customers_pkey` (PG) / `_id_` (Mongo);
- exact count возвращает `3`;
- ошибок `not permitted` для admin-профилей нет.

### RO-04. Data, пагинация, фильтры, JSON

**Профиль:** `live-pg-admin` и `live-mongo-admin`.

**Шаги:**
1. Откройте `Data` для `app.customers` / `appdb.customers`.
2. Нажмите `m`.
3. Нажмите `/`, введите `active`, `Enter`.
4. Нажмите `n` и `N`.
5. Нажмите `Esc`.
6. Нажмите `Ctrl+F`, введите `active`, `Enter`.
7. В Mongo на `analytics.events` нажмите `` ` `` на поле `payload`.

**Ожидаемо:**
- `m` добавляет следующую страницу;
- `/` фильтрует загруженные строки/документы;
- `n`/`N` переходят по совпадениям;
- `Ctrl+F` применяет серверный regexp (SQL);
- JSON-поле разворачивается в подколонки;
- `Esc` очищает фильтр.

### RO-05. Privileges tab

**Профиль:** `live-pg-admin`, `live-maria-admin`, `live-mongo-admin`.

**Шаги:**
1. Выберите `app_user` / `app_user@appdb` в Security или в дереве.
2. Откройте `Privileges`.
3. Для PG выберите `app.customers`.
4. Для Mongo выберите `appdb.customers`.

**Ожидаемо:**
- видны native-гранты и membership;
- PG показывает direct grants и membership;
- MariaDB показывает `app_user@%` и роли;
- Mongo показывает роли `app_writer`/`readWrite` и inherited privileges;
- при ошибке прав показывается `not permitted`, но остальные секции не скрываются.

### RO-06. Read SQL

**Профиль:** `live-pg-admin`, `live-maria-admin`.

**Шаги:**
1. В query bar выполните:

```sql
SELECT id, name, status FROM app.customers ORDER BY id;
```

```sql
SELECT id, name, status FROM appdb.customers ORDER BY id;
```

2. Выполните `EXPLAIN`:

```sql
EXPLAIN SELECT * FROM app.customers WHERE id = 1;
```

3. Выполните `SHOW` и запрос к `information_schema`:

```sql
SHOW search_path;
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'app' AND table_name = 'customers'
ORDER BY ordinal_position;
```

Для MySQL/MariaDB:

```sql
SHOW GRANTS FOR CURRENT_USER();
DESCRIBE appdb.customers;
```

**Ожидаемо:**
- запросы выполняются;
- открывается `Results`;
- строки/колонки и ошибки отображаются без секретов в тексте;
- `Esc` закрывает Results.

### RO-07. Write SQL заблокирован

**Профиль:** `live-pg-admin`, `live-maria-admin` (в `r/o`).

**Шаги:**
1. Выполните:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;
```

```sql
DELETE FROM appdb.customers WHERE id = 2;
```

2. Нажмите `Ctrl+X Ctrl+W` без подтверждения и посмотрите сообщение.

**Ожидаемо:**
- запросы не доходят до движка;
- сообщение содержит `write statements are blocked in read-only mode` и
  `Ctrl+X Ctrl+W to switch.`;
- данные в native-базе не изменились;
- режим остаётся `r/o`.

### RO-08. MQL read и блокировка записи

**Профиль:** `live-mongo-admin`, база `appdb`, коллекция `customers`.

**Шаги:**
1. Выполните:

```js
db.customers.find({ status: "active" }).sort({ name: 1 }).limit(10)
db.customers.count({ status: "active" })
db.customers.distinct("status")
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $group: { _id: "$customer_id", total: { $sum: "$total" } } }
])
```

2. Выполните:

```js
db.orders.aggregate([{ $match: { status: "paid" } }, { $out: "tmp_ro_out" }])
```

3. Выполните:

```js
db.customers.find({ $where: "this.name" })
```

**Ожидаемо:**
- read-команды выполняются;
- `$out` блокируется сообщением `write stages are blocked in read-only mode`;
- JavaScript блокируется сообщением `arbitrary JavaScript is not allowed`;
- `tmp_ro_out` не создаётся.

### RO-09. Смена режима и сброс

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Нажмите `Ctrl+X Ctrl+W`.
2. Нажмите `Esc` — подтверждение отменено.
3. Снова `Ctrl+X Ctrl+W`, затем `Enter`.
4. Нажмите `Ctrl+X Ctrl+W` ещё раз.
5. Переподключитесь к профилю (`Ctrl+D`, затем `Enter`).

**Ожидаемо:**
- при отмене режим остаётся `r/o`;
- после `Enter` badge `r/w`, сообщение `read-write mode on. Read-only guard is off.`;
- обратный переход сразу возвращает `r/o`;
- после переподключения снова `r/o`.

### RO-10. Results block mode switch

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Выполните запрос с большим результатом:

```sql
SELECT generate_series(1, 1000) AS n;
```

2. Не закрывая Results, нажмите `Ctrl+X Ctrl+W`.

**Ожидаемо:**
- сообщение `results open — Esc to release`;
- режим не меняется;
- после `Esc` смена режима снова доступна.

### RO-11. Elevated/root warning

**Профиль:** `live-pg-root`, `live-maria-root`, `live-mongo-root`.

**Шаги:**
1. Подключитесь root-профилем.
2. Нажмите `Ctrl+X Ctrl+W`.
3. Посмотрите текст подтверждения.
4. Нажмите `Esc`.

**Ожидаемо:**
- в верхней строке есть тег `privileged`;
- подтверждение содержит предупреждение, что read-only — продуктовый guard,
  а не граница безопасности;
- если TUI запущен от OS root, дополнительно показывается предупреждение
  `running as root`;
- `Esc` не меняет режим.

## 4. Read-write: SQL

Перед каждым тестом переключитесь в `r/w`:

```text
Ctrl+X Ctrl+W → Enter
```

### RW-01. PostgreSQL UPDATE

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Выполните:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;
```

2. Выполните:

```sql
SELECT id, status FROM app.customers WHERE id = 2;
```

3. Восстановите исходное значение:

```sql
UPDATE app.customers SET status = 'paused' WHERE id = 2;
```

**Ожидаемо:**
- UPDATE выполняется;
- чтение показывает `active`;
- после восстановления снова `paused`;
- в history есть запись с режимом `r/w`.

### RW-02. MariaDB UPDATE

**Профиль:** `live-maria-admin`.

**Шаги:**
1. Выполните:

```sql
UPDATE appdb.customers SET status = 'active' WHERE id = 2;
SELECT id, status FROM appdb.customers WHERE id = 2;
UPDATE appdb.customers SET status = 'paused' WHERE id = 2;
```

**Ожидаемо:**
- все три statement выполняются по отдельности;
- в native-базе id=2 возвращается в `paused`.

### RW-03. MongoDB `$out`

**Профиль:** `live-mongo-admin`, база `appdb`.

**Шаги:**
1. В `r/w` выполните:

```js
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $out: "tmp_paid_orders" }
])
```

2. Выполните:

```js
db.tmp_paid_orders.find({})
```

3. Удалите временную коллекцию на сервере:

```bash
ssh jojo@uuu
cd ~/live-test/mongo
docker compose --env-file .env exec -T mongo \
  mongosh --port 57017 \
  -u root -p"$(grep '^MONGO_ROOT_PASSWORD=' .env | cut -d= -f2-)" \
  --authenticationDatabase admin appdb \
  --eval 'db.tmp_paid_orders.drop()'
```

**Ожидаемо:**
- `$out` создаёт `tmp_paid_orders` только в `r/w`;
- документы читаются;
- после очистки коллекция удалена.

### RW-04. Multi-statement и кавычки

**Профиль:** `live-pg-admin` (в `r/w`).

**Шаги:**
1. Выполните:

```sql
SELECT 1; SELECT 2;
```

2. Выполните:

```sql
SELECT 'unterminated
```

**Ожидаемо:**
- `multiple statements are not allowed`;
- `unterminated quoted text`;
- ни один statement не доходит до движка.

### RW-05. History/audit

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Выполните RW-01.
2. Откройте history в query bar (`Up`).
3. На локальной машине посмотрите файл:

```bash
tail -n 5 ~/.local/state/db-adm-tui/history/commands.jsonl
```

Если `XDG_STATE_HOME` задан, путь будет
`$XDG_STATE_HOME/db-adm-tui/history/commands.jsonl`.

**Ожидаемо:**
- entry содержит statement, mode `r/w` (или `r/o`), status и affected count;
- пароли и password literals замаскированы;
- history не содержит `pg_..._password`.

## 5. Матрица ролей и привилегий

### MX-01. Матрица заблокирована в r/o

**Профиль:** `live-pg-admin` в `r/o`.

**Шаги:**
1. Нажмите `Ctrl+X Ctrl+R`.

**Ожидаемо:**
- матрица не открывается;
- сообщение `Blocked: the access matrix requires read-write mode. Ctrl+X Ctrl+W to switch.`

### MX-02. Навигация и фильтр

**Профиль:** `live-pg-admin` в `r/w`.

**Шаги:**
1. Нажмите `Ctrl+X Ctrl+R`.
2. Переключите панели: `1`, `2`, `Tab`, `Shift+Tab`.
3. На панели `1` задайте фильтр `/` → `app`.
4. Перейдите на панель `2` и вернитесь на `1`.
5. Нажмите `Esc`.

**Ожидаемо:**
- активная панель подсвечена;
- фильтр и cursor панели `1` сохраняются после переключения;
- `Esc` закрывает матрицу;
- после повторного открытия фильтр и cursor можно задать заново.

### MX-03. Membership PostgreSQL

**Профиль:** `live-pg-admin` в `r/w`.

**Подготовка:** создайте временную роль `app_tmp_role` через
`Security → roles → +`.

**Шаги:**
1. `Ctrl+X Ctrl+R`, панель `1`.
2. `/` → `app`.
3. Найдите строку `app_user`, столбец `app_tmp_role`.
4. `Space` — выдать membership.
5. Проверьте native:

```sql
SELECT r.rolname AS member, g.rolname AS role
FROM pg_auth_members m
JOIN pg_roles r ON r.oid = m.member
JOIN pg_roles g ON g.oid = m.roleid
WHERE r.rolname = 'app_user' AND g.rolname = 'app_tmp_role';
```

6. `Space` — отозвать membership.
7. Проверьте, что запрос выше вернул 0 строк.
8. Удалите роль `app_tmp_role` через `Security → roles → -`.

**Ожидаемо:**
- ячейка меняется `[ ]`/`[x]`;
- native-запрос подтверждает grant/revoke;
- сообщение после операции `change applied`;
- после drop роль исчезает из дерева/матрицы.

### MX-04. Membership MariaDB

**Профиль:** `live-maria-admin` в `r/w`.

**Подготовка:** создайте роль `app_tmp_role`.

**Шаги:**
1. Откройте матрицу, панель `1`, фильтр `app`.
2. Выдайте membership `app_user@%` → `app_tmp_role`.
3. Проверьте:

```sql
SHOW GRANTS FOR 'app_user'@'%';
```

4. Отзовите membership.
5. Проверьте, что `app_tmp_role` больше не выдан.
6. Удалите роль.

**Ожидаемо:**
- role name отображается без host (`app_tmp_role`), user — как
  `app_user@%`;
- `GRANT`/`REVOKE` выполняются без syntax error;
- после drop роль исчезает.

### MX-05. Membership MongoDB

**Профиль:** `live-mongo-admin` в `r/w`, база `appdb`.

**Подготовка:** создайте роль `app_tmp_role` в `appdb` (текущая база).

**Шаги:**
1. Откройте матрицу, панель `1`, фильтр `app`.
2. Выдайте `app_tmp_role@appdb` пользователю `app_user@appdb`.
3. Проверьте:

```js
db.getSiblingDB("appdb").getUser("app_user").roles
```

4. Отзовите membership.
5. Проверьте, что роли больше нет.
6. Удалите роль.

**Ожидаемо:**
- grant/revoke отражаются в `usersInfo`;
- built-in роли в колонках не ломают матрицу;
- после drop роль исчезает.

### MX-06. Privileges PostgreSQL + `e`

**Профиль:** `live-pg-admin` в `r/w`.

**Подготовка:** создайте пустую роль `app_tmp_role`.

**Шаги:**
1. Откройте матрицу, панель `2`.
2. `/` → `app_tmp_role`.
3. Для ячейки `[db] appdb` нажмите `Space` — выдать `CONNECT`.
4. Для ячейки `[schema] app` нажмите `Space` — выдать `USAGE`.
5. Для ячейки `customers` нажмите `e`, выберите `SELECT`, `INSERT`,
   `UPDATE`, нажмите `Enter`.
6. Проверьте native:

```sql
SELECT grantee, table_schema, table_name, privilege_type
FROM information_schema.role_table_grants
WHERE grantee = 'app_tmp_role'
ORDER BY table_schema, table_name, privilege_type;

SELECT datname, datacl
FROM pg_database WHERE datname = 'appdb';

SELECT nspname, nspacl
FROM pg_namespace WHERE nspname = 'app';
```

7. Через `e` снимите `INSERT`/`UPDATE`, оставив `SELECT`; проверьте.
8. Через `e` снимите `SELECT`; проверьте, что table grants пусты.
9. Снимите `CONNECT` и `USAGE`.
10. Удалите `app_tmp_role`.

**Ожидаемо:**
- `Space` меняет default `CONNECT`/`USAGE`/`SELECT`;
- `e` применяет только diff;
- native ACL содержит только выбранные привилегии;
- после очистки роль удаляется.

### MX-07. Privileges MariaDB + `e`

**Профиль:** `live-maria-admin` в `r/w`.

**Подготовка:** создайте роль `app_tmp_role`.

**Шаги:**
1. Панель `2`, фильтр `app_tmp_role`.
2. `[db] appdb` + `Space` — выдать `SELECT`.
3. `customers` + `e` — выбрать `SELECT`, `INSERT`; `Enter`.
4. Проверьте:

```sql
SELECT GRANTEE, TABLE_SCHEMA, TABLE_NAME, PRIVILEGE_TYPE
FROM information_schema.TABLE_PRIVILEGES
WHERE GRANTEE LIKE '%app_tmp_role%';

SELECT GRANTEE, TABLE_SCHEMA, PRIVILEGE_TYPE
FROM information_schema.SCHEMA_PRIVILEGES
WHERE GRANTEE LIKE '%app_tmp_role%';
```

5. Снимите `INSERT`, затем `SELECT`, затем DB `SELECT`.
6. Удалите роль.

**Ожидаемо:**
- hostless-роль корректно отображается и цитируется;
- diff из `e` применяется;
- native-таблицы не содержат лишних грантов после очистки.

### MX-08. Privileges MongoDB + `e`

**Профиль:** `live-mongo-admin` в `r/w`, база `appdb`.

**Подготовка:** создайте роль `app_tmp_role`.

**Шаги:**
1. Панель `2`, фильтр `app_tmp_role`.
2. `[db] appdb` + `Space` — выдать `find`.
3. `customers` + `Space` — выдать `find`.
4. `customers` + `e` — добавить `insert`, `update`; `Enter`.
5. Проверьте:

```js
db.getSiblingDB("appdb").getRole("app_tmp_role")
```

6. Через `e` снимите `insert`/`update`, затем `find`.
7. Снимите DB `find`.
8. Удалите роль.

**Ожидаемо:**
- в `getRole` privileges видны точные resource/actions;
- `e` меняет только выбранные действия;
- после очистки `app_tmp_role` удалена.

### MX-09. Create/drop role

**Профиль:** `live-pg-admin`, `live-maria-admin`, `live-mongo-admin` в `r/w`.

**Шаги:**
1. `Security → roles → +`.
2. Введите `tmp_drop_role`, `Enter`.
3. Проверьте native-наличие роли.
4. Выберите `tmp_drop_role` в дереве.
5. `-` → `Down Down Enter` (`Drop role`) → `Enter`.
6. Проверьте, что роль удалена.

**Ожидаемо:**
- create/drop работают на всех движках;
- ошибки зависимостей показываются native-текстом;
- tool не изобретает cascade.

### MX-10. Non-admin native error

**Профиль:** `live-pg-user`, `live-maria-user`, `live-mongo-user` в `r/w`.

**Шаги:**
1. Откройте матрицу `Ctrl+X Ctrl+R`.
2. Попробуйте изменить membership/privilege (например, выдать
   `app_tmp_role` пользователю или изменить `reporting`).
3. Посмотрите сообщение.

**Ожидаемо:**
- изменение не применяется;
- показывается native `not permitted`/`permission denied`;
- подсказка `Ctrl+D to reconnect with an admin profile`;
- tool не переключает аккаунт автоматически.

### MX-11. Scope и refresh

**Профиль:** `live-pg-admin` в `r/w`.

**Шаги:**
1. Выберите `app.customers`, откройте матрицу.
2. Убедитесь, что в заголовке scope `appdb · app`.
3. Закройте матрицу, выберите `analytics.events`, откройте снова.
4. Измените membership/privilege.
5. Закройте матрицу.
6. Проверьте, что Privileges/Details и Security обновились.

**Ожидаемо:**
- scope меняется на `appdb · analytics`;
- после mutation обновляются матрица, Security, Privileges и access markers;
- ошибок stale data нет.

### MX-12. MongoDB built-in roles

**Профиль:** `live-mongo-admin` в `r/w`.

**Шаги:**
1. Откройте панель `1` (`Principal × Role`).
2. Убедитесь, что built-in роли (`read@analytics`, `readWrite@appdb`, ...)
   присутствуют как колонки.
3. Откройте панель `2` (`Principal/Role × Object`).
4. Убедитесь, что built-in роли не являются редактируемыми строками.
5. Убедитесь, что custom roles (`app_writer`, `analytics_read`) присутствуют
   как строки.

**Ожидаемо:**
- built-in роли в membership доступны как колонки и их можно выдавать
  пользователям;
- в privileges встроенные роли не редактируются;
- custom roles редактируются через `Space`/`e`.

## 6. Пользователи (Story 9c)

Все операции — только в `r/w`. Пароли в тестах одноразовые; после теста
удалите созданного пользователя.

### US-01. PostgreSQL: создание пользователя и credential

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Перейдите в `r/w`.
2. `Security → users → +`.
3. Введите имя `tmp_user_pg`, `Tab`, пароль `tmp_pg_secret`.
4. `Enter`.
5. Проверьте native:

```sql
SELECT rolname, rolcanlogin FROM pg_roles WHERE rolname = 'tmp_user_pg';
```

6. Проверьте файл credential на машине, где запущен TUI:

```bash
grep 'tmp_user_pg' ~/.pgpass
stat -c '%a %n' ~/.pgpass
```

7. В модальном окне save-as-profile нажмите `Enter`.
8. Удалите пользователя: `Security → users → tmp_user_pg → - → Enter`.

**Ожидаемо:**
- native-роль существует с `LOGIN`;
- в `~/.pgpass` есть строка
  `127.0.0.1:5432:appdb:tmp_user_pg:tmp_pg_secret`;
- файл `0600`;
- save-as-profile сохраняет профиль, но верхняя строка остаётся
  `live-pg-admin`;
- после drop native-роль исчезает;
- в history нет пароля, только `PASSWORD '***'`.

### US-02. MariaDB: создание пользователя и `.mypass`

**Профиль:** `live-maria-admin`.

**Шаги:**
1. `Security → users → +`.
2. Имя `tmp_user_my`, host `%`, пароль `tmp_my_secret`.
3. `Enter`.
4. Проверьте native:

```sql
SELECT User, Host FROM mysql.user WHERE User = 'tmp_user_my';
```

5. Проверьте:

```bash
grep 'tmp_user_my' ~/.mypass
stat -c '%a %n' ~/.mypass
```

6. Удалите пользователя через `-`.

**Ожидаемо:**
- создан `tmp_user_my@%`;
- в `~/.mypass` есть `127.0.0.1:3306:appdb:tmp_user_my:tmp_my_secret`;
- файл `0600`;
- после drop пользователь исчезает;
- повторное создание того же имени даёт native duplicate error и не меняет
  credential-файл.

### US-03. MongoDB: создание пользователя и `.mongo.*`

**Профиль:** `live-mongo-admin`, текущая база `appdb`.

**Шаги:**
1. `Security → users → +`.
2. Имя `tmp_user_mongo`, пароль `tmp_mongo_secret`.
3. `Enter`.
4. Проверьте native:

```js
db.getSiblingDB("appdb").getUser("tmp_user_mongo")
```

5. Проверьте файл:

```bash
grep 'tmp_user_mongo' ~/.mongo.tmp_user_mongo-in-appdb
stat -c '%a %n' ~/.mongo.tmp_user_mongo-in-appdb
```

6. Удалите пользователя через `-`.

**Ожидаемо:**
- пользователь создан в `appdb` с `roles: []`;
- credential-файл содержит полный `mongodb://tmp_user_mongo:...@127.0.0.1:57017/appdb?...`;
- файл `0600`;
- после drop `getUser` возвращает `null`.

### US-04. Ротация credential-файла

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Создайте `tmp_rotate_pg` с паролем `first_password`.
2. Сохраните профиль или нажмите `Esc`.
3. Удалите пользователя через `-`.
4. Создайте пользователя с тем же именем и паролем `second_password`.
5. Откройте `~/.pgpass` и проверьте старую строку.

**Ожидаемо:**
- старая строка закомментирована `#`;
- новая строка находится ниже и содержит `second_password`;
- unrelated lines не изменены;
- файл остаётся `0600`;
- удалите пользователя после проверки.

### US-05. Удаление текущего пользователя

**Профиль:** `live-pg-user` (`app_user`).

**Шаги:**
1. Перейдите в `r/w`.
2. `Security → users → app_user → -`.
3. Подтвердите `Enter`.

**Ожидаемо:**
- локальная ошибка `cannot drop the currently connected user`;
- native drop не выполняется;
- сессия остаётся открытой.

### US-06. Save-as-profile без переключения

**Профиль:** `live-pg-admin`.

**Шаги:**
1. Создайте `tmp_profile_pg`.
2. В save-as-profile нажмите `Enter`.
3. Посмотрите верхнюю строку и статус.
4. `Ctrl+D`, найдите новый профиль в picker.

**Ожидаемо:**
- профиль `tmp_profile_pg@appdb` сохранён;
- верхняя строка всё ещё показывает `live-pg-admin`;
- текущая сессия не переключилась на нового пользователя;
- пароль отсутствует в профиле, history и статусе.

## 7. Строки и документы (Story 9a)

Все действия — только из вкладки Data и только в `r/w`. Results не редактируется.
Тестовые строки используют id/`_id` `9001` и удаляются в конце каждого теста.

### RS-01. Read-only gate

**Профиль:** `live-pg-admin` в `r/o`.

**Шаги:**
1. Откройте `app.customers` → `Data`.
2. Нажмите `+`.
3. Нажмите `-`.

**Ожидаемо:**
- модальное окно не открывается;
- сообщение содержит `read-write mode` и `Ctrl+X Ctrl+W`;
- native-запрос не выполняется.

### RS-02. PostgreSQL: insert + refresh

**Профиль:** `live-pg-admin` в `r/w`.

**Шаги:**
1. `app.customers` → `Data` → `+`.
2. Заполните:
   - `id` = `9001`;
   - `Tab`, `name` = `tmp_pg_row`;
   - `Tab`, `status` = `active`;
   - `Tab`, `balance` → `Ctrl+D` (`DEFAULT`);
   - `Tab`, `created_at` → `Ctrl+D` (`DEFAULT`).
3. `Enter`.
4. Проверьте native:

```sql
SELECT id, name, status, balance, created_at
FROM app.customers WHERE id = 9001;
```

5. Проверьте history на машине TUI:

```bash
tail -n 5 ~/.local/state/db-adm-tui/history/commands.jsonl
```

**Ожидаемо:**
- строка создана, `balance = 0`, `created_at` заполнен;
- Data перезагружена с offset 0 и показывает `9001`;
- статус содержит `row inserted` и native `INSERT ...`;
- history содержит mode `r/w`, status `ok`, affected count;
- значения переданы параметрами, а не вклеены в SQL.

### RS-03. PostgreSQL: delete + confirmation

**Профиль:** `live-pg-admin` в `r/w`, строка `9001` из RS-02.

**Шаги:**
1. В `Data` найдите `tmp_pg_row` (`/` или `Ctrl+End`).
2. Нажмите `-`.
3. Проверьте подтверждение.
4. `Enter` (`Enter remove`).
5. Проверьте native count и history.

**Ожидаемо:**
- подтверждение называет профиль `live-pg-admin`, объект `public.customers` и
  ключ `id=9001`;
- до `Enter` native DELETE не выполняется;
- после `Enter` строка исчезает, Data обновляется;
- history содержит `DELETE`, mode `r/w`, status `ok`.

### RS-04. MariaDB: insert/delete

**Профиль:** `live-maria-admin` в `r/w`.

**Шаги:**
1. `appdb.customers` → `Data` → `+`.
2. `id`=`9001`, `name`=`tmp_my_row`, `status`=`active`,
   `balance` и `created_at` → `Ctrl+D`.
3. `Enter`; проверьте native:

```sql
SELECT id, name, status, balance, created_at
FROM appdb.customers WHERE id = 9001;
```

4. Выберите строку и нажмите `-` → `Enter`.
5. Проверьте, что строка удалена.

**Ожидаемо:**
- insert/delete работают так же, как в PostgreSQL;
- native default values применены;
- Data и history обновляются после каждой операции.

### RS-05. MongoDB: insert extended JSON

**Профиль:** `live-mongo-admin` в `r/w`, база `appdb`.

**Шаги:**
1. `customers` → `Data` → `+`.
2. Очистите поле и введите:

```json
{"_id":9001,"name":"tmp_mongo_row","status":"active","balance":{"$numberDecimal":"0"}}
```

3. `Enter`.
4. Проверьте native:

```js
db.getSiblingDB("appdb").customers.findOne({_id: 9001})
```

**Ожидаемо:**
- документ создан через `insertOne`;
- Data перезагружена и показывает `_id=9001`;
- status содержит `row inserted`;
- history содержит `insertOne`, mode `r/w`, status `ok`.

### RS-06. MongoDB: delete by `_id`

**Профиль:** `live-mongo-admin` в `r/w`, документ `9001` из RS-05.

**Шаги:**
1. Выберите документ `9001` в Data.
2. Нажмите `-`, проверьте подтверждение с `_id=9001`.
3. `Enter`.
4. Проверьте native:

```js
db.getSiblingDB("appdb").customers.findOne({_id: 9001})
```

**Ожидаемо:**
- подтверждение называет профиль, `appdb.customers` и `_id=9001`;
- ровно один документ удалён;
- Data обновлена, history содержит `deleteOne`, mode `r/w`, status `ok`.

### RS-07. Native error и no-key guard

**Профиль:** `live-pg-admin` в `r/w`.

**Шаги:**
1. `app.customers` → `Data` → `+`.
2. Введите id `1` (уже существует), name `tmp_dup`, status `active`,
   остальные поля `Ctrl+D`; `Enter`.
3. Убедитесь, что Data не изменилась и ошибка visible.
4. Создайте native-таблицу без primary key и откройте её Data:

```sql
CREATE TABLE app.no_pk (id bigint, body text);
INSERT INTO app.no_pk VALUES (1, 'no key');
```

5. Нажмите `-`.

**Ожидаемо:**
- duplicate key показывается native-текстом PostgreSQL, без подмены;
- history получает статус `error`, а не `ok`;
- для `no_pk` delete блокируется локально сообщением
  `cannot identify a row without a primary key or _id`, native DELETE не
  выполняется;
- после теста удалите таблицу:

```sql
DROP TABLE app.no_pk;
```

## 8. Очистка

### CL-01. Reset и идемпотентность

**Шаги:**
1. На сервере:

```bash
cd ~/live-test
./reset.sh --yes
```

2. Дождитесь `healthy`.
3. Повторно:

```bash
./up.sh
```

4. Проверьте:

```bash
./status.sh
```

5. В TUI переподключитесь и проверьте count `3`.

**Ожидаемо:**
- reset удаляет данные и поднимает чистый стенд;
- повторный `up.sh` не падает и не дублирует seed;
- все три контейнера healthy;
- после reset снова доступны исходные пользователи/роли/гранты.

## 8. Форма фиксации результатов

| ID | Статус | Профиль | Факт / сообщение | Артефакт |
| --- | --- | --- | --- | --- |
| RO-01 | `PASS` / `FAIL` / `BLOCKED` | | | скрин/лог |
| RO-02 | | | | |
| ... | | | | |

Для `FAIL` приложите:

- точную последовательность клавиш;
- профиль и базу;
- сообщение в статусной строке;
- native-проверку до/после;
- фрагмент history без секретов.

Для `BLOCKED` укажите, чего не хватило: SSH, Docker, native-права, сеть,
конкретный движок.
