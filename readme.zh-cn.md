# 移动端命令行程序管理器(costrict-host)

## 目的

在移动端，可能有多个服务，如果完全由vscode扩展进行管理，会让扩展变得复杂。

使用一个独立的命令行程序管理器，管理多个CLI程序的下载、安装、启动、配置、监控、服务注册，可以大大简化vscode扩展的复杂度。

## 技术原理

costrict-host连接服务端，获取各移动端的业务定义文件，根据业务定义文件，下载需要的其它程序，并维护这些程序的生命周期。

## 整体方案

移动端业务描述文件格式：

```json
{
    "configuration": "1.0.0",   //配置文件格式的版本
    "platform": "windows",      //配置文件适用的平台
    "arch": "amd64",            //配置文件适用的平台
    "version": "1.2.0",         //配置版本(用于配置本身的更新)
    "services": [{              //需要costrict-host管理的客户端程序
        "name": "codebase-syncer",  //程序名称
        "versions": {
            "lowest": "1.0.1",      //最低版本，当前版本低于该版本，强制升级到该版本
            "highest": "1.2.3"      //最高版本，超过该版本不自动升级
        },
        "startup": "always",        //启动模式：always=常驻, once=运行一次, none=不自动运行
        "protocol": "http",         //服务对外接口协议
        "port": "8080"              //服务端口
    }, {
        "name": "codebase-indexer",
        "version": "1.1.1",
        "startup": "always",
        "protocol": "http",
        "port": "8081"
    }, {
        "name": "codebase-parser",
        "version": "1.1.1",
        "startup": "always",
        "protocol": "http",
        "port": "8082"
    }, {
        "name": "chisel",
        "version": "1.0.0",
        "startup": "none"
    }, {
        "name": "tunnel-client",
        "version": "1.0.0",
        "startup": "always"
    }, {
        "name": "cleaner",
        "version": "1.1.0",
        "startup": "once"
    }]
}
```

costrict-host启动后，从https://zgsm.sangfor.com/shenma/costrict-host/
