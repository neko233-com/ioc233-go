package tests

import (
	"log/slog"
	"math/rand"
	"sync"
	"testing"

	"github.com/neko233-com/ioc233-go/ioc233"
)

// ==================== 测试用的接口和实现 ====================

type UserService interface {
	GetUser(id int) string
}

type UserServiceImpl struct {
	ID int
}

func (s *UserServiceImpl) GetUser(id int) string {
	return "User"
}

type OrderService interface {
	GetOrder(id int) string
}

type OrderServiceImpl struct {
	UserService UserService `autowire:"true"`
}

func (s *OrderServiceImpl) GetOrder(id int) string {
	return "Order"
}

// ==================== 生命周期测试结构体 ====================

type LifecycleTracker struct {
	ProvideAfterCalled   bool
	InjectBeforeCalled   bool
	InjectAfterCalled    bool
	InjectCompleteCalled bool
	mu                   sync.Mutex
}

func (l *LifecycleTracker) OnProvideAfter() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ProvideAfterCalled = true
}

func (l *LifecycleTracker) OnInjectBefore() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.InjectBeforeCalled = true
}

func (l *LifecycleTracker) OnInjectAfter() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.InjectAfterCalled = true
}

func (l *LifecycleTracker) OnInjectComplete() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.InjectCompleteCalled = true
}

// ==================== 测试辅助函数 ====================

func resetContainer() {
	// 使用 build tag 提供的 Reset 函数
	// 需要在测试时使用 -tags test 标志
	ioc233.Reset()
}

// ==================== 基本功能测试 ====================

func TestContainer_Provide(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service := &UserServiceImpl{ID: 1}
	container.Provide(service)

	// 验证可以通过类型获取
	retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
	if retrieved == nil {
		t.Fatal("应该能获取到注册的服务")
	}
	if retrieved.ID != 1 {
		t.Errorf("期望 ID=1, 得到 ID=%d", retrieved.ID)
	}
}

func TestContainer_ProvideByName(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service := &UserServiceImpl{ID: 2}
	err := container.ProvideByName("MyUserService", service)
	if err != nil {
		t.Fatalf("ProvideByName 应该成功, 错误: %v", err)
	}

	// 验证可以通过名称获取（通过类型名）
	retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
	if retrieved == nil {
		t.Fatal("应该能获取到注册的服务")
	}
}

func TestContainer_ProvideDuplicateType(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service1 := &UserServiceImpl{ID: 1}
	service2 := &UserServiceImpl{ID: 2}

	container.Provide(service1)
	container.Provide(service2) // 重复注册应该被忽略

	retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
	if retrieved.ID != 1 {
		t.Errorf("重复注册应该保留第一个, 期望 ID=1, 得到 ID=%d", retrieved.ID)
	}
}

func TestContainer_ProvideByNameDuplicate(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service1 := &UserServiceImpl{ID: 1}
	service2 := &UserServiceImpl{ID: 2}

	err1 := container.ProvideByName("MyService", service1)
	if err1 != nil {
		t.Fatalf("第一次注册应该成功, 错误: %v", err1)
	}

	err2 := container.ProvideByName("MyService", service2)
	if err2 == nil {
		t.Fatal("重复名称注册应该返回错误")
	}

	// 验证启动应该失败
	err := container.StartUp()
	if err == nil {
		t.Fatal("存在致命错误时启动应该失败")
	}
}

func TestContainer_ProvideNil(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	container.Provide(nil) // 应该安全处理 nil

	// 验证容器仍然可用
	service := &UserServiceImpl{ID: 1}
	container.Provide(service)
	retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
	if retrieved == nil {
		t.Fatal("容器应该仍然可用")
	}
}

// ==================== 依赖注入测试 ====================

func TestContainer_AutowireByType(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	userService := &UserServiceImpl{ID: 1}
	orderService := &OrderServiceImpl{}

	container.Provide(userService)
	container.Provide(orderService)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if orderService.UserService == nil {
		t.Fatal("UserService 应该被注入")
	}

	if orderService.UserService.(*UserServiceImpl).ID != 1 {
		t.Errorf("注入的 UserService ID 应该为 1, 得到: %d", orderService.UserService.(*UserServiceImpl).ID)
	}
}

func TestContainer_AutowireByName(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceA struct {
		ServiceB *UserServiceImpl `autowire:"MyServiceB"`
	}

	serviceB := &UserServiceImpl{ID: 2}
	serviceA := &ServiceA{}

	err := container.ProvideByName("MyServiceB", serviceB)
	if err != nil {
		t.Fatalf("注册应该成功, 错误: %v", err)
	}
	container.Provide(serviceA)

	err = container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if serviceA.ServiceB == nil {
		t.Fatal("ServiceB 应该被注入")
	}

	if serviceA.ServiceB.ID != 2 {
		t.Errorf("注入的 ServiceB ID 应该为 2, 得到: %d", serviceA.ServiceB.ID)
	}
}

func TestContainer_AutowireOptional(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceA struct {
		OptionalService *UserServiceImpl `autowire:"false"`
	}

	serviceA := &ServiceA{}
	container.Provide(serviceA)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 可选注入，找不到应该保持 nil
	if serviceA.OptionalService != nil {
		t.Fatal("可选注入未找到时应该保持 nil")
	}
}

func TestContainer_AutowireInterface(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceA struct {
		UserService UserService `autowire:"true"`
	}

	userService := &UserServiceImpl{ID: 3}
	serviceA := &ServiceA{}

	container.Provide(userService)
	container.Provide(serviceA)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if serviceA.UserService == nil {
		t.Fatal("UserService 接口应该被注入")
	}

	result := serviceA.UserService.GetUser(1)
	if result != "User" {
		t.Errorf("期望 'User', 得到 '%s'", result)
	}
}

// ==================== 生命周期回调测试 ====================

func TestLifecycle_IProvideAfter(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	tracker := &LifecycleTracker{}
	container.Provide(tracker)

	if !tracker.ProvideAfterCalled {
		t.Fatal("OnProvideAfter 应该被调用")
	}

	if tracker.InjectBeforeCalled {
		t.Fatal("OnInjectBefore 不应该在注册时被调用")
	}
}

func TestLifecycle_IInjectBefore(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	tracker := &LifecycleTracker{}
	container.Provide(tracker)

	if tracker.InjectBeforeCalled {
		t.Fatal("OnInjectBefore 不应该在注册时被调用")
	}

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if !tracker.InjectBeforeCalled {
		t.Fatal("OnInjectBefore 应该在注入前被调用")
	}
}

func TestLifecycle_IInjectAfter(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	tracker := &LifecycleTracker{}
	container.Provide(tracker)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if !tracker.InjectAfterCalled {
		t.Fatal("OnInjectAfter 应该在注入后被调用")
	}

	// 注意：对于单个对象，OnInjectComplete 会在 OnInjectAfter 之后立即调用
	// 这是正确的行为，因为所有对象的注入都已完成
	if !tracker.InjectCompleteCalled {
		t.Fatal("OnInjectComplete 应该在所有对象的注入完成后被调用")
	}
}

func TestLifecycle_IObject_OnInjectComplete(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	tracker := &LifecycleTracker{}
	container.Provide(tracker)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	if !tracker.InjectCompleteCalled {
		t.Fatal("OnInjectComplete 应该被调用")
	}
}

func TestLifecycle_AllCallbacks(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	tracker := &LifecycleTracker{}
	container.Provide(tracker)

	// 验证注册后回调
	if !tracker.ProvideAfterCalled {
		t.Fatal("OnProvideAfter 应该被调用")
	}

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 验证所有回调都被调用
	if !tracker.ProvideAfterCalled {
		t.Fatal("OnProvideAfter 应该被调用")
	}
	if !tracker.InjectBeforeCalled {
		t.Fatal("OnInjectBefore 应该被调用")
	}
	if !tracker.InjectAfterCalled {
		t.Fatal("OnInjectAfter 应该被调用")
	}
	if !tracker.InjectCompleteCalled {
		t.Fatal("OnInjectComplete 应该被调用")
	}
}

// FullLifecycleTest 用于测试所有生命周期回调的结构体
type FullLifecycleTest struct {
	orderTracker *CallbackOrderTest
}

type CallbackOrderTest struct {
	order []string
	mu    sync.Mutex
}

func (f *FullLifecycleTest) OnProvideAfter() {
	f.orderTracker.mu.Lock()
	defer f.orderTracker.mu.Unlock()
	f.orderTracker.order = append(f.orderTracker.order, "OnProvideAfter")
}

func (f *FullLifecycleTest) OnInjectBefore() {
	f.orderTracker.mu.Lock()
	defer f.orderTracker.mu.Unlock()
	f.orderTracker.order = append(f.orderTracker.order, "OnInjectBefore")
}

func (f *FullLifecycleTest) OnInjectAfter() {
	f.orderTracker.mu.Lock()
	defer f.orderTracker.mu.Unlock()
	f.orderTracker.order = append(f.orderTracker.order, "OnInjectAfter")
}

func (f *FullLifecycleTest) OnInjectComplete() {
	f.orderTracker.mu.Lock()
	defer f.orderTracker.mu.Unlock()
	f.orderTracker.order = append(f.orderTracker.order, "OnInjectComplete")
}

func TestLifecycle_CallbackOrder(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	orderTracker := &CallbackOrderTest{order: make([]string, 0)}
	fullLifecycle := &FullLifecycleTest{orderTracker: orderTracker}

	container.Provide(fullLifecycle)

	// 验证注册后回调
	if len(orderTracker.order) == 0 || orderTracker.order[0] != "OnProvideAfter" {
		t.Fatal("OnProvideAfter 应该首先被调用")
	}

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 验证回调顺序
	expectedOrder := []string{"OnProvideAfter", "OnInjectBefore", "OnInjectAfter", "OnInjectComplete"}
	if len(orderTracker.order) != len(expectedOrder) {
		t.Fatalf("期望 %d 个回调, 得到 %d 个", len(expectedOrder), len(orderTracker.order))
	}

	for i, expected := range expectedOrder {
		if orderTracker.order[i] != expected {
			t.Errorf("位置 %d: 期望 %s, 得到 %s", i, expected, orderTracker.order[i])
		}
	}
}

// ==================== 字段自动初始化测试 ====================

func TestContainer_AutoInitMap(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceWithMap struct {
		DataMap map[string]int
	}

	service := &ServiceWithMap{}
	container.Provide(service)

	if service.DataMap == nil {
		t.Fatal("map 字段应该被自动初始化")
	}

	if len(service.DataMap) != 0 {
		t.Fatal("map 应该被初始化为空 map")
	}
}

func TestContainer_AutoInitSlice(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceWithSlice struct {
		DataSlice []string
	}

	service := &ServiceWithSlice{}
	container.Provide(service)

	if service.DataSlice == nil {
		t.Fatal("slice 字段应该被自动初始化")
	}

	if len(service.DataSlice) != 0 {
		t.Fatal("slice 应该被初始化为空 slice")
	}
}

func TestContainer_AutoInitRand(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceWithRand struct {
		Rand *rand.Rand
	}

	service := &ServiceWithRand{}
	container.Provide(service)

	if service.Rand == nil {
		t.Fatal("*rand.Rand 字段应该被自动初始化")
	}
}

// ==================== 错误处理测试 ====================

func TestContainer_StartUpWithFatalError(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service1 := &UserServiceImpl{ID: 1}
	service2 := &UserServiceImpl{ID: 2}

	container.ProvideByName("Duplicate", service1)
	container.ProvideByName("Duplicate", service2) // 重复名称

	err := container.StartUp()
	if err == nil {
		t.Fatal("存在致命错误时启动应该失败")
	}
}

func TestContainer_AutowireRequiredNotFound(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceA struct {
		RequiredService *UserServiceImpl `autowire:"true"`
	}

	serviceA := &ServiceA{}
	container.Provide(serviceA)

	// 启动应该成功（只记录错误，不阻止启动）
	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功（即使有注入错误）, 错误: %v", err)
	}

	// 但字段应该保持 nil（因为找不到）
	if serviceA.RequiredService != nil {
		t.Fatal("找不到依赖时字段应该保持 nil")
	}
}

// ==================== 并发安全测试 ====================

func TestContainer_ConcurrentProvide(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	var wg sync.WaitGroup
	count := 10

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			service := &UserServiceImpl{ID: id}
			container.Provide(service)
		}(i)
	}

	wg.Wait()

	// 验证容器仍然可用
	retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
	if retrieved == nil {
		t.Fatal("并发注册后应该能获取到服务")
	}
}

func TestContainer_ConcurrentGet(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	service := &UserServiceImpl{ID: 1}
	container.Provide(service)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	var wg sync.WaitGroup
	count := 10

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retrieved := ioc233.GetObjectByType[*UserServiceImpl]()
			if retrieved == nil {
				t.Error("应该能获取到服务")
			}
		}()
	}

	wg.Wait()
}

// ==================== 边界情况测试 ====================

func TestContainer_NonPointerType(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type NonPointerService struct {
		ID int
	}

	service := NonPointerService{ID: 1}
	container.Provide(service) // 非指针类型应该被接受但警告

	// 验证仍然可以工作
	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}
}

func TestContainer_UnexportedField(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceWithUnexported struct {
		ExportedField   *UserServiceImpl `autowire:"true"` // 大写开头，可导出
		unexportedField *UserServiceImpl `autowire:"true"` // 小写开头，不可导出
	}

	userService := &UserServiceImpl{ID: 1}
	service := &ServiceWithUnexported{}

	container.Provide(userService)
	container.Provide(service)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 可导出字段应该被注入
	if service.ExportedField == nil {
		t.Fatal("可导出字段应该被注入")
	}

	if service.ExportedField.ID != 1 {
		t.Errorf("注入的字段 ID 应该为 1, 得到: %d", service.ExportedField.ID)
	}

	// 不可导出字段不会被注入（这是预期的，因为 CanSet() 返回 false）
}

func TestContainer_EmptyContainer(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	err := container.StartUp()
	if err != nil {
		t.Fatalf("空容器启动应该成功, 错误: %v", err)
	}
}

// ==================== 复杂场景测试 ====================

// CircularServiceA 和 CircularServiceB 用于测试循环依赖
type CircularServiceA struct {
	ServiceB *CircularServiceB `autowire:"CircularServiceB"`
}

type CircularServiceB struct {
	ServiceA *CircularServiceA `autowire:"CircularServiceA"`
}

func TestContainer_CircularDependency(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	serviceA := &CircularServiceA{}
	serviceB := &CircularServiceB{}

	// 使用名称注册
	container.ProvideByName("CircularServiceA", serviceA)
	container.ProvideByName("CircularServiceB", serviceB)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 验证循环依赖被正确注入
	if serviceA.ServiceB == nil {
		t.Fatal("ServiceA.ServiceB 应该被注入")
	}
	if serviceB.ServiceA == nil {
		t.Fatal("ServiceB.ServiceA 应该被注入")
	}

	// 验证循环引用
	if serviceA.ServiceB != serviceB {
		t.Fatal("ServiceA.ServiceB 应该指向同一个 ServiceB 实例")
	}
	if serviceB.ServiceA != serviceA {
		t.Fatal("ServiceB.ServiceA 应该指向同一个 ServiceA 实例")
	}
}

func TestContainer_MultipleImplementations(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	type ServiceA struct {
		UserService UserService `autowire:"true"`
	}

	impl1 := &UserServiceImpl{ID: 1}
	impl2 := &UserServiceImpl{ID: 2}
	serviceA := &ServiceA{}

	container.Provide(impl1)
	container.Provide(impl2)
	container.Provide(serviceA)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 应该注入第一个找到的实现
	if serviceA.UserService == nil {
		t.Fatal("UserService 应该被注入")
	}
}

// ==================== 工具函数测试 ====================

func TestGetObjectByType_NotFound(t *testing.T) {
	resetContainer()

	type UnregisteredService struct {
		ID int
	}

	retrieved := ioc233.GetObjectByType[*UnregisteredService]()
	if retrieved != nil {
		t.Fatal("未注册的服务应该返回 nil")
	}
}

func TestGetObjectByType_Interface(t *testing.T) {
	resetContainer()
	container := ioc233.Instance()

	userService := &UserServiceImpl{ID: 1}
	container.Provide(userService)

	err := container.StartUp()
	if err != nil {
		t.Fatalf("启动应该成功, 错误: %v", err)
	}

	// 通过接口类型获取实现
	retrieved := ioc233.GetObjectByType[UserService]()
	if retrieved == nil {
		t.Fatal("应该能通过接口类型获取实现")
	}

	// 验证可以调用接口方法
	result := retrieved.GetUser(1)
	if result != "User" {
		t.Errorf("期望 'User', 得到 '%s'", result)
	}
}

// ==================== 日志测试 ====================

func TestSetLogger(t *testing.T) {
	originalLogger := ioc233.GetLogger()

	// 设置自定义日志（使用 slog）
	customLogger := slog.Default()
	ioc233.SetLogger(customLogger)

	currentLogger := ioc233.GetLogger()
	if currentLogger == nil {
		t.Fatal("GetLogger 应该返回设置的日志实例")
	}

	// 恢复原始日志
	ioc233.SetLogger(originalLogger)
}

func TestSetLogger_Nil(t *testing.T) {
	ioc233.SetLogger(nil)
	logger := ioc233.GetLogger()
	if logger == nil {
		t.Fatal("设置 nil 日志应该使用默认 slog.Default()")
	}
}
