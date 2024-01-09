# dproject

A web system write in golang.

## 开发说明

本项目开发的时候使用了[fresh](https://github.com/gravityblast/fresh)这个工具，它可以监听文件改动并实时重启进程，请务必在开发之前全局安装该模块。

不管是本地开发还是生产环境，主要用 docker compose 来管理各个服务，所以在开发前请确保本机安装了 docker。

启动服务之前需要自定义环境变量，请直接参考[.env.example](./.env.example.dev)，根据里面的示例新建一个文件，如`.env.local.dev`，切记：**不要把本地开发用的环境变量文件提交到仓库**。

以上这些准备好之后，先安装依赖

```go
go mod download
```

然后运行

```bash
./dev.sh .env.local.dev
```

即可启动服务。

你也可以直接使用`docker compose`命令来启动服务，不过有一些配置文件需要提前生成，具体你可以阅读[dev.sh](./dev.sh)里面的代码。

## 测试

本项目写了单元测试和端到端测试，两种测试在启动之前都需要先启动服务（因为有些测试需要读写数据库）。

运行单元测试

```go
go test ./...
```

运行端到端测试

```go
go run ./e2e
```

尽管已经配置了 CI 跑自动化测试，还是建议每次推送之前手动运行一下测试，避免把报错的代码推送到远程仓库。

## 模拟数据

`./seed`模块专门用于批量生成模拟数据，以方便测试，具体使用方法请查看帮助

```
go run ./seed --help
```

## 国际化配置

本项目做了 i18n 配置，所有配置文件存放在`./i18n`里面，其中 TOML 文件都是由[go-i18n](https://github.com/nicksnyder/go-i18n/)这个工具生成的，请不要直接修改里面的内容。

新增翻译请直接修改 i18n 目录下的 go 文件里面的代码，改完之后在运行 i18n 相关命令，并添加各种语言的翻译内容，具体请参考[go-i18n的说明文档](https://github.com/nicksnyder/go-i18n/#command-goi18n)。
