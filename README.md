# ArchiveDB





## archive demo

1. sync favorites weibo

enter  `cmd/dashboard/sync`

create and edit `.config.yaml`

sample

```yaml
uid: 123456
cookie: 'SINAGLOBAL=8xxxxxxx'
```

```sh
go build && ./sync
```

2. start web server

enter `cmd/dashboard/server`

```sh
go run main.go ../sync/.data
```


3. start web ui in dev mode

```sh
cd cmd/dashboard/ui
yarn && yarn start
```

![demo](cmd/dashboard/images/demo.png)

