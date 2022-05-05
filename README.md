# ArchiveDB


![workflow](https://github.com/jialeicui/archivedb/actions/workflows/go.yml/badge.svg)


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

![demo](cmd/dashboard/images/demo.png)

favicon from : https://www.iconfont.cn/ outline-archive-tick
