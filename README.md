# ArchiveDB


![workflow](https://github.com/jialeicui/archivedb/actions/workflows/go.yml/badge.svg)[![Go Report Card](https://goreportcard.com/badge/github.com/sincaw/archivedb)](https://goreportcard.com/report/github.com/sincaw/archivedb)


## archive demo

### run

* Enter `cmd/dashboard` folder
* copy and edit config
```shell
cp server/.config.yaml.example server/.config.yaml
vim server/.config.yaml
```
* run
```shell
make && make run
```

### start web ui in dev mode

```sh
cd ui
yarn && yarn start
```

