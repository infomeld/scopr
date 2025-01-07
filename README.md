# Scopr

Scopr 主要用于列出赏金目标。支持单一项目、多个项目或者所有项目的资产查询，也可对项目进行新增资产监控。
目前支持 hackerone、bugcrowd、 intigriti

#### install

```
    go install github.com/infomeld/scopr/cmd/scopr@latest
```

#### 文件配置

配置文件位于 ～/.config/scopr/config.yaml
可自行根据情况配置，如有私密项目，需要配置 Private 中的 ApiToken 等信息

```
Bugcrowd:
    Enable: false
    Concurrency: 15
    AssetType: []  # 需要收集的资产类型
HackerOne:
    Enable: true
    Concurrency: 200  # 最高并发，获取所有项目时，并发量不宜太高
    AssetType:
        - DOMAIN
        - URL
        - WILDCARD
        - API
        - OTHER
        - GOOGLE_PLAY_APP_ID
        - APPLE_STORE_APP_ID
    Private:
        Enable: true
        ApiToken: {apitoken}
        ApiName: {name}
Intigriti:
    Enable: false
    Concurrency: 50
    AssetType: []
Black:
    - .gov
    - .edu
    - .json
    - .[0-9.]+$
    - github.com/
DingTalk:  # 钉钉通知 api key
    AppKey: ""
    AppSecret: ""
EnableProxy: false

```
