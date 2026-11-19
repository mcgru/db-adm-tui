# Live-test credentials and profiles

**Только тестовые данные.** Все пароли ниже несекретны и предназначены только
для одноразового `live-test` стенда.

## Быстрая установка

На той машине, где будет запускаться `db-adm-tui`:

```bash
cd live-test
./install-home.sh
```

Если TUI запускается на `jojo@uuu`, то сначала обновите папку на сервере, затем:

```bash
ssh jojo@uuu 'cd ~/live-test && ./install-home.sh'
```

Скрипт:

- копирует `.pgpass`, `.mongo.*` и
  `.config/db-adm-tui/profiles.json` в `$HOME`;
- делает backup существующих файлов с суффиксом
  `.live-test-backup-<timestamp>`;
- выставляет `0700` на `~/.config/db-adm-tui`;
- выставляет `0600` на `.pgpass`, `.mongo.*`, `profiles.json` и этот файл.

Если вы копируете вручную, обязательно выполните после копирования:

```bash
chmod 700 ~/.config/db-adm-tui
chmod 600 ~/.pgpass ~/.mongo.* ~/.config/db-adm-tui/profiles.json
```

`~/.pgpass` с правами не `0600` игнорируется, и PostgreSQL-профили не смогут
получить пароль.

## Что появится в TUI

После установки в пикере будут готовые профили:

| Label | Движок | База | Пользователь | Пароль | Источник |
| --- | --- | --- | --- | --- | --- |
| `live-pg-root` | PostgreSQL | `appdb` | `postgres` | `pg_root_test_password` | `~/.pgpass`, no TLS |
| `live-pg-admin` | PostgreSQL | `appdb` | `db_admin` | `db_admin_test_password` | `~/.pgpass`, no TLS |
| `live-pg-user` | PostgreSQL | `appdb` | `app_user` | `pg_app_test_password` | `~/.pgpass`, no TLS |
| `live-pg-analyst` | PostgreSQL | `analytics` | `analyst` | `pg_analyst_test_password` | `~/.pgpass`, no TLS |
| `live-maria-root` | MariaDB | `appdb` | `root` | `mariadb_root_test_password` | prompt |
| `live-maria-admin` | MariaDB | `appdb` | `mdb_admin` | `mdb_admin_test_password` | prompt |
| `live-maria-user` | MariaDB | `appdb` | `app_user` | `mdb_app_test_password` | prompt |
| `live-maria-analyst` | MariaDB | `analytics` | `analyst` | `mdb_analyst_test_password` | prompt |
| `live-mongo-root` | MongoDB | `admin` | `root` | `mongo_root_test_password` | `~/.mongo.root-in-admin` |
| `live-mongo-admin` | MongoDB | `admin` | `mongo_admin` | `mongo_admin_test_password` | `~/.mongo.mongo_admin-in-admin` |
| `live-mongo-user` | MongoDB | `appdb` | `app_user` | `mongo_app_test_password` | `~/.mongo.app_user-in-appdb` |
| `live-mongo-analyst` | MongoDB | `analytics` | `analyst` | `mongo_analyst_test_password` | `~/.mongo.analyst-in-analytics` |

MariaDB пока не поддерживает файловый password source в TUI, поэтому пароль
запрашивается при каждом новом подключении. Это ожидаемо.

PostgreSQL-профили уже содержат `allow_no_tls: true`, потому что стенд работает
без TLS. MongoDB-профили используют `replicaSet=rs0` и нужный `authSource` из
`.mongo.*` файлов.

## Подключение

Если TUI запускается на сервере, профили используют локальные `127.0.0.1` и
порты стенда. Если TUI запускается локально, сначала поднимите SSH-туннель:

```bash
ssh -N \
  -L 5432:127.0.0.1:5432 \
  -L 3306:127.0.0.1:3306 \
  -L 57017:127.0.0.1:57017 \
  jojo@uuu
```

После установки достаточно запустить TUI и выбирать профили по label.

## Если profiles.json уже существовал

`install-home.sh` не удаляет старый файл: он сохраняет его как
`profiles.json.live-test-backup-<timestamp>`. Если нужно объединить профили
вручную, не перезаписывайте `profiles.json`, а импортируйте только `.pgpass` и
`.mongo.*` через `i` в TUI.
