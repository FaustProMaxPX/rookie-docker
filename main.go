package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

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
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNET,
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

	// TODO: 一个失败的文件系统隔离方案，具体bug原因有待研究
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
	// TODO: 这里需要超管权限，sudo还不好使（逆天）
	// 切换根目录
	must(syscall.Chroot("/workspaces/container-fs"))
	must(syscall.Chdir("/"))
	// 这一步实际执行容器中需要运行的命令
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
