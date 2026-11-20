# Использование db-adm-tui в режимах r/o и r/w

_[English](usage-modes.md) | Русский_

Каждое подключение начинается в режиме только для чтения (`r/o`). Режим
принадлежит живой сессии: он показывается рядом с базой в верхней строке
статуса, никогда не записывается в профиль и сбрасывается в `r/o` при
переподключении и при переключении базы в PostgreSQL.

Режим только для чтения — это продуктовое ограничение, а не граница
безопасности. Он блокирует операторы до отправки в движок и, где движок это
поддерживает, дополнительно включает нативный read-only слой сессии. Реальные
права по-прежнему определяются учётной записью в базе.

| | `r/o` | `r/w` |
| --- | --- | --- |
| Дерево, Details, индексы, статистика, exact count | да | да |
| Просмотр данных, пагинация, фильтры, разворачивание JSON | да | да |
| Вкладка Privileges | да | да |
| Read-only SQL/MQL запросы | да | да |
| Write/admin SQL | заблокирован | да, с учётом native-прав |
| MQL `$out` / `$merge` | заблокирован | да, с учётом native-прав |
| Матрица ролей/привилегий | заблокирована | да, при наличии native admin-прав |
| Создание/удаление роли | заблокировано | да |
| Создание/удаление пользователя | заблокировано | да, при наличии native admin-прав |
| Вставка/обновление/удаление строк | заблокировано | да, одна ячейка или выбранный набор строк |

В picker профилей `e` редактирует выбранный профиль, а `Del` или `-` удаляет
его с подтверждением. Удаление профиля никогда не удаляет и не изменяет его
source-файл `.pgpass`, `.mypass` или `.mongo.*`. При изменении полей подключения
совпадающая запись source-файла переносится на новый tuple с тем же паролем.

## Режим только для чтения (`r/o`)

Подключение выполняется как обычно; отдельная клавиша не нужна. В верхней
строке отображается `r/o`.

### Осмотреть базу

1. Выберите профиль и нажмите `Enter`.
2. Раскройте базу и схему клавишей `Right`.
3. Используйте `/` для поиска по загруженному дереву, `n`/`N` для перехода
   между совпадениями и `Enter` для перехода к найденному узлу.
4. Выберите таблицу или коллекцию и откройте её вкладки.

Вкладка Details показывает идентичность, статистику, структуру и маркеры
доступа. `Indexes` и `Privileges` показывают native-метаданные. `Ctrl+X s`
считает точное количество строк для выбранной таблицы/коллекции.

### Просмотр и фильтрация данных

С выбранной таблицей или коллекцией откройте `Data`:

```text
Enter        открыть объект
Tab          перейти в основную панель
/            фильтр по загруженным строкам/документам
n / N        следующее / предыдущее совпадение
m            добавить следующую страницу
Ctrl+A       добавить полный батч
Ctrl+Home    перейти к первой загруженной строке
Ctrl+End     перейти к последней загруженной строке
`            развернуть выбранный JSON-столбец
```

`Ctrl+F` добавляет серверный regexp-фильтр по выбранному SQL-столбцу; `Esc`
очищает фильтр или закрывает Results.

### Вставка, обновление и удаление строк/документов

Откройте вкладку Data в `r/w`. `+` открывает форму вставки: SQL-столбцы
начинают в режиме `DEFAULT` и принимают `val`, `NULL` или `DEFAULT`
(`Ctrl+N`/`Ctrl+D`), MongoDB использует редактор extended-JSON документа. `e`
редактирует выбранную ячейку или поле; primary key и MongoDB `_id` доступны
только для чтения. `Space` переключает выделение строки: `-` удаляет текущую
или все выбранные строки/документы, а `e` применяет новое значение выбранного
столбца ко всем выделенным строкам. Batch выполняется одним native statement.
Подтверждение показывает профиль, объект и ключ(и); у страницы должен быть
стабильный ключ, иначе действие блокируется локально. После успеха Data
перезагружается, а операция попадает в history с affected count. Results
доступен только для чтения.

### Read-only SQL

Введите оператор в query bar и нажмите `Enter` либо используйте
`Ctrl+X Ctrl+X` из любой зоны рабочего пространства:

```sql
SELECT id, name, status FROM app.customers ORDER BY id;

WITH active AS (
  SELECT id, name FROM app.customers WHERE status = 'active'
)
SELECT * FROM active ORDER BY name;

EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM app.customers WHERE id = 1;

SHOW search_path;
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'app' AND table_name = 'customers'
ORDER BY ordinal_position;
```

Примеры для MySQL/MariaDB:

```sql
SELECT id, name, status FROM appdb.customers ORDER BY id;
SHOW GRANTS FOR CURRENT_USER();
EXPLAIN SELECT * FROM appdb.customers WHERE id = 1;
```

За одно выполнение принимается только один логический statement. Завершающая
точка с запятой допустима; два statement отклоняются. Guard пропускает
read-only начала вроде `SELECT`, `WITH`, `EXPLAIN`, `SHOW`, `DESCRIBE`,
`VALUES` и `TABLE`.

Read-only SQL отклоняется сообщением:

```text
Blocked: write statements are blocked in read-only mode Ctrl+X Ctrl+W to switch.
```

Примеры, которые не должны попасть в движок при активном `r/o`:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;
DELETE FROM app.customers WHERE id = 2;
CREATE ROLE reporting_ro;
GRANT SELECT ON app.customers TO reporting_ro;
```

### Read-only MQL

MQL ограничен командами `find`, `aggregate`, `count` и `distinct`;
необязательный префикс `db.<collection>.` допускается. Сначала выберите
коллекцию.

```js
db.customers.find({ status: "active" }).sort({ name: 1 }).limit(10)

db.orders.aggregate([
  { $match: { status: "paid" } },
  { $group: { _id: "$customer_id", total: { $sum: "$total" } } }
])

db.customers.count({ status: "active" })

db.customers.distinct("status")
```

В read-only режиме write-стадии в aggregate блокируются:

```text
Blocked: write stages are blocked in read-only mode Ctrl+X Ctrl+W to switch.
```

Произвольный JavaScript (`$where`, `$function`, `$accumulator`) запрещён в
обоих режимах. Прямые команды `insert`, `update` и `delete` не входят в
MQL-поверхность ни в одном из режимов.

### Чек-лист для `r/o`

Используйте `r/o`, когда нужно:

- изучить незнакомую базу;
- посмотреть native-гранты и effective-проверки;
- выполнить `SELECT`/`EXPLAIN`/`SHOW`/`find`/`aggregate`;
- пролистать данные и проверить количество строк;
- скопировать оператор из истории, не рискуя случайно выполнить запись.

В `r/o` следующие действия блокируются с подсказкой о смене режима:

- `+`, `-` и `e` на ролях в Security;
- матрица `Ctrl+X Ctrl+R`;
- write/admin SQL;
- MQL `$out` и `$merge`.

## Режим чтения и записи (`r/w`)

Read-write включается явно и с подтверждением. Режим действует только в
текущей живой сессии.

### Включение и выключение

```text
Ctrl+X Ctrl+W     открыть подтверждение
Enter             включить r/w (в модальном окне "Writes will run on <profile>")
Esc               отмена
```

Для elevated/root-аккаунтов подтверждение повторяет предупреждение. После
`Enter` бейдж меняется на `r/w`, а строка статуса показывает:

```text
read-write mode on. Read-only guard is off.
```

Вернуться в `r/o` можно тем же сочетанием в любой момент. Обратный переход
выполняется сразу, без подтверждения:

```text
read-only mode on. Write statements are blocked.
```

Смена режима запрещена, пока открыт курсор результатов запроса. Сначала
нажмите `Esc`, чтобы закрыть Results.

### Выполнить write statement

`r/w` разрешает один SQL statement любого типа с учётом native-прав текущей
учётной записи. Множественные statements, незакрытые кавычки и произвольный
Mongo JavaScript по-прежнему запрещены.

PostgreSQL:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;

INSERT INTO app.audit_log (id, actor, action)
VALUES (3, 'db_admin', 'manual-update');

CREATE TABLE app.scratch_notes (
  id bigint PRIMARY KEY,
  body text NOT NULL
);

DROP TABLE app.scratch_notes;
```

MySQL/MariaDB:

```sql
UPDATE appdb.customers SET status = 'active' WHERE id = 2;

