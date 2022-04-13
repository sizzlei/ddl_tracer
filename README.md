# DDL Tracer
DDL Tracer는 변경되는 컬럼 및 테이블 정보를 누락없이 타 부서와 공유하고, DBA 외 Admin 계정으로 접근하여 DDL 이 수행되는 것을 확인하기 위해 개발되었습니다. 

## File structure
---
DDL Tracer는 DB를 파일구조로 저장하여 비교 분석합니다. 
```
{서버 Alias}
|-> {Schema Name}
  |-> {Table Name}
    |-> {Column Name}:Hash Value(Column Definition)
```
> 모든 컬럼파일은 0byte 입니다

### ex)
```shell
$ cd test_table
$ ll
total 0
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:00 asset_id:4c6977efb62764a0a06aa6c94019ee2b
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:00 created_at:5bd2caaab1e46ffc3d03358cb263276a
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:30 hist_id:079fd672888c4bff38c63d7e1a55724d
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:00 hist_type:7a8e591f018e8ec509510d324d958459
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:00 id:07e09c63dc9139e5a152cbf8456f3b16
-rwxr-x--- 1 monitor monitor 0 Apr 11 04:00 qty:dda560ae10d8a70590752e91d2324fce
```
## Configure
---
```yaml
tracker:
  user: 
  password:  # 개발
  datapath: /data/ddl_tracer
  webhook_url: 
  tableddl : false

servers:
  - alias: testDB
    host: testdb.endpoint
    port: 3306
    db: ["test"]
```
### Tracker
- user : DB 접근 사용자 계정(모니터링 계정으로 초기설정합니다.)
- password : DB 접근 사용자 패스워드를 특정 키로 AES_ENCRYPT한 구문(암호화 과정은 하단부에 기술합니다.)
- datapath : 스키마 정의 파일이 저장되는 디렉터리
- webhook_url : DDL 발생시 Notifircation 채널
- tableddl : true/false 로 Noti시 테이블 스크립트를 포함하는지 여부입니다. 

### Servers
- alias : Noti시 해당 서버 Alias
- host : 접근 가능한 DB Endpoint(Read Replica)
- port : DB 접근 포트
- db : Trace가 적용될 스키마 대상 목록

## Usage
---
### Password Encrypt
Config에 패스워드를 넣기 위해 실행하는 Password Encrypt 기능입니다. 
```shell
./ddl_tracer -opt=ENCRYPT -password={패스워드}
INFO[0000] Regist tracer : DDL_Tracer Version : 0.1.0   
INFO[0000] Password Encrypt Complete : bPDLd0lssjqxyOPSfwgAMQ== 
```
> 암호화 키는 소스내에 존재합니다. `lib/crypto.go`

### Initialize
DDL Tracer를 초기화 하는것으로 작성된 서버 및 스키마에 대한 정보를 파일구조로 최초 저장합니다. 
```
./ddl_tracer -opt=init -conf=./config.yml
INFO[0000] Regist tracer : DDL_Tracer Version : 0.1.0   
INFO[0000] Config Load Success.                         
INFO[0000] Set Option init.                             
INFO[0000] Set Initialize testdb.port. 
INFO[0000] Setup Host Path : /data/ddl_trace/testdb
```
### Check
실제 스키마를 비교하는 옵션입니다. 
```
./ddl_tracer -opt=check -conf=./config.yml
```

- Initialize / Check 수행시에는 설정된 Webhook으로 알림이 발송됩니다. 
- Rename(Column, Table)은 추적이 불가능하며 Dropped / Added로 2건 발생합니다. 