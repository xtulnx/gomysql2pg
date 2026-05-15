# gomysql2pg

![logo.png](image/logo.png)


([CN](https://github.com/iverycd/gomysql2pg/blob/master/readme_cn.md))

## Features
  MySQL database migration to postgresql kernel database,such as postgresql(pgsql),vastbase,Huawei postgresql,GaussDB,telepg,Kingbase V8R6

* No need for cumbersome deployment, ready to use out of the box, compact and lightweight

* Online migration of MySQL to target database tables, views, indexes, foreign keys, self increasing columns, and other objects

* Multiple goroutines migrate data concurrently, fully utilizing CPU multi-core performance

* Migrate Partial Tables and row data

* Record migration logs, dump SQL statements for DDL object creation failures such as tables and views

* One click migration of MySQL to postgreSQL, convenient, fast, and easy to use

![gomysql2pg_en_struct.png](image/gomysql2pg_en_struct.png)


## Pre-requirement
The running client PC needs to be able to connect to both the source MySQL database and the target database simultaneously

run on Windows,Linux,macOS

## Installation

tar and run 

e.g.

`[root@localhost opt]# tar -zxvf gomysql2pg-linux64-0.1.7.tar.gz`

## How to use

The following is an example of a Windows platform, with the same command-line parameters as other operating systems

`Note`: Please run this tool in `CMD` on a `Windows` system, or in a directory with read and write permissions on `MacOS` or `Linux`

### 1 Edit yml configuration file

Edit the `example.cfg` file and input the source(src) and target(dest) database information separately

```yaml
src:
  host: 192.168.1.3
  port: 3306
  database: test
  username: root
  password: 11111
dest:
  dbType: Gauss # If you are using the openGauss type (openGauss 5.0.2 passed the test), please be sure to add this line, and for non openGauss types, please annotate this line
  host: 192.168.1.200
  port: 5432
  database: test
  username: test
  password: 11111
pageSize: 100000
maxParallel: 30
charInLength: false
useNvarchar2: false
Distributed: false
tables:
  test1:
    - select * from test1
  test2:
    - select * from test2
exclude:
  - 'log1'
  - 'log2'
  - '*_log'

```

pageSize: Number of records per page for pagination query
```
e.g.
pageSize:100000
SELECT t.* FROM (SELECT id FROM test  ORDER BY id LIMIT 0, 100000) temp LEFT JOIN test t ON temp.id = t.id;
```
- maxParallel: The maximum number of concurrency that can run goroutine simultaneously

- tables: Customized migrated tables and customized query source tables, indented in yml format

- exclude: Tables that do not migrate to target database, indented in yml format,Currently, there is new support for wildcard asterisk (\*), such as test*

- charInLength: If true, varchar type stores character length instead of bytes, so it is only compatible with some databases

- useNvarchar2: if true，dest database use nvarchar2(like GaussDB)

- Distributed: If it is true, the database is a distributed database such as GaussDB 8.1.3 And before adding the primary key, first change the table distribution column as the primary key, and then add the primary key.

### 2 Full database migration

Migrate entire database table structure, row data, views, index constraints, and self increasing columns to target database

gomysql2pg.exe  --config file.yml
```
e.g.
gomysql2pg.exe --config example.yml

on Linux and MacOS you can run
./gomysql2pg --config example.yml
```

### 3 Batch migration (multiple databases in one run)

Use case: migrate several databases / source-destination pairs in a single run. Two scripts at the repo root provide identical behavior: `run_batch.sh` for Linux / macOS and `run_batch.ps1` for Windows.

**Preparation**

Place one yml per database under `configs/`. Use a numeric prefix (e.g. `01_a.yml`, `02_b.yml`) to control execution order. You can also generate them in bulk from `configs/db_config.csv` via `gen_configs.sh`:

```bash
bash gen_configs.sh                       # default: configs/db_config.csv -> configs/
bash gen_configs.sh my.csv out_dir        # custom csv and output dir
```

Or generate from an xlsx workbook using the Go tool under `tools/xlsx2yml`:

```bash
go run ./tools/xlsx2yml                              # reads configs/example.xlsx -> configs/
go run ./tools/xlsx2yml -f my.xlsx -o out_dir        # custom input/output
go run ./tools/xlsx2yml --sheet Sheet2 --overwrite   # pick a sheet, overwrite existing yml
go run ./tools/xlsx2yml --schema-mapping             # also emit schemaMapping: { src_db: dest_user }
```

Required header columns (first row, case-insensitive): `src_host`, `src_port`, `src_database`, `src_username`, `src_password`, `dest_host`, `dest_port`, `dest_database`, `dest_username`, `dest_password`. Output files are named `NNN_<src_database>.yml` in row order; existing files are skipped unless `--overwrite` is set; duplicate rows abort the run.

**Linux / macOS: `run_batch.sh`**

```bash
bash run_batch.sh                       # default: ./configs
bash run_batch.sh path/to/cfg_dir       # custom directory
bash run_batch.sh configs dryrun        # read-only pre-flight, no migration / no object creation
```

**Windows: `run_batch.ps1`** (PowerShell 5.1+, built into Windows; keep the script next to `gomysql2pg.exe`)

```powershell
powershell -ExecutionPolicy Bypass -File .\run_batch.ps1
powershell -ExecutionPolicy Bypass -File .\run_batch.ps1 -ConfigDir my_cfgs
powershell -ExecutionPolicy Bypass -File .\run_batch.ps1 -Mode dryrun
```

**Script behavior**

1. Print a preview: `[i] file | src=host:port/db -> dest=user@host:port/db`.
2. Wait for the user to enter `yes` before running; any other input aborts (exit code 1).
3. Run `gomysql2pg --config <file>` sequentially; a single failure does NOT abort the rest.
4. All output is teed to the screen and to a batch log `batch-YYYYMMDD-HHMMSS.log`.
5. Print a final success / failed summary. **The exit code equals the number of failed configs** (0 on full success).

**Dryrun mode**

Pass `dryrun` as the second positional arg to `run_batch.sh` (or `-Mode dryrun` to `run_batch.ps1`) and the scripts will invoke `gomysql2pg --config <file> dryRun` for each yml. Three read-only checks run per config: source MySQL ping, destination ping, and whether a schema with the same name as `dest.username` exists on the destination. In this mode:

- No objects are created and no data is migrated; the interactive `yes` confirmation is skipped.
- The batch log is named `batch-dryrun-YYYYMMDD-HHMMSS.log`. Each failing config also writes `dryRunFailed.log` under `log/<timestamp>__src__to__dest/` (picked up by `check_log.sh` / `check_log.ps1`).
- If any config fails because the destination is missing the same-name schema, after the run the terminal (and batch log) appends a remediation SQL block, ready to copy and paste:

  ```
  === missing same-name schema(s) detected ===
  Run the following SQL on the destination database as a privileged user:

  create user admin2 with password '123456';
  create schema admin2 authorization admin2;

  (Replace '123456' with a real password before executing.)
  ```

  Run it on the destination as a privileged user, then re-run the batch. It is recommended to run a `dryrun` pass before the real `migrate` to catch connectivity / missing-schema problems in one shot.

### 4 View Migration Summary

After the entire database migration is completed, a migration summary will be generated to observe if there are any failed objects. By querying the migration log, the failed objects can be analyzed

```bash
+-------------------------+---------------------+-------------+----------+
|        SourceDb         |       DestDb        | MaxParallel | PageSize |
+-------------------------+---------------------+-------------+----------+
| 192.168.149.37-sourcedb | 192.168.149.33-test |     30      |  100000  |
+-------------------------+---------------------+-------------+----------+

+------------+----------------------------+----------------------------+-------------+---------------+
|Object      |         BeginTime          |          EndTime           |FailedTotal  |ElapsedTime    |
+------------+----------------------------+----------------------------+-------------+---------------+
|TableData   | 2023-07-11 12:23:55.584092 | 2023-07-11 12:28:44.105372 |6            |4m48.5212802s  |
|Sequence    | 2023-07-11 12:30:04.697570 | 2023-07-11 12:30:12.549534 |1            |7.8519647s     |
|Index       | 2023-07-11 12:30:12.549534 | 2023-07-11 12:33:45.312366 |5            |3m32.7628317s  |
|ForeignKey  | 2023-07-11 12:33:45.312366 | 2023-07-11 12:34:00.413767 |0            |15.1014013s    |
|View        | 2023-07-11 12:34:00.413767 | 2023-07-11 12:34:01.240472 |14           |826.705ms      |
|Trigger     | 2023-07-11 12:34:01.240472 | 2023-07-11 12:34:01.339078 |1            |98.6061ms      |
+------------+----------------------------+----------------------------+-------------+---------------+

Table Create finish elapsed time  5.0256021s
time="2023-07-11T12:34:01+08:00" level=info msg="All complete totalTime 10m30.1667987s\nThe Report Dir C:\\go\\src\\gomysql2pg\\2023_07_11_12_23_31" func=gomysql2pg/cmd.mysql2pg file="C:/go/src/gomysql2pg/cmd/root.go:207"

```

### 5 Compare Source and Target database

After migration finish you can compare source table and target database table rows,displayed failed table only

`gomysql2pg.exe --config your_file.yml compareDb`

```
e.g.
gomysql2pg.exe --config example.yml compareDb

on Linux and MacOS you can run
./gomysql2pg --config example.yml compareDb
```

```bash
Table Compare Result (Only Not Ok Displayed)
+-----------------------+------------+----------+-------------+------+
|Table                  |SourceRows  |DestRows  |DestIsExist  |isOk  |
+-----------------------+------------+----------+-------------+------+
|abc_testinfo           |7458        |0         |YES          |NO    |
|log1_qweharddiskweqaz  |0           |0         |NO           |NO    |
|abcdef_jkiu_button     |4           |0         |YES          |NO    |
|abcdrf_yuio            |5           |0         |YES          |NO    |
|zzz_ss_idcard          |56639       |0         |YES          |NO    |
|asdxz_uiop             |290497      |190497    |YES          |NO    |
|abcd_info              |1052258     |700000    |YES          |NO    |
+-----------------------+------------+----------+-------------+------+ 
INFO[0040] Table Compare finish elapsed time 11.307881434s 
```







## Other migration modes

In addition to migrating the entire database, the tool also supports the migration of some database objects, such as partial table structures, views, self increasing columns, indexes, and so on


#### 1 Full database migration

Migrate entire database table structure, row data, views, index constraints, and self increasing columns to target database

gomysql2pg.exe  --config file.yml

```
e.g.
gomysql2pg.exe --config example.yml
```

#### 2 Custom SQL Query Migration

only migrate some tables not entire database, and migrate the table structure and table data to the target database according to the custom query statement in file.yml

gomysql2pg.exe  --config file.yml -s

```
e.g.
gomysql2pg.exe  --config example.yml -s
```

#### 3 Migrate all table structures in the entire database

Create all table structure(only table metadata not row data) to  target database

gomysql2pg.exe  --config file.yml createTable -t

```
e.g.
gomysql2pg.exe  --config example.yml createTable -t
```

#### 4 Migrate the table structure of custom tables

Read custom tables from yml file and create target table 

gomysql2pg.exe  --config file.yml createTable -s -t

```
e.g.
gomysql2pg.exe  --config example.yml createTable -s -t
```


#### 5 Migrate full database table data

Only migrate the entire database table row data to the target database, only row data, not contain table structure

gomysql2pg.exe  --config file.yml onlyData
```
e.g.
gomysql2pg.exe  --config example.yml onlyData
```

#### 6 Migrate custom table data

Only migrate custom query SQL from yml file, only row data, not contain table structure

gomysql2pg.exe  --config file.yml onlyData -s

```
e.g.
gomysql2pg.exe  --config example.yml onlyData -s
```

#### 7 Migrate self increasing columns to the target sequence

Only migrate MySQL's autoincrement columns to target database sequences

gomysql2pg.exe  --config file.yml seqOnly

```
e.g.
gomysql2pg.exe  --config example.yml seqOnly
```

#### 8 Migrate index and primary key

Only migrate MySQL primary keys, indexes, and other objects to the target database

gomysql2pg.exe  --config file.yml idxOnly

```
e.g.
gomysql2pg.exe  --config example.yml idxOnly
```

#### 9 Migration View

Only migrate MySQL views to the target database

gomysql2pg.exe  --config file.yml viewOnly

```
e.g.
gomysql2pg.exe  --config example.yml viewOnly
```

#### 10 DryRun (read-only pre-flight)

Validate the environment without creating any objects or migrating any data. Useful before a real batch migration to filter out connectivity and missing-schema problems in one shot.

gomysql2pg.exe  --config file.yml dryRun

```
e.g.
gomysql2pg.exe  --config example.yml dryRun

on Linux and MacOS:
./gomysql2pg --config example.yml dryRun
```

Checks performed:

1. `SourcePing`: ping the source MySQL.
2. `DestPing`: ping the destination (driver auto-selected from `dest.dbType` — `postgres` or `opengauss`).
3. `SameNameSchema`: a schema with the same name as `dest.username` exists on the destination.

Exit code is 0 when all checks pass; 1 otherwise. On failure, details are written to `log/<timestamp>__src__to__dest/dryRunFailed.log` (compatible with `check_log.sh` / `check_log.ps1`).

If `SameNameSchema` reports `no schema named "<user>" in database "<db>"`, on the destination run as a privileged user:

```sql
create user <user> with password '123456';
create schema <user> authorization <user>;
```

Replace `<user>` with the `dest.username` from the yml and `'123456'` with a real password. For batch use, `bash run_batch.sh configs dryrun` / `run_batch.ps1 -Mode dryrun` print a ready-to-copy SQL list at the end (see "Dryrun mode" under section 3).