INSERT INTO appdb.audit_log (id, actor, action)
VALUES (3, 'mdb_admin', 'manual-update');
```

MongoDB: query bar по-прежнему принимает только `find`, `aggregate`, `count` и
`distinct`. Запись возможна через aggregate с `$out` или `$merge`:

```js
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $out: "paid_orders" }
])
```

Прямые `insert`, `update` и `delete` не принимаются. Для ролей и привилегий
Mongo write-поверхностью является матрица.

### Создание и удаление ролей

Роли управляются из дерева:

```text
Security
  roles
    +              создать роль
    Enter          подтвердить имя
```

```text
Security
  roles
    reporting_ro
    -              открыть меню роли
    Down Down Enter
                         Drop role
    Enter              подтвердить
```

Создание роли не добавляет ей привилегии и membership. Для этого используйте
матрицу.

### Создание и удаление пользователей

Пользователи управляются из дерева:

```text
Security
  users
    +              создать пользователя
    Enter          отправить форму
```

Форма запрашивает:

- PostgreSQL: имя login-роли и пароль;
- MySQL/MariaDB: имя пользователя, host (`%` по умолчанию) и пароль;
- MongoDB: имя пользователя, пароль и auth-база. База по умолчанию `admin`; на
  поле базы `Space` переключает `admin` и базы, загруженные в дереве, а
  произвольное значение можно ввести вручную. Это место хранения и
  аутентификации пользователя, а не выдача доступа к базе.

Поле пароля маскируется. После успешного создания native-пользователя пароль
записывается в соответствующий source-файл:

- PostgreSQL: `~/.pgpass` или активный `PGPASSFILE`;
- MySQL/MariaDB: `~/.mypass` или `MYPASSFILE`;
- MongoDB: `~/.mongo.<USER>-in-<DATABASE>`.

Старая совпадающая строка password-файла комментируется `#`, новая строка
вставляется ниже. Файл записывается атомарно с правами `0600`, директория
остаётся `0700`. Остальные строки сохраняются. Пароль не попадает в профиль,
history, лог, argv или текст ошибки.

После создания предлагается:

```text
Save <user>@<database> as a profile? Enter save · Esc skip
```

Сохранение создаёт только optional-профиль и не переключает активную сессию на
нового пользователя. Новый пользователь создаётся без грантов и Mongo-ролей;
назначайте их через матрицы.

Удаление: `Security → users → user → -`, затем подтвердить `Enter`. Удаление
текущего подключённого пользователя блокируется локально. Native-зависимости
могут всё равно заблокировать операцию; native-ошибка показывается как есть.

### Матрица membership

Вкладки Details/Privileges для `Security`, `users`, `roles` и отдельных
principal показывают подсказку `Ctrl+X Ctrl+R`. Откройте матрицу через
`Ctrl+X Ctrl+R`; `1` выбирает `Role × Principal`:

```text
role           app_user   app_operator
reporting        [ ]          [x]
app_ro           [x]          [ ]
```

Роли остаются слева, где помещаются длинные имена и суффиксы баз; пользователи
образуют компактные столбцы. Пользователи MongoDB в `admin` показываются без
`@admin`, а MySQL-столбцы скрывают стандартный host `%`. Если имя текущего
столбца усечено, полное имя показывается после `Tab switch` в заголовке.

- `Space` выдаёт или отзывает выбранный membership.
- `e` открывает детали membership; `Space`/`Enter` переключает ячейку.
- `/` фильтрует имена principal и ролей.
- `Tab`/`Shift+Tab`, `1`/`2` переключают панели матрицы; выбранный principal
  сохраняется между панелями, а целевая панель старается удержать ту же позицию
  на экране.
- `Esc` закрывает матрицу или сначала очищает активный фильтр.

В примере `Space` на `reporting × app_operator` выполняет native
`GRANT reporting TO app_operator`; повторное переключение — соответствующий
`REVOKE`.

### Матрица object privileges

`2` выбирает `Principal/Role × Object`:

```text
principal/role     [db] appdb   [schema] app   customers   orders
reporting              CONNECT       USAGE       SELECT     SELECT
app_user               CONNECT       USAGE       SELECT+2   SELECT
app_writer             —             —           INSERT+2   —
```

- `Space` переключает default native privilege для ячейки.
- `e` открывает полный native multi-select привилегий.
- `Ctrl+A` выбирает все привилегии списка или снимает все, если выбраны все.
- `Enter` применяет diff между текущей ячейкой и выбранным списком.
- Встроенные роли MongoDB доступны только для чтения.

Default-привилегии:

| Движок | Ячейка базы | Ячейка схемы | Ячейка таблицы/представления/коллекции |
| --- | --- | --- | --- |
| PostgreSQL | `CONNECT` | `USAGE` | `SELECT` |
| MySQL/MariaDB | `SELECT` | — | `SELECT` |
| MongoDB | `find` | — | `find` |

Пример:

```text
2                         перейти к privileges
Down / Right              перейти к app_user × app.customers
Space                     выдать или отозвать SELECT
e                         открыть native-список: SELECT, INSERT, UPDATE, ...
Space Space               выбрать INSERT и UPDATE
Enter                     применить diff
```

Матрица пишет native-операторы, а не переносимую абстракцию. PostgreSQL и
MySQL/MariaDB показывают прямые grant'ы пользователям; в MongoDB в панели
привилегий есть только строки ролей.

### Scope и клавиши матрицы

Матрица ограничена текущей базой/схемой и объектами, загруженными в дереве.
Закройте её, выберите другую схему или базу и откройте снова, чтобы сменить
scope.

```text
Tab / Shift+Tab   переключить панели
1 / 2             выбрать membership / privileges
Стрелки           перемещать курсор по ячейкам
Home / End        перейти к краю матрицы
PgUp / PgDn       переместиться на страницу
Space             переключить ячейку
e                 открыть детали
/                 фильтровать строки и столбцы
Esc               закрыть или очистить фильтр
```

Каждая панель хранит собственные cursor, scroll и фильтр, пока активна другая
панель.

### Native-права и другой пользователь

Действует текущее подключение. Чтобы изменить membership или grant'ы другого
пользователя, текущая учётная запись должна иметь native admin-права (например,
PostgreSQL role admin option, MySQL `WITH ADMIN OPTION` или MongoDB
`userAdminAnyDatabase`/`dbAdminAnyDatabase`). Логин и пароль целевого
пользователя не нужны.

Если прав недостаточно, показывается native-ошибка:

```text
not permitted · this account cannot administer roles · Ctrl+D to reconnect with an admin profile
```

Инструмент не выдаёт себя за целевого пользователя и не переключает аккаунт
молча. Подключитесь админским профилем через `Ctrl+D`.

### Аудит и обновление

Успешные запросы, изменения ролей, пользователей и строк записываются в
существующую history с режимом (`r/o` или `r/w`), native statement, статусом и
количеством затронутых строк. Пароли маскируются. После изменения матрицы
обновляются матрица, списки Security, текущая панель Privileges и загруженные
маркеры доступа; после изменения строки Data перезагружается с первой
страницы.

### Чего `r/w` пока не делает

- Вставка более одной строки за раз или изменение нескольких столбцов одним
  оператором; используйте query bar.
- Редактирование атрибутов membership `WITH ADMIN OPTION`/`INHERIT`.
- Смена пароля существующего пользователя и редактирование Mongo-ролей
  пользователя вне матрицы.
- Insert документов MongoDB через query bar.
- Несколько SQL statements за одно выполнение.

## Диагностика режима

| Сообщение | Значение | Действие |
| --- | --- | --- |
| `write statements are blocked in read-only mode` | SQL write/admin в `r/o` | `Ctrl+X Ctrl+W`, подтвердить, повторить |
| `read-write mode is required; Ctrl+X Ctrl+W to switch.` | действие с ролью или матрица в `r/o` | переключиться в `r/w` |
| `write stages are blocked in read-only mode` | MQL `$out`/`$merge` в `r/o` | переключиться в `r/w` |
| `results open — Esc to release` | смена режима или новое действие при открытом курсоре результатов | `Esc`, затем повторить |
| `not permitted ... Ctrl+D` | у текущей учётной записи нет native admin-прав | переподключиться админским профилем |
| `arbitrary JavaScript is not allowed` | `$where`/`$function`/`$accumulator` | переписать MQL без JavaScript |
| `multiple statements are not allowed` | два SQL statement за одно выполнение | выполнять по одному |

Готовое окружение со всеми тремя движками, пользователями, грантами и seed-
данными описано в [`live-test/README.md`](../live-test/README.md). Пошаговый
чек-лист ручной проверки TUI — в
[`live-test/TUI-TEST-PLAN.md`](../live-test/TUI-TEST-PLAN.md).
