## windows 开发
参见 docker-compose.yml 
将文件导入 golang:1.14 容器中，可以设置对应的环境变量
```env
CGO_ENABLED: 0
GOOS: linux
GOARCH: amd64
```
```shell
go env -w CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go env -w CGO_ENABLED=1 GOOS=windows GOARCH=amd64
```
在容器中进行
```sh
go build main.go
./main --flag
```
学习的项目
- [containerd](https://github.com/containerd/containerd)
- [runc](https://github.com/opencontainers/runc)
- [go-docker](https://github.com/pibigstar/go-docker/)


## syscall
现阶段，只考虑 [linux docker](http://docscn.studygolang.com/pkg/syscall/#SysProcAttr)
 针对 windows, linux, go 实现了不同的底层接口，通过环境变量，引入对应的标准库

## cgroups Create a new cgroup

This creates a new cgroup using a static path for all subsystems under `/test`.

* /sys/fs/cgroup/cpu/test
* /sys/fs/cgroup/memory/test
* etc....

It uses a single hierarchy and specifies cpu shares as a resource constraint and
uses the v1 implementation of cgroups.

```go
shares := uint64(100)
control, err := cgroups.New(cgroups.V1, cgroups.StaticPath("/test"), &specs.LinuxResources{
    CPU: &specs.CPU{
        Shares: &shares,
    },
})
defer control.Delete()
```


## namespace
- uts: 用来隔离主机名
- pid：用来隔离进程PID号的
- user: 用来隔离用户的
- mount：用来隔离各个进程看到的挂载点视图
- network: 用来隔离网络
- ipc：用来隔离System V IPC 和 POSIX message queues
```go
// 对应的容器的隔离属性和 uid, gid
cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUSER |
            syscall.CLONE_NEWNET,
        // User ID mappings for user namespaces.
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 1,
				HostID:      0,
				Size:        1,
			},
        },
        // Group ID mappings for user namespaces.
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 1,
				HostID:      0,
				Size:        1,
			},
		},
	}
```
但是，这里没有提及，如何动态启动不同的命令，而不是直接进入 shell 环境，允许用户的自定义 cmd 入口。

抽象一个创建容器的过程，需要创建一个进程就可以了，根据实际情况，隔离 CloneFlags, 设置 uid, gid。


## cgroup
Linux Cgroup提供了对一组进程及子进程的资源限制，控制和统计的能力，这些资源包括CPU，内存，存储，网络等。

Cgroup完成资源限制主要通过下面三个组件

- cgroup: 是对进程分组管理的一种机制
- subsystem: 是一组资源控制的模块
- hierarchy: 把一组cgroup串成一个树状结构(可让其实现继承)

- cgroup.clone_children：cpuset的subsystem会读取该文件，如果该文件里面的值为1的话，那么子cgroup将会继承父cgroup的cpuset配置
- cgroup.procs：记录了树中当前节点cgroup中的进程组ID
- task: 标识该cgroup下的进程ID，如果将某个进程的ID写到该文件中，那么便会将该进程加入到当前的cgroup中。

/proc/self/exe 指向当前正在运行的可执行文件的路径

1.创建 subsystem 中的需要控制的资源文件 cpu, memory, network
2.在 /sys/fs/cgroup 中构建项目的专用文件路径 /sys/fs/cgroup/godocker/:containerId/memory 等等
3.通过写入文件的方式，控制 cgroup 的资源。


## read write layer
默认全局定义的的存放 image 的目录为：
/root/busybox.tar
解压为
/root/busybox

writelayer 为
/root/writelayer/:containerName

挂载点为
/root/mnt

挂载目录为 
dirs := fmt.Sprintf("dirs=%s:%s", writeLayPath, imagePath)
mount", "-t", "aufs", "-o", dirs, "none", mntPath

对应创建的 volume 为 out-path:in-path
需要挂载在 mntPath := MntPath+containerName+in-path

但是 pivoRoot 那里怎么确定需要重新挂载的 root  节点呢？

把 write layer container iit layer 和相 关镜
像的 layers mount mnt 目录下，然后把这个 mnt 目录作为容器启动的根目录
CreateReadOnlyLayer 函数新建 busybox 文件夹，将 busybox .tar 解压到 busybox 目录下，
作为容器的只读层。
CreateWriteLayer 函数创建了 个名为 writeLayer 的文件夹，作为容器唯 的可写层。
。在 CreateMountPoint 函数中，首先创建了 nt 文件夹，作为挂载点，然后把 writeLayer
目录 busybox mount mnt 目录下。
mnt
最后 NewParentProcess 函数中将容器使用的宿主机目录 root/busybox 替换成／root/


在容器退出 的时候 删除 Write Layer DeleteWorkSpace 函数
包括 DeleteMountPoint DeleteWriteLayer
。首先，在 DeleteMountPoint 函数中 umountmnt 目录
。然后，删除 mnt 目录。
。最后，在 DeleteWriteLayer 函数中删除 writeLayer 文件夹 这样容器对文件系统的更改
就都己经抹去了。


## volumes
普通的卷挂载是基本同 cgroup 的挂载读写层，只读层
```go
	// 将宿主机上关于容器的读写层和只读层挂载到 /root/mnt/容器名 里
	writeLayPath := path.Join(common.RootPath, common.WriteLayer, containerName)
	imagePath := path.Join(common.RootPath, imageName)
	dirs := fmt.Sprintf("dirs=%s:%s", writeLayPath, imagePath)
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntPath)
```
```go
    // 把宿主机文件目录挂载到容器挂载点中
    dirs := fmt.Sprintf("dirs=%s", parentPath)
    cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumePath)
```

## image
需要生成 只读层 的文件 tar 文件。
将 mntPath 打包即可

## container
container 是一个进程与相关的文件，需要管理 cmd.Start() 对应的 Pid 进程，
和对应的文件信息。
那么删除一个 docker 时，是否需要删除对应的 task 和对应的 limit 设置呢？
可以不处理，但是，会产生很多冗余的数据，会污染新生成的数据吗？

Because containers are spawned in a two step process you will need a binary that
will be executed as the init process for the container. In libcontainer, we use
the current binary (/proc/self/exe) to be executed as the init process, and use
arg "init", we call the first step process "bootstrap", so you always need a "init"
function as the entry of "bootstrap".

生成 parent, child socket pair, 对应 init 命令。
内部通过变量  _LIBCONTAINER_INITPIPE 等环境变量和
通过 parent, child 管道写入 pid cmd 的方式启动进程。


## log
不管是否输出 tty 都可以写入到一个 file 中，需要固定对应的 container 的日志文件路径

## network


## bridge



### 流程分析，代码结构分析
使用 cli 包，生成一个 cli 程序，
注册对应的 command 。如果需要注册子命令的话，需要使用 SubCommand。
需要学习 cli 的 Action ctx.Args() 相关接口函数，如果得到名字，各种 flags 。

可以依循 docker 命令工具，划分 image, container, network, volume 
同时 run 命令可以额外指定对应的 flag 绑定 network, volume

Run 函数
- 1.生成 parent process
- 2.记录 container 信息
- 3.生成项目的 cgroups manager，并对这个 parent process 进行 subsystem 资源控制
- 4.针对 net 进行网络连接
- 5.对容器进行初始化命令启动
- 6.如果 rm 那么，清除容器相关信息资源。




