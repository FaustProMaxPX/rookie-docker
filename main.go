package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// 容器进程的文件系统路径
const childFsPath = "/home/faust/develop/funny/rookie-docker/ubuntu-fs"

func main() {
	switch os.Args[1] {
	case "run":
		parent()
	case "child":
		child()
	default:
		panic("not supported operation")
	}
}

// 主机环境下要执行的操作
func parent() {
	// /proc/self/exe是当前运行程序的符号链接
	// 因此这行代码的意义就是在当前程序内再次启动自己，不过参数是child
	// 这一步实现了容器的隔离特性，
	// 现在child运行的程序的上下文不再是主机上下文而是父程序的上下文
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	// 这一步是创建命名空间的操作
	// 这里使用到了Linux的线程隔离机制
	// Linux提供了六种隔离机制：
	// 1. uts: 隔离主机名
	// 2. pid: 隔离进程PID
	// 3. user: 隔离用户
	// 4. mount: 隔离各个进程看到的挂载点视图
	// 5. network: 隔离网络
	// 6. ipc: 用来隔离 System V IPC 和 POSIX message queues

	// 默认情况下，容器内的所有信息都会被共享给主机，如果我们不希望某些信息被共享
	// 可以在Unshareflags中添加对应的flag

	// 这里需要注意，如果隐藏了用户，那么fork出来的子进程会使用nobody用户运行，没有权限进行chroot等操作
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

// 容器环境下要执行的操作
func child() {
	// 在默认情况下，线程会继承来自主机的挂载信息，因此这里我们需要做出修改

	// // 一个trick，因为PivotRoot要求交换的两个文件系统不属于同一棵文件树，
	// // 因此使用Mount来解决这个问题（不懂）
	// must(syscall.Mount("rootfs", "rootfs", "", syscall.MS_BIND, ""))
	// must(os.MkdirAll("rootfs/oldrootfs", 0700))
	// // 调用PivotRoot将位于'/'的当前目录移动到'rootfs/oldrootfs'
	// // 并将新的rootfs目录交换到'/'
	// must(syscall.PivotRoot("rootfs", "rootfs/oldrootfs"))
	// // PivotRoot调用完成后，容器内的'/'目录就会指向rootfs
	// must(os.Chdir("/"))
	syscall.Sethostname([]byte("container"))
	// 切换根目录，childFsPath类似容器中的镜像，当容器以某个镜像启动，实际上就是以某个文件系统启动
	must(syscall.Chroot(childFsPath))
	must(syscall.Chdir("/"))
	// proc文件夹是操作系统与进程共享信息的一种渠道，如果没有进行mount操作，OS内核就不知道自己还要跟这个进程共享信息
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	// 这一步实际执行容器中需要运行的命令
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	syscall.Unmount("proc", 0)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
