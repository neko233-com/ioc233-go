package ioc233

// IProvideAfter 注册后生命周期接口
// 实现此接口的对象在注册到容器后会调用 OnProvideAfter 方法
type IProvideAfter interface {
	// OnProvideAfter 对象注册到容器后的回调方法
	OnProvideAfter()
}

// IInjectBefore 注入前生命周期接口
// 实现此接口的对象在依赖注入开始前会调用 OnInjectBefore 方法
type IInjectBefore interface {
	// OnInjectBefore 依赖注入开始前的回调方法
	OnInjectBefore()
}

// IInjectAfter 注入后生命周期接口
// 实现此接口的对象在依赖注入完成后会调用 OnInjectAfter 方法
// 注意：这是在单个对象的字段注入完成后调用，而不是所有对象都注入完成后
type IInjectAfter interface {
	// OnInjectAfter 依赖注入完成后的回调方法（单个对象）
	OnInjectAfter()
}

// IObject 对象生命周期接口
// 实现此接口的对象在所有对象的依赖注入完成后会调用 OnInjectComplete 方法
// 这是整个容器启动完成后的最终回调
type IObject interface {
	// OnInjectComplete 所有依赖注入完成后的回调方法
	OnInjectComplete()
}
