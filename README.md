# windows下同时打开多个exe，当退出第一个后其他的会退出

### 编译

~~~
go build -ldflags="-H=windowsgui" -o launcher.exe launcher.go
~~~
 
### 目录演示

项目根目录/
│
├── launcher.exe            # 编译后的启动管理器程序
├── launcher.log            # 程序运行日志文件
│
├── project1/                 # 光学分析工具目录
│   └── project1.exe          # 光学分析工具主程序
│
├── project2/              # 保护系统目录
│   └── project2.exe       # 保护系统程序 