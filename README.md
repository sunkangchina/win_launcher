# windows下同时打开多个exe，当退出第一个后其他的会退出

### 编译

~~~
go build -ldflags="-H=windowsgui" -o launcher.exe launcher.go
~~~
 
### 目录演示

项目根目录/

~~~
│
├── launcher.exe            
├── launcher.log            
│
├── project1/                 
│   └── project1.exe           
│
├── project2/               
│   └── project2.exe        
~~~