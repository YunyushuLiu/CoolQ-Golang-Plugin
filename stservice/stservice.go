package stservice

import (
	"sort"
)

// ISTService 服务接口
// 为使得一个结构体成为服务，其应当具备初始化、处理私聊和群聊消息的能力
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

// stService 服务结构体
type stService struct {
	name             string     // 服务名
	service          ISTService // 服务实例
	priority         int        // 服务优先级，应该是一个大于等于0的数字
	postServiceNames []string   // 后置服务名，获得的输入是该服务的输出
}

// Controller 服务控制器
type Controller struct {
	nServices     map[string]*stService // 服务名到服务的映射
	pServiceNames map[int][]string      // 服务优先级到服务名的映射
}

// NewService 创建一个服务
func NewService(name string, service ISTService, priority int) *stService {
	return &stService{
		name:             name,
		service:          service,
		priority:         priority,
		postServiceNames: nil,
	}
}

// PostService 向服务添加一个后置服务
func (s *stService) PostService(name string) *stService {
	s.postServiceNames = append(s.postServiceNames, name)
	return s
}

// NewController 创建一个服务控制器
func NewController() *Controller {
	nServices := make(map[string]*stService)
	pServiceNames := make(map[int][]string)
	return &Controller{
		nServices:     nServices,
		pServiceNames: pServiceNames,
	}
}

// RegisterService 向服务控制器注册一个服务
// 该服务的Init函数将被调用，如果将服务注册到同一个服务名，新的服务将不会被注册
// 函数返回一个bool，代表服务是否被成功注册
func (c *Controller) RegisterService(service *stService) bool {
	// 如果已经存在给定服务名到服务的映射，注册失败，返回false
	if _, exist := c.nServices[service.name]; exist {
		return false
	}
	// 更新服务名到服务的映射
	c.nServices[service.name] = service
	// 更新服务优先级到服务名的映射
	c.pServiceNames[service.priority] = append(c.pServiceNames[service.priority], service.name)
	service.service.Init()
	return true
}

// OnGroupMsg 收到群聊消息时触发
func (c *Controller) OnGroupMsg(subType, msgID int32, fromGroup, fromQQ int64, fromAnonymous, msg string, font int32) int32 {
	// 先获取所有服务的优先级
	var priorities []int
	for key := range c.pServiceNames {
		priorities = append(priorities, key)
	}
	// 将优先级排序
	sort.Ints(priorities[:])
	// 从小到大遍历优先级
	for _, priority := range priorities {
		// 如果优先级小于0，其为后置服务，不处理
		if priority < 0 {
			continue
		}
		// 获取当前优先级所有的服务
		serviceNames := c.pServiceNames[priority]
		// 原始消息是否向优先级低的服务传递，默认为false，交由下文判定
		transparent := false
		// 如果当前优先级没有服务，不处理
		if len(serviceNames) == 0 {
			continue
		}
		// 遍历服务名
		for _, serviceName := range serviceNames {
			// 从服务名到服务的映射中查找服务
			service, exist := c.nServices[serviceName]
			// 如果服务不存在，处理下一个服务
			if !exist {
				continue
			}
			// 将原始消息交由服务处理，获取服务的输出
			serviceReply, serviceTransparent := service.service.OnGroupMsg(subType, msgID, fromGroup, fromQQ, fromAnonymous, msg, font, false)
			// 一旦当前优先级的服务中有一个服务允许向下传递，原始消息就会向下传递
			if !transparent && serviceTransparent {
				transparent = true
			}
			// 如果服务返回不为空，将服务的输出送入后置服务
			if serviceReply != "" {
				// 遍历后置服务名
				for _, postServiceName := range service.postServiceNames {
					// 从服务名到服务的映射中查找服务
					postService, postServiceExist := c.nServices[postServiceName]
					// 如果后置服务不存在，处理下一个后置服务
					if !postServiceExist {
						continue
					}
					// 将服务输出交由后置服务处理，丢弃后置服务的输出
					postService.service.OnGroupMsg(subType, msgID, fromGroup, fromQQ, fromAnonymous, serviceReply, font, true)
				}
			}
		}
		// 如果不允许向下传递，跳出循环，结束处理
		if !transparent {
			break
		}
	}
	return 0
}

// OnPrivateMsg 收到私聊消息时触发
func (c *Controller) OnPrivateMsg(subType, msgID int32, fromQQ int64, msg string, font int32) int32 {
	// 先获取所有服务的优先级
	var priorities []int
	for key := range c.pServiceNames {
		priorities = append(priorities, key)
	}
	// 将优先级排序
	sort.Ints(priorities[:])
	// 从小到大遍历优先级
	for _, priority := range priorities {
		// 如果优先级小于0，其为后置服务，不处理
		if priority < 0 {
			continue
		}
		// 获取当前优先级所有的服务
		serviceNames := c.pServiceNames[priority]
		// 原始消息是否向优先级低的服务传递，默认为false，交由下文判定
		transparent := false
		// 如果当前优先级没有服务，不处理
		if len(serviceNames) == 0 {
			continue
		}
		// 遍历服务名
		for _, serviceName := range serviceNames {
			// 从服务名到服务的映射中查找服务
			service, exist := c.nServices[serviceName]
			// 如果服务不存在，处理下一个服务
			if !exist {
				continue
			}
			// 将原始消息交由服务处理，获取服务的输出
			serviceReply, serviceTransparent := service.service.OnPrivateMsg(subType, msgID, fromQQ, msg, font, false)
			// 一旦当前优先级的服务中有一个服务允许向下传递，原始消息就会向下传递
			if !transparent && serviceTransparent {
				transparent = true
			}
			// 如果服务返回不为空，将服务的输出送入后置服务
			if serviceReply != "" {
				// 遍历后置服务名
				for _, postServiceName := range service.postServiceNames {
					// 从服务名到服务的映射中查找服务
					postService, postServiceExist := c.nServices[postServiceName]
					// 如果后置服务不存在，处理下一个后置服务
					if !postServiceExist {
						continue
					}
					// 将服务输出交由后置服务处理，丢弃后置服务的输出
					postService.service.OnPrivateMsg(subType, msgID, fromQQ, serviceReply, font, true)
				}
			}
		}
		// 如果不允许向下传递，跳出循环，结束处理
		if !transparent {
			break
		}
	}
	return 0
}
