# stservice

stservice是针对[CoolQ-Golang-SDK](https://github.com/Tnze/CoolQ-Golang-SDK)开发的服务框架，赋予其服务化开发的能力，以简化开发流程，精简代码。

## 起步

stservice使用控制器控制各个服务，在开始前，你应该创建一个控制器全局变量：

```go
var c *stservice.Controller
```

同时，在`app.go`的`init()`函数中初始化它，并将消息处理交由控制器完成：

```go
c = stservice.NewController()
cqp.PrivateMsg = c.OnPrivateMsg
cqp.GroupMsg = c.OnGroupMsg
// 你也可以自己编写消息处理函数，然后手动调用控制器的相关函数
```

控制器中默认没有服务。为创建一个服务，首先需要一个服务实例。你可以定义一个结构体，并让其实现ISTService接口。以下代码可以让你创建一个服务实例：

```go
type TestService struct {
    
}

func (ts *TestService) Init() {
    // cqp.AddLog(cqp.Info, "ts", "Init TestService")
}

func (ts *TestService) OnGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32, post bool) (string, bool) {
    // 在这里处理群组消息
    // 将服务的输出返回，stservice并不会自动将返回发送，因此你需要自己执行cqp.SendGroupMsg()
    // 返回原始消息是否向优先级低的服务传递
}

func (ts *TestService) OnPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32, post bool) (string, bool) {
    // 在这里处理私聊消息
    // 将服务的输出返回，stservice并不会自动将返回发送，因此你需要自己执行cqp.SendPrivateMsg()
    // 返回原始消息是否向优先级低的服务传递
}
```

随后将服务实例和服务名相绑定，生成服务：

```go
ts := &TestService{}
stts := stservice.NewService("testService", &ts, 0)
```

`NewService()`函数的第一个参数为服务名，第二个参数为服务实例，第三个参数为优先级。下文将对优先级做出更加详细的说明。

创建完服务后，将其注册到控制器，它就可以正常工作了：

```go
c.RegisterService(stts)
// 此时，服务实例的Init()函数将被调用
```

## 服务

服务化开发是本框架的核心构思，在CoolQ机器人开发的环境下，我们认为一个服务应该具备：

- 群聊消息处理的能力，同时产生一个输出
- 私聊消息处理的能力，同时产生一个输出
- 初始化的能力

同时，我们认为需要考虑以下情况：

- 服务是有优先级的，高优先级的服务优先获得消息
- 服务可能会动态决定是否拦截消息，使得消息不再向优先级低的服务传递
- 服务产生的输出可能需要其他服务继续处理

因此，服务间的消息处理分为两条线：

- 原始消息的传递处理
- 服务输出的处理

### 原始消息的传递处理

在消息处理时优先级为0的服务会优先获得消息。你也很容易发现，如果任何一个服务允许将消息向下传递，那么原始消息就会被传递到下一个优先级的服务里。原始消息的传递处理流程图如下所示：

![stservice消息传递机制](img\stservice消息传递机制.png)

### 服务输出的处理

服务可能会产生输出，这就需要后置服务对这个输出进行处理。服务输出的处理流程图如下所示：

![stservice后置服务机制](img\stservice后置服务机制.png)

我们希望所有的服务能够以一个井然有序的状态运行，不出现混乱，防止套娃。因此，所有后置服务的优先级和及其后置服务的设定均会被忽略。所有后置服务将同时收到服务输出。

使用`PostService()`函数添加后置服务：

```go
stts = stts.PostService("testService")
```

使用`PostService()`函数添加后置服务时使用服务名，并且在添加时不会检查服务是否存在，因此你无需注意服务的声明先后。同时，从给出的代码你可以发现，`PostService()`函数返回的仍然是服务，所以你可以链式调用它：

```go
stts := stservice.NewService("testService", &ts, 0).PostService("testService")
```

## ISTService

ISTService是服务接口，它描述了一个服务需要实现的方法：

```go
type ISTService interface {
	// OnGroupMsg 收到群聊消息时触发，返回服务的输出和是否向下传递消息
	// post表示其是否作为后置服务被触发
	OnGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32, post bool) (string, bool)
	// OnPrivateMsg 收到私聊消息时触发，返回服务的输出和是否向下传递消息
	// post表示其是否作为后置服务被触发
	OnPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32, post bool) (string, bool)
	// 注册服务时触发
	Init()
}
```

你会发现`OnGroupMsg()`和`OnPrivateMsg()`基本符合CoolQ-Golang-SDK中`cqp.GroupMsg`和`cqp.PrivateMsg`的类型。`ISTService`添加了`post`参数，用以标记服务是否作为后置服务被触发。

**注意：**作为后置服务被触发时，`msg`将被替换成服务的输出，`post`置为`true`，其余的参数会保持不变！