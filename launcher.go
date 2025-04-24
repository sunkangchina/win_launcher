package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Windows API 常量
const (
	SW_HIDE           = 0
	SW_RESTORE        = 9
	CREATE_NO_WINDOW  = 0x08000000
	PROCESS_TERMINATE = 0x0001
)

// 程序配置常量
const (
	OpticsProgram    = "project1/project1.exe"    // 主程序，退出时会关闭其他程序
	ProtectedProgram = "project2/project2.exe" // 从属程序
)

// Windows API 函数
var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	findWindow       = user32.NewProc("FindWindowW")
	showWindow       = user32.NewProc("ShowWindow")
	setForegroundWin = user32.NewProc("SetForegroundWindow")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	openProcess      = kernel32.NewProc("OpenProcess")
	terminateProcess = kernel32.NewProc("TerminateProcess")
	closeHandle      = kernel32.NewProc("CloseHandle")
)

// 日志文件路径
const logFilePath = "launcher.log"

// 全局变量存储进程信息
var (
	processes = make(map[string]*os.Process) // 存储所有启动的进程
)

func main() {
	// 初始化日志
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// 隐藏当前窗口（如果存在）
	hideConsoleWindow()

	// 获取当前程序所在目录
	currentDir, err := getCurrentDir()
	if err != nil {
		log.Println("获取当前目录失败:", err)
		return
	}

	// 需要启动的程序配置
	programs := []struct {
		name      string
		path      string
		showIfRun bool
	}{
		{"project1", OpticsProgram, true},  // 主程序 project1 是名称
		{"project2", ProtectedProgram, false}, // 从属程序
	}

	// 启动所有程序
	for _, prog := range programs {
		absPath := filepath.Join(currentDir, prog.path)
		processProgram(prog.name, absPath, prog.showIfRun)
	}

	// 启动监控goroutine
	go monitorMainProgram()

	// 记录完成信息
	log.Println("启动流程完成")

	// 保持主程序运行
	select {}
}

func monitorMainProgram() {
	mainExe := filepath.Base(OpticsProgram)
	for {
		if !isProcessRunning(mainExe) {
			log.Println("主程序已退出，正在关闭其他程序...")
			terminateAllPrograms()
			os.Exit(0)
		}
		time.Sleep(1 * time.Second)
	}
}

func terminateAllPrograms() {
	// 不终止主程序（因为它已经退出了）
	for name, proc := range processes {
		if name != filepath.Base(OpticsProgram) {
			terminateProcessByPID(proc.Pid)
			log.Printf("已终止程序: %s\n", name)
		}
	}
}

func terminateProcessByPID(pid int) {
	hProcess, _, _ := openProcess.Call(
		PROCESS_TERMINATE,
		0,
		uintptr(pid),
	)
	if hProcess != 0 {
		terminateProcess.Call(hProcess, 0)
		closeHandle.Call(hProcess)
	} else {
		// 如果无法通过PID终止，尝试通过进程名终止
		exeName := getProcessNameByPID(pid)
		if exeName != "" {
			cmd := exec.Command("taskkill", "/IM", exeName, "/F")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Run()
		}
	}
}

func getProcessNameByPID(pid int) string {
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "Name")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) >= 2 {
		return strings.TrimSpace(lines[1])
	}
	return ""
}

func processProgram(name, path string, showIfRun bool) {
	log.Printf("处理程序: %s (%s)\n", name, path)

	// 1. 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("错误: %s 文件不存在\n", name)
		return
	}

	// 2. 检查是否已在运行
	exeName := filepath.Base(path)
	if isProcessRunning(exeName) {
		log.Printf("%s 已在运行\n", name)
		if showIfRun {
			if err := activateWindow(exeName); err != nil {
				log.Printf("窗口激活失败: %v\n", err)
			} else {
				log.Printf("已激活 %s 窗口\n", name)
			}
		}
		return
	}

	// 3. 启动程序
	if proc, err := launchProgram(path); err != nil {
		log.Printf("启动 %s 失败: %v\n", name, err)
	} else {
		log.Printf("%s 启动成功\n", name)
		// 保存进程对象
		processes[exeName] = proc
	}
} 

func launchProgram(path string) (*os.Process, error) {
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	
	// 使用更可靠的隐藏窗口方法
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: CREATE_NO_WINDOW,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 验证进程是否真的启动
	time.Sleep(500 * time.Millisecond)
	if !isProcessRunning(filepath.Base(path)) {
		return nil, fmt.Errorf("进程启动后未检测到")
	}

	return cmd.Process, nil
}

// 其他函数保持不变...
func getCurrentDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func isProcessRunning(exeName string) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", exeName))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, _ := cmd.Output()
	return strings.Contains(string(output), exeName)
}

func activateWindow(exeName string) error {
	baseName := strings.TrimSuffix(exeName, filepath.Ext(exeName))
	titlePtr, _ := syscall.UTF16PtrFromString(baseName)
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return fmt.Errorf("窗口未找到")
	}
	showWindow.Call(hwnd, SW_RESTORE)
	setForegroundWin.Call(hwnd)
	return nil
}

func hideConsoleWindow() {
	consoleWindow, _, _ := getConsoleWindow.Call()
	if consoleWindow != 0 {
		showWindow.Call(consoleWindow, SW_HIDE)
	}
}