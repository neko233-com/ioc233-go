# ioc233-go

一个轻量级的 Go 语言依赖注入（IOC）容器库。

## 特性

- 🚀 **简单易用**：提供简洁的 API，易于集成到现有项目
- 🔧 **自动依赖注入**：支持通过结构体标签自动注入依赖
- 📝 **多种注入方式**：支持按类型注入、按名称注入、可选注入
- 🔄 **生命周期管理**：支持对象初始化完成后的回调
- 🎯 **类型安全**：使用 Go 泛型提供类型安全的对象获取
- 📊 **可扩展日志**：支持自定义日志实现

## 安装

```bash
go get github.com/neko233-com/ioc233-go
```

## 项目结构

项目采用类似 Java 的目录结构，将核心代码和测试代码分离：

```
ioc233-go/
├── ioc233/          # 核心代码目录（类似 Java 的 src/）
│   ├── ioc.go       # IOC 容器核心实现
│   ├── iobject.go   # 生命周期接口
│   ├── logger.go    # 日志实现
│   └── field_creator.go  # 字段默认值提供器
├── tests/           # 测试代码目录（类似 Java 的 test/）
│   └── ioc_test.go  # 单元测试
└── README.md        # 项目文档
```

导入时使用：

```go
import "github.com/neko233-com/ioc233-go/ioc233"
```

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/neko233-com/ioc233-go/ioc233"
)

// 定义服务接口
type UserService interface {
    GetUser(id int) string
}

// 实现服务
type UserServiceImpl struct {
    // 自动注入其他依赖
    // 注意：这里只是示例，实际使用时需要根据你的需求定义依赖
}

func (s *UserServiceImpl) GetUser(id int) string {
    return "User"
}

// 实现生命周期接口（可选）
func (s *UserServiceImpl) OnInjectComplete() {
    // 依赖注入完成后的初始化逻辑
}

func main() {
    // 注册服务
    container := ioc233.Instance()
    container.Provide(&UserServiceImpl{})
    
    // 启动容器，执行依赖注入
    if err := container.StartUp(); err != nil {
        panic(err)
    }
    
    // 获取服务
    service := ioc233.GetObjectByType[UserService]()
    user := service.GetUser(1)
    fmt.Println(user)
}
```

## 依赖注入方式

### 1. 按类型自动注入（必须）

使用 `autowire:"true"` 标签，容器会自动查找匹配类型的实现：

```go
type ServiceA struct {
    ServiceB *ServiceB `autowire:"true"`
}
```

如果找不到匹配的实现，会记录错误。

### 2. 按类型自动注入（可选）

使用 `autowire:"false"` 标签，如果找不到匹配的实现，字段保持为 `nil`：

```go
type ServiceA struct {
    OptionalService *OptionalService `autowire:"false"`
}
```

### 3. 按名称注入

使用 `autowire:"BeanName"` 指定要注入的 bean 名称：

```go
type ServiceA struct {
    ServiceB *ServiceB `autowire:"MyServiceB"`
}
```

### 4. 接口注入

容器会自动查找实现了接口的具体类型：

```go
type ServiceA struct {
    UserService UserService `autowire:"true"`
}
```

如果有多个实现，会注入第一个找到的，并记录警告。

## 注册对象

### 按类型注册（自动命名）

```go
container := ioc233.Instance()
container.Provide(&MyService{})
// bean 名称自动使用结构体名称 "MyService"
```

### 按名称注册

```go
container := ioc233.Instance()
err := container.ProvideByName("MyService", &MyService{})
if err != nil {
    // 处理错误（如名称重复）
}
```

## 生命周期回调

ioc233-go 提供了完整的生命周期回调机制，支持在对象的不同阶段执行自定义逻辑：

### 1. IProvideAfter - 注册后回调

对象注册到容器后立即调用：

```go
type MyService struct {
    Dep *Dependency `autowire:"true"`
}

func (s *MyService) OnProvideAfter() {
    // 对象已注册到容器，但依赖尚未注入
    fmt.Println("MyService registered")
}
```

### 2. IInjectBefore - 注入前回调

依赖注入开始前调用：

```go
func (s *MyService) OnInjectBefore() {
    // 即将开始注入依赖，可以在这里做一些准备工作
    fmt.Println("About to inject dependencies")
}
```

### 3. IInjectAfter - 注入后回调

单个对象的依赖注入完成后调用：

```go
func (s *MyService) OnInjectAfter() {
    // 当前对象的依赖已注入完成
    fmt.Println("Dependencies injected for MyService")
}
```

### 4. IObject - 所有注入完成回调

所有对象的依赖注入完成后调用（最终回调）：

```go
func (s *MyService) OnInjectComplete() {
    // 所有对象的依赖都已注入完成，可以在这里进行最终初始化
    fmt.Println("All dependencies injected, MyService ready")
}
```

### 完整生命周期示例

```go
type MyService struct {
    Dep *Dependency `autowire:"true"`
}

// 实现所有生命周期接口
func (s *MyService) OnProvideAfter() {
    fmt.Println("1. OnProvideAfter - 注册后")
}

func (s *MyService) OnInjectBefore() {
    fmt.Println("2. OnInjectBefore - 注入前")
}

func (s *MyService) OnInjectAfter() {
    fmt.Println("3. OnInjectAfter - 注入后")
}

func (s *MyService) OnInjectComplete() {
    fmt.Println("4. OnInjectComplete - 所有注入完成")
}
```

**回调执行顺序：**
1. `OnProvideAfter()` - 对象注册时
2. `OnInjectBefore()` - 启动容器时，每个对象注入前
3. `OnInjectAfter()` - 每个对象注入后
4. `OnInjectComplete()` - 所有对象注入完成后（最后执行）

## 日志配置

ioc233-go 使用 Go 标准库的 `log/slog` 作为日志入口。默认情况下使用 `slog.Default()`，你可以通过以下方式自定义：

### 方式一：设置全局 slog 默认日志

```go
import (
    "log/slog"
    "os"
)

// 设置全局默认日志（影响所有使用 slog.Default() 的代码）
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
slog.SetDefault(logger)
```

### 方式二：为 ioc233-go 单独设置日志

```go
import (
    "log/slog"
    "os"
    "github.com/neko233-com/ioc233-go"
)

// 为 ioc233-go 创建专用的日志实例
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
ioc233.SetLogger(logger)
```

### 方式三：使用自定义 Handler

```go
import (
    "log/slog"
    "os"
    "github.com/neko233-com/ioc233-go"
)

// 使用自定义 Handler（例如写入文件）
file, _ := os.OpenFile("ioc.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)
ioc233.SetLogger(logger)
```

### 静默日志

如果不设置日志，ioc233-go 会使用 `slog.Default()`，默认情况下不会输出任何内容（除非你通过 `slog.SetDefault()` 设置了全局日志）。

## 自动初始化字段

容器会自动初始化以下类型的字段（如果为 nil）：

- `map` 类型
- `slice` 类型
- `*rand.Rand` 类型

```go
type MyService struct {
    DataMap map[string]int  // 自动初始化为空 map
    DataSlice []string      // 自动初始化为空 slice
    Rand *rand.Rand        // 自动初始化为新的随机数生成器
}
```

## API 参考

### Container

- `Instance() *Container` - 获取全局容器实例（单例）
- `Provide(instance any)` - 注册对象（自动命名）
- `ProvideByName(name string, instance any) error` - 按名称注册对象
- `StartUp() error` - 启动容器，执行依赖注入
- `GetControllersAny() []any` - 获取所有控制器（兼容旧代码）

### 全局函数

- `GetObjectByType[T any]() T` - 按类型获取对象（泛型）
- `SetLogger(logger Logger)` - 设置全局日志
- `GetLogger() Logger` - 获取当前日志实例

### 接口

- `IProvideAfter` - 注册后生命周期接口
- `IInjectBefore` - 注入前生命周期接口
- `IInjectAfter` - 注入后生命周期接口
- `IObject` - 所有注入完成生命周期接口
- `Logger` - 日志接口

## 注意事项

1. **指针类型**：建议注册指针类型，以便容器可以修改字段值
2. **字段导出**：只有导出的字段（首字母大写）才能被注入
3. **启动顺序**：先注册所有对象，最后调用 `StartUp()` 执行注入
4. **线程安全**：容器内部使用读写锁，支持并发访问

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
