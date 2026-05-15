# gomysql2pg

![logo.png](image/logo.png)

## 一、工具特性以及环境要求
### 1.1 功能特性

支持MySQL数据库一键迁移到postgresql内核类型的目标数据库，如postgresql数据库、海量数据库vastbase、华为GaussDB、电信telepg、人大金仓Kingbase V8R6等

- 无需繁琐部署，开箱即用，小巧轻量化
- 支持批量迁移多对数据库
- 在线迁移MySQL到目标数据库的表、视图、索引、外键、自增列等对象
- 多个goroutine并发迁移数据，充分利用CPU多核性能
- 支持迁移源库部分表功能
- 记录迁移日志，转储表、视图等DDL对象创建失败的sql语句
- 一键迁移MySQL到postgresql，方便快捷，轻松使用


![gomysql2pg_en_struct.png](image/gomysql2pg_en_struct.png)

需要知道的是MySQL跟其他基于pgsql内核或者协议的国产数据库，无论是数据库架构还是在表结构以及列类型，自增列实现方式，函数，存储过程等多个对象存在很多差异，如果通过传统的SQL备份文件导入到目标数据库，这势必是不可取也是最没效率的一种方式

通过此工具，可以降低异构数据库之间开发的人工成本，并尽可能的将MySQL绝大多数的,表结构，自增列，视图，行数据等多对象并发迁移到目标数据库，尽可能的适配目标类型，提供了全库级别，表对象级，自定义查询SQL等多种迁移方式

gomysql2pg根据MySQL内部的数据字典获取到数据库各个对象的定义和属性信息，并适配到目标数据库

数据迁移，本质上就是把表数据从一个数据库"搬家"到另一个数据库，其中主要涉及读表，传输，写表，而读表和写表占据迁移绝大多数时间

### 1.2 环境要求
在运行的客户端PC需要同时能连通源端MySQL数据库以及目标数据库

支持Windows、Linux、MacOS

### 1.3 如何安装

解压之后即可运行此工具

若在Linux环境下请使用tar解压，例如：


`[root@localhost opt]# tar -zxvf gomysql2pg-linux64-0.1.7.tar.gz`

## 二、使用方法

以下为Windows平台示例，其余操作系统命令行参数一样

`注意:`在`Windows`系统请在`CMD`运行此工具，如果是在`MacOS`或者`Linux`系统，请在有读写权限的目录运行

### 2.1 编辑yml配置文件

编辑`example.cfg`文件，分别输入源库跟目标数据库信息

```yaml
src:
  host: 192.168.1.3
  port: 3306
  database: test
  username: root
  password: 11111
dest:
  dbType: Gauss # 如果使用的是openGauss类型(openGauss 5.0.2测试通过)请一定要添加此行，非openGauss类型一定要注释本行
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
  - '*_cswysk'

```

pageSize: 分页查询每页的记录数
```
e.g.
pageSize:100000
SELECT t.* FROM (SELECT id FROM test  ORDER BY id LIMIT 0, 100000) temp LEFT JOIN test t ON temp.id = t.id;
```
- maxParallel: 最大能同时运行goroutine的并发数

- tables: 自定义迁移的表以及自定义查询源表，按yml格式缩进

- exclude: 不需要迁移的表，按yml格式缩进，目前新增支持通配符星号(\*)，例如test*

- charInLength: 如果是true，varchar类型存储的是字符长度而不是字节，所以仅兼容部分数据库(例如海量)

- useNvarchar2: 如果是true，目标数据库使用nvarchar2类型(例如GaussDB)

- Distributed: 默认为false即非分布式数据库，如果是分布式数据库就写true，如GaussDB 8.1.3，在增加主键之前，先更改表分布列为主键的列，随后再增加主键

### 2.2 全库迁移

>`注意:` 如果目标数据库是opengauss或者是GaussDB，有2种方案可以让类型varchar按字符作为长度(即跟MySQL一样)
>> 方案1:调整配置文件`example.yml`,确保useNvarchar2字段值为true，这将在目标数据库使用nvarchar2类型(gauss系列此类型支持按字符长度存储而不是字节)
> 
>> 方案2:创建数据库指定兼容模式为MySQL或者PG，例如create database test owner test DBCOMPATIBILITY='B'或者create database test owner test DBCOMPATIBILITY='PG'



迁移全库表结构、行数据，视图、索引约束、自增列等对象

gomysql2pg.exe  --config 配置文件
```
示例
gomysql2pg.exe --config example.yml

如果是Linux或者macOS请在终端运行
./gomysql2pg --config example.yml
```

### 2.3 批量迁移（多库一次执行）

适用场景：一次迁移多个库 / 多对源-目标。仓库根目录提供两份脚本：`run_batch.sh`（Linux / macOS）和 `run_batch.ps1`（Windows），二者行为、提示、日志格式、退出码完全一致。

1 批量生成配置文件

编辑 `configs/` 目录下的Excel文件`example.xlsx`在各个表头下方输入正确的连接信息，`src`开头的即源库，`dest`开头的即目标库
必需的表头列（首行，大小写不敏感）：`src_host`、`src_port`、`src_database`、`src_username`、`src_password`、`dest_host`、`dest_port`、`dest_database`、`dest_username`、`dest_password`。输出文件命名为 `NNN_<src_database>.yml` 按行顺序编号；已存在的文件默认跳过（加 `--overwrite` 覆盖）；发现重复行会直接中止。

使用 `tools/xlsx2yml` 这个 Go 工具从 xlsx 工作簿批量生成：

```bash
开发环境运行方式
go run ./tools/xlsx2yml                              # 默认读取 configs/example.xlsx，输出到 configs/
go run ./tools/xlsx2yml -f my.xlsx -o out_dir        # 指定输入文件与输出目录
go run ./tools/xlsx2yml --sheet Sheet2 --overwrite   # 指定 sheet、覆盖已存在的 yml
go run ./tools/xlsx2yml --schema-mapping             # 同时生成 schemaMapping: { src_db: dest_user }
或者
二进制文件运行方式
xlsx2yml.exe                              # 默认读取 configs/example.xlsx，输出到 configs/
xlsx2yml.exe -f my.xlsx -o out_dir        # 指定输入文件与输出目录
xlsx2yml.exe --sheet Sheet2 --overwrite   # 指定 sheet、覆盖已存在的 yml
xlsx2yml.exe --schema-mapping             # 同时生成 schemaMapping: { src_db: dest_user }
```

以上执行后会在`configs`目录下生成多个`yml`格式的配置文件
```
configs/
├── 002_db_1.yml
├── 002_db_2.yml
└── 003_db_3.yml
```

2 迁移预检(dryrun)
**脚本简介**

`run_batch.sh`或者`run_batch.ps1`配合参数`dryrun`，脚本会改成对每个 yml 调用 `gomysql2pg --config <file> dryRun`，
仅做三项只读检查：源 MySQL 连接 ping、目的库连接 ping、目的库是否存在与 `dest.username` 同名的 schema。
建议在正式 `migrate` 前先跑一次 `dryrun` 把环境问题（连通性、模式缺失）一次性筛掉。

**脚本行为**

1. 不创建任何对象、不迁移任何数据，跳过 `yes` 交互确认；
2. 批次日志改名为 `batch-dryrun-YYYYMMDD-HHMMSS.log`；单条失败时还会在 `log/<时间>__src__to__dest/` 下生成 `dryRunFailed.log`，可被 `check_log.sh` / `check_log.ps1` 捕获；
3. 当存在"目的端缺少同名 schema"类失败时，全部跑完后终端最后会追加一段补救 SQL 提示（也同步进 batch 日志），可直接拷贝到目的库执行：
  ```
  === missing same-name schema(s) detected ===
  Run the following SQL on the destination database as a privileged user:

  create user admin2 with password '123456';
  create schema admin2 authorization admin2;

  (Replace '123456' with a real password before executing.)
  ```

执行方式
**Linux / macOS：`run_batch.sh`**

```bash
bash run_batch.sh configs dryrun        # 仅做只读预检，不迁移、不创建对象
```

**Windows：`run_batch.ps1`**（非cmd，需使用PowerShell 5.1+，系统自带；脚本默认与 `gomysql2pg.exe` 在同一目录）

```powershell
开始菜单以管理员运行powershell,执行如下授权powershell脚本执行权限（必须项）
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
Get-ExecutionPolicy -List
```

```powershell
工具解压之后使用powershell进入到正确的目录比如cd C:\db_tool\gomysql2pg-win-x64-v0.2.10
运行预检脚本
.\run_batch.ps1 configs dryrun
```

预检通过示例，如有错误请查看对应目录的配置文件
```powershell
=== batch done 2026-05-15 15:00:51 ===
success: 3
  OK   C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\01_a.yml
  OK   C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\02_b.yml
  OK   C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\03_c.yml
failed:  0
PS C:\db_tool\gomysql2pg-win-x64-v0.2.10>
```


3 正式批量迁移
`run_batch.sh`或者`run_batch.ps1`执行多个库批量迁移
**脚本行为**

1. 先列出预览：`[i] file | src=host:port/db -> dest=user@host:port/db`；
2. 等待用户输入 `yes` 才开始执行，输入其它内容则放弃并退出（退出码 1）；
3. 顺序逐个调用 `gomysql2pg --config <file>`，单条失败不中断后续；
4. 全程输出同时打到屏幕和批次日志 `batch-YYYYMMDD-HHMMSS.log`；
5. 全部结束后打印 success / failed 列表，**退出码等于失败条数**（全部成功为 0）。


**Linux / macOS：`run_batch.sh`**

脚本参数
```bash
bash run_batch.sh                       # 默认读取当前configs目录
bash run_batch.sh path/to/cfg_dir       # 指定其它目录
```
脚本示例
```bash
bash run_batch.sh
Found 3 config(s) to migrate:
[1] configs/01_a.yml  |  src=192.168.149.86:3306/tenantdb_a101  ->  dest=admin@192.168.149.95:5432/test
[2] configs/02_b.yml  |  src=192.168.149.86:3306/tenantdb_a102  ->  dest=admin2@192.168.149.95:5432/test
[3] configs/03_c.yml  |  src=192.168.149.86:3306/tenantdb_a999  ->  dest=admin3@192.168.149.95:5432/test

Proceed with these 3 migration(s)? [yes/no]: -> 在这里输入yes或者no选择继续或者退出
```


**Windows：`run_batch.ps1`**（PowerShell 5.1+，系统自带；脚本须与 `gomysql2pg.exe` 在同一目录）
脚本参数
```powershell
powershell -ExecutionPolicy Bypass -File .\run_batch.ps1
powershell -ExecutionPolicy Bypass -File .\run_batch.ps1 -ConfigDir my_cfgs
```

脚本示例
开始菜单以管理员运行powershell,执行如下授权powershell脚本执行权限（必须项）
```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
Get-ExecutionPolicy -List
```

```powershell
PS C:\db_tool\gomysql2pg-win-x64-v0.2.10> .\run_batch.ps1
Found 3 config(s) to migrate:
  [1] C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\01_a.yml  |  src=192.168.1.86:3306/db_a101  ->  dest=admin@192.168.1.95:5432/test
  [2] C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\02_b.yml  |  src=192.168.1.86:3306/db_a102  ->  dest=admin2@192.168.1.95:5432/test
  [3] C:\db_tool\gomysql2pg-win-x64-v0.2.10\configs\03_c.yml  |  src=192.168.1.86:3306/db_a999  ->  dest=admin3@192.168.1.95:5432/test

Proceed with these 3 migration(s)? [yes/no]: -> 在这里输入yes或者no选择继续或者退出
```









### 2.4 查看迁移摘要

全库迁移完成之后会生成迁移摘要，观察下是否有失败的对象，通过查询迁移日志可对迁移失败的对象进行分析

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

### 2.4 比对数据库

迁移完之后比对源库和目标库，查看是否有迁移数据失败的表

`windows使用:gomysql2pg.exe --config your_file.yml compareDb`

```
e.g.
gomysql2pg.exe --config example.yml compareDb

在Linux，MacOS使用示例如下
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




### 2.4 其他迁移模式

除了迁移全库之外，工具还支持迁移部分数据库对象，如部分表结构，视图，自增列，索引等对象


#### 2.4.1 全库迁移

迁移全库表结构、行数据，视图、索引约束、自增列等对象

gomysql2pg.exe  --config 配置文件

```
示例
gomysql2pg.exe --config example.yml
```

#### 2.4.2 自定义SQL查询迁移

不迁移全库数据，只迁移部分表，根据配置文件中自定义查询语句迁移表结构和表数据到目标库

gomysql2pg.exe  --config 配置文件 -s

```
示例
gomysql2pg.exe  --config example.yml -s
```

#### 2.4.3 迁移全库所有表结构

仅在目标库创建所有表的表结构

gomysql2pg.exe  --config 配置文件 createTable -t

```
示例
gomysql2pg.exe  --config example.yml createTable -t
```

#### 2.4.4 迁移自定义表的表结构

仅在目标库创建自定义的表

gomysql2pg.exe  --config 配置文件 createTable -s -t

```
示例
gomysql2pg.exe  --config example.yml createTable -s -t
```


#### 2.4.5 迁移全库表数据

只迁移全库表行数据到目标库，仅行数据，不包括表结构

gomysql2pg.exe  --config 配置文件 onlyData
```
示例
gomysql2pg.exe  --config example.yml onlyData
```

#### 2.4.6 迁移自定义表数据

只迁移yml配置文件中自定义查询sql，仅行数据，不包括表结构

gomysql2pg.exe  --config 配置文件 onlyData -s

```
示例
gomysql2pg.exe  --config example.yml onlyData -s
```

#### 2.4.7 迁移自增列到目标序列形式

只迁移MySQL的自增列转换为目标数据库序列

gomysql2pg.exe  --config 配置文件 seqOnly

```
示例
gomysql2pg.exe  --config example.yml seqOnly
```

#### 2.4.8 迁移索引等约束

只迁移MySQL的主键、索引这类对象到目标数据库

gomysql2pg.exe  --config 配置文件 idxOnly

```
示例
gomysql2pg.exe  --config example.yml idxOnly
```

#### 2.4.9 迁移视图

只迁移MySQL的视图到目标数据库

gomysql2pg.exe  --config 配置文件 viewOnly

```
示例
gomysql2pg.exe  --config example.yml viewOnly
```

#### 2.4.10 预检（dryRun）

仅做只读预检，不创建任何对象，也不迁移任何数据。适用于在批量正式迁移前一次性筛掉连通性和目的端模式缺失等环境问题。

gomysql2pg.exe  --config 配置文件 dryRun

```
示例
gomysql2pg.exe  --config example.yml dryRun

Linux / macOS:
./gomysql2pg --config example.yml dryRun
```

检查项：

1. `SourcePing`：ping 源 MySQL；
2. `DestPing`：ping 目的库（自动按 `dest.dbType` 选 `postgres` 或 `opengauss` 驱动）；
3. `SameNameSchema`：目的库是否存在与 `dest.username` 同名的 schema。

退出码：全部通过为 0；任一失败为 1，并把失败明细写入 `log/<时间>__src__to__dest/dryRunFailed.log`（与 `check_log.sh` / `check_log.ps1` 兼容）。

如果 `SameNameSchema` 报 `no schema named "<user>" in database "<db>"`，请到目的库以特权用户执行：

```sql
create user <user> with password '123456';
create schema <user> authorization <user>;
```

把 `<user>` 替换成 yml 里的 `dest.username`，并把 `'123456'` 换成真实密码。建议结合 `bash run_batch.sh configs dryrun` / `run_batch.ps1 -Mode dryrun` 批量跑一次，脚本会在终端尾部直接给出可拷贝执行的 SQL 列表（详见 2.3 节"预检模式"）。