## branch simple
- 1.建立对应的 configs.Config 传入，允许 json 配置对应的 cgroup name, resource, process limit
- 2.简单化 process, container, 从 go-docker 中抽象出对应的逻辑载体，
- 3.简单化 cgroup manager，不考虑 interface 接口形式，直接上手 struct，全部导出
- 4.需要实现 container state infomation json.strigify

需要用户确定 bundle 目录，该目录下需要有文件
```sh
config.json  // 在 configs.Config 中的序列化文件
/rootfs      // 可以使挂载 busybus 的文件夹
```