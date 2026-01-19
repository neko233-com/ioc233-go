package ioc233

import (
	"errors"
	"reflect"
	"strings"
	"sync"
)

// Container 全局 IOC 容器
// 设计目标：
//   - 只负责"对象注册 + 依赖注入"，不做业务维度的归类管理
//   - Controller/Service 的分类与控制器列表维护、ConfigManager 的业务注册，交由 apps 包统一管理
//   - 注入语义说明：
//     autowire:"true"  -> 必须注入，按字段类型（接口或具体类型）自动查找实现；找不到记录错误
//     autowire:"false" -> 可选注入，按字段类型自动查找实现；找不到则保持 nil
//     autowire:"名称"   -> 名称注入，按 bean 名称查找；类型不兼容或未找到则记录错误
type Container struct {
	mutex sync.RWMutex

	// 业务模块依赖容器
	serviceMap      map[reflect.Type]any
	controllerMap   map[reflect.Type]any
	typeToObjectMap map[reflect.Type]any
	nameToObjMap    map[string]any

	// 控制器列表
	controllerList []any

	// 启动前的致命错误（例如重复的 ProvideByName）
	fatalErrors []error
}

var (
	_instance *Container
	_once     sync.Once
)

// Instance 获取全局 IOC 容器实例（单例）
func Instance() *Container {
	_once.Do(func() {
		_instance = &Container{
			serviceMap:      make(map[reflect.Type]any),
			controllerMap:   make(map[reflect.Type]any),
			typeToObjectMap: make(map[reflect.Type]any),
			nameToObjMap:    make(map[string]any),
			controllerList:  make([]any, 0, 64),
			fatalErrors:     make([]error, 0, 8),
		}
	})
	return _instance
}

// Provide 注册一个对象到 IOC 容器（自动使用结构体名作为 bean 名）
// 说明：
// - 仅在 ioc 内维护类型/名称到实例的映射
// - 不进行业务维度的分类判断（Controller/Service/ConfigManager），由 apps 统一处理
func (c *Container) Provide(instance any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if instance == nil {
		return
	}

	t := reflect.TypeOf(instance)
	if t.Kind() != reflect.Ptr {
		logWarn("[ioc233] Provide 建议注册指针类型: %v", t)
	}

	// 初始化基础字段（跳过 autowire:"true"）
	c.initBasicFields(instance)

	// 记录类型映射（重复类型则忽略并警告，保留首个实例）
	if _, exists := c.typeToObjectMap[t]; exists {
		logWarn("[ioc233] Provide 重复类型注册，忽略: %v", t)
		return
	}
	c.typeToObjectMap[t] = instance

	// 默认 bean 名为结构体名（不含包名）
	beanName := t.Name()
	if beanName == "" && t.Kind() == reflect.Ptr {
		beanName = t.Elem().Name()
	}
	if beanName == "" {
		beanName = t.String()
	}
	// 如果默认名已存在，警告并跳过名称注册（不阻断启动）
	if _, exists := c.nameToObjMap[beanName]; exists {
		logWarn("[ioc233] Provide 默认 bean 名重复，忽略: %s", beanName)
	} else {
		c.nameToObjMap[beanName] = instance
	}

	typeName := t.String()
	logInfo("[ioc233] 注册 bean | struct name = %s (type: %v)", typeName, t)

	// 触发注册后回调
	if obj, ok := instance.(IProvideAfter); ok {
		logInfo("[ioc233] 触发注册后回调: %v", t)
		obj.OnProvideAfter()
	}

	// 业务分类与 ConfigManager 的注册由 apps 包负责
}

// ProvideByName 按指定名称注册对象（重复名视为致命错误）
// 说明：
// - 仅维护名称到实例的映射；业务维度的分类与注册交由 apps 包处理
func (c *Container) ProvideByName(name string, instance any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if instance == nil || strings.TrimSpace(name) == "" {
		return errors.New("[ioc233] ProvideByName 参数非法")
	}

	if _, exists := c.nameToObjMap[name]; exists {
		err := errors.New("[ioc233] ProvideByName 重复注册: name=" + name)
		logError("%s", err.Error())
		c.fatalErrors = append(c.fatalErrors, err)
		return err
	}

	t := reflect.TypeOf(instance)
	if t.Kind() != reflect.Ptr {
		logWarn("[ioc233] ProvideByName 建议注册指针类型: %v", t)
	}

	c.initBasicFields(instance)

	c.typeToObjectMap[t] = instance
	c.nameToObjMap[name] = instance

	typeName := t.String()
	logInfo("[ioc233] 注册 bean(byName) | name = %s, struct = %s (type: %v)", name, typeName, t)

	// 触发注册后回调
	if obj, ok := instance.(IProvideAfter); ok {
		logInfo("[ioc233] 触发注册后回调: %v", t)
		obj.OnProvideAfter()
	}

	// 业务分类与 ConfigManager 的注册由 apps 包负责
	return nil
}

// StartUp 执行依赖注入（autowire）
// 行为：
// - 遍历所有注册对象，按字段标签执行注入
// - 触发对象的 OnInjectComplete 生命周期回调
// - 若之前记录致命错误（如 ProvideByName 重复），则阻止启动
func (c *Container) StartUp() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	logInfo("[ioc233] 🚀 正在启动 IOC 容器并执行依赖注入...")

	// 先检查是否存在致命错误（例如重复 ProvideByName）
	if len(c.fatalErrors) > 0 {
		for _, e := range c.fatalErrors {
			logError("[ioc233] 致命错误: %v", e)
		}
		return errors.New("[ioc233] 容器存在致命错误，启动失败")
	}

	// 注入字段
	for t, instance := range c.typeToObjectMap {
		typeName := t.Name()
		if typeName == "" && t.Kind() == reflect.Ptr {
			typeName = t.Elem().Name()
		}
		if typeName == "" {
			typeName = t.String()
		}
		logInfo("[ioc233] 开始注入对象字段: struct=%s", typeName)

		// 触发注入前回调
		if obj, ok := instance.(IInjectBefore); ok {
			logInfo("[ioc233] 触发注入前回调: %v", t)
			obj.OnInjectBefore()
		}

		// 执行注入
		c.injectInternal(instance)

		// 触发注入后回调
		if obj, ok := instance.(IInjectAfter); ok {
			logInfo("[ioc233] 触发注入后回调: %v", t)
			obj.OnInjectAfter()
		}
	}

	// 注入完成回调
	for t, instance := range c.typeToObjectMap {
		if obj, ok := instance.(IObject); ok {
			logInfo("[ioc233] 注入完成回调: %v", t)
			obj.OnInjectComplete()
		}
	}

	logInfo("[ioc233] ✅ IOC 容器启动完成，所有依赖注入已就绪")
	return nil
}

// initBasicFields 初始化基础字段（map、slice、*rand.Rand 等）
// 规则：
// - 跳过携带 autowire/inject 标签的字段，避免与注入阶段冲突
// - 对 map/slice/*rand.Rand 等可导出字段进行默认初始化
func (c *Container) initBasicFields(instance any) {
	v := reflect.ValueOf(instance)
	if v.Kind() != reflect.Ptr {
		return
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}

	t := elem.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !elem.Field(i).CanSet() {
			continue
		}
		aw := field.Tag.Get("autowire")
		inj := field.Tag.Get("inject")
		if aw != "" || inj != "" {
			// 任何声明了 autowire/inject 的字段都跳过基础初始化
			continue
		}
		fv := elem.Field(i)

		if ApplyDefaultProviders(field, fv) {
			logDebug("[ioc233] 字段默认值提供器应用: struct=%s field=%s type=%s", t.Name(), field.Name, field.Type.String())
		}
	}
}

// injectInternal 执行依赖注入（核心）
// 规则：
// - autowire:"true"  -> 必须按类型注入；找不到实现则记录错误
// - autowire:"false" -> 可选按类型注入；找不到实现则保持 nil
// - 其他             -> 作为名称注入；不兼容或未找到则记录错误
func (c *Container) injectInternal(instance any) {
	v := reflect.ValueOf(instance)
	if v.Kind() != reflect.Ptr {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("autowire")
		if tag == "" {
			tag = field.Tag.Get("inject")
			if tag == "" {
				continue
			}
		}
		if !v.Field(i).CanSet() {
			logError("[ioc233] 字段 %s.%s 带有 autowire 标签但不可导出，跳过注入", t.Name(), field.Name)
			continue
		}

		fieldType := field.Type
		structName := t.Name()
		if structName == "" && t.Kind() == reflect.Ptr {
			structName = t.Elem().Name()
		}
		if structName == "" {
			structName = t.String()
		}
		logInfo("[ioc233] 尝试注入: struct=%s field=%s type=%v autowire=%s", structName, field.Name, fieldType, tag)

		// 选择注入模式：true/false 按类型；其他值按名称
		if tag == "true" || tag == "false" {
			mandatory := tag == "true"
			// 自动按字段类型注入
			if fieldType.Kind() == reflect.Interface {
				var candidates []reflect.Value
				for _, obj := range c.typeToObjectMap {
					if obj == nil {
						continue
					}
					objVal := reflect.ValueOf(obj)
					objType := objVal.Type()
					if objType.Implements(fieldType) || (objType.Kind() == reflect.Ptr && objType.Elem().Implements(fieldType)) {
						candidates = append(candidates, objVal)
					}
				}
				if len(candidates) >= 1 {
					v.Field(i).Set(candidates[0])
					if len(candidates) > 1 {
						typeNames := make([]string, 0, len(candidates))
						for _, cnd := range candidates {
							typeNames = append(typeNames, cnd.Type().String())
						}
						logWarn("[ioc233] 接口类型存在多个实现，默认注入第一个: struct=%s field=%s iface=%v impls=%v",
							structName, field.Name, fieldType, typeNames)
					} else {
						logDebug("[ioc233] 接口类型注入成功: %s.%s (iface=%v, impl=%v)", structName, field.Name, fieldType, candidates[0].Type())
					}
				} else if mandatory {
					logError("[ioc233] 接口类型注入失败: struct=%s field=%s (未找到实现 iface=%v)", structName, field.Name, fieldType)
				} else {
					// 可选注入：不报错，保持 nil
					logInfo("[ioc233] 接口类型可选注入: 未找到实现，保持 nil (struct=%s field=%s iface=%v)", structName, field.Name, fieldType)
				}
				continue
			}
			// 非接口类型：按类型名在 nameToObjMap 查找
			typeName := fieldType.Name()
			if typeName == "" && fieldType.Kind() == reflect.Ptr {
				typeName = fieldType.Elem().Name()
			}
			if typeName == "" {
				typeName = fieldType.String()
			}
			if obj, ok := c.nameToObjMap[typeName]; ok && obj != nil {
				objVal := reflect.ValueOf(obj)
				objType := objVal.Type()
				if objType.AssignableTo(fieldType) {
					v.Field(i).Set(objVal)
					logDebug("[ioc233] 类型名注入成功: %s.%s (typeName=%s, actualType=%v)", structName, field.Name, typeName, objType)
				} else if mandatory {
					logError("[ioc233] 类型名注入不匹配: struct=%s field=%s (fieldType=%v, foundType=%v)",
						structName, field.Name, fieldType, objType)
				} else {
					logInfo("[ioc233] 类型名可选注入不匹配，保持 nil: struct=%s field=%s (fieldType=%v, foundType=%v)",
						structName, field.Name, fieldType, objType)
				}
			} else if mandatory {
				logError("[ioc233] 类型名注入失败: struct=%s field=%s (未找到类型名=%q 的实例)", structName, field.Name, typeName)
			} else {
				logInfo("[ioc233] 类型名可选注入: 未找到实例，保持 nil (struct=%s field=%s typeName=%q)", structName, field.Name, typeName)
			}
			continue
		}

		// 名称注入：autowire:"BeanName"
		if obj, ok := c.nameToObjMap[tag]; ok && obj != nil {
			objVal := reflect.ValueOf(obj)
			objType := objVal.Type()
			compatible := objType.AssignableTo(fieldType) ||
				(fieldType.Kind() == reflect.Interface && (objType.Implements(fieldType) ||
					(objType.Kind() == reflect.Ptr && objType.Elem().Implements(fieldType))))
			if compatible {
				v.Field(i).Set(objVal)
				logDebug("[ioc233] 名称注入成功: %s.%s (name=%s, type=%v)", structName, field.Name, tag, objType)
			} else {
				logError("[ioc233] 名称注入类型不匹配: struct=%s field=%s (name=%s, fieldType=%v, foundType=%v)",
					structName, field.Name, tag, fieldType, objType)
			}
		} else {
			logError("[ioc233] 名称注入失败: struct=%s field=%s (未找到名称为 %q 的实例)", structName, field.Name, tag)
		}
		continue
	}
}

// GetObjectByType 按类型获取对象（泛型）
// 优先查找：serviceMap/controllerMap/typeToObjectMap
// 如果 T 是接口类型，会查找实现了该接口的具体类型
func GetObjectByType[T any]() T {
	c := Instance()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var zero T
	targetType := reflect.TypeOf((*T)(nil)).Elem()

	// 如果是接口类型，查找实现了该接口的对象
	if targetType.Kind() == reflect.Interface {
		for _, instance := range c.typeToObjectMap {
			if instance == nil {
				continue
			}
			objType := reflect.TypeOf(instance)
			if objType.Implements(targetType) || (objType.Kind() == reflect.Ptr && objType.Elem().Implements(targetType)) {
				if typed, ok := instance.(T); ok {
					return typed
				}
			}
		}
		// 也检查 serviceMap 和 controllerMap
		for _, instance := range c.serviceMap {
			if instance == nil {
				continue
			}
			objType := reflect.TypeOf(instance)
			if objType.Implements(targetType) || (objType.Kind() == reflect.Ptr && objType.Elem().Implements(targetType)) {
				if typed, ok := instance.(T); ok {
					return typed
				}
			}
		}
		for _, instance := range c.controllerMap {
			if instance == nil {
				continue
			}
			objType := reflect.TypeOf(instance)
			if objType.Implements(targetType) || (objType.Kind() == reflect.Ptr && objType.Elem().Implements(targetType)) {
				if typed, ok := instance.(T); ok {
					return typed
				}
			}
		}
		logError("[ioc233] 未找到实现接口 %v 的实例", targetType)
		return zero
	}

	// 具体类型查找
	if instance, ok := c.serviceMap[targetType]; ok {
		if typed, ok := instance.(T); ok {
			return typed
		}
	}
	if instance, ok := c.controllerMap[targetType]; ok {
		if typed, ok := instance.(T); ok {
			return typed
		}
	}
	if instance, ok := c.typeToObjectMap[targetType]; ok {
		if typed, ok := instance.(T); ok {
			return typed
		}
	}
	logError("[ioc233] 未找到类型的实例: %v", targetType)
	return zero
}

// GetControllersAny 获取所有控制器（兼容旧代码）
// 说明：实际的控制器列表由 apps 维护；此处仅保留以兼容历史调用
func (c *Container) GetControllersAny() []any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.controllerList
}
