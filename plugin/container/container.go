package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/elvisNg/broccoliv2/plugin"
	"github.com/elvisNg/broccoliv2/plugin/zcontainer"
)

func init() {
	containerProvidors = map[string]NewContainerFn{
		CATEGORY_broccoli: NewbroccoliContainer,
	}
}

// CATEGORY_broccoli 框架默认提供的containerprovidor名称
const CATEGORY_broccoli = "broccoli"

var containerProvidors map[string]NewContainerFn

type NewContainerFn func(ctx context.Context) zcontainer.Container

// GetContainer 返回框架默认提供的container
func GetContainer() zcontainer.Container {
	return NewContainer(context.Background(), CATEGORY_broccoli)
}

// NewbroccoliContainer plugin.NewContainer框架默认提供的containerprovidor
func NewbroccoliContainer(ctx context.Context) zcontainer.Container {
	cnt := plugin.NewContainer()
	return cnt
}

// NewContainer 创建指定的container
func NewContainer(ctx context.Context, categoryName string) zcontainer.Container {
	name := categoryName
	if len(strings.TrimSpace(categoryName)) == 0 {
		name = CATEGORY_broccoli
	}
	p, ok := containerProvidors[name]
	if !ok || p == nil {
		panic(fmt.Errorf("unsupport containerprovidor categoryName: %s", name))
	}
	return p(ctx)
}

// RegistryProvidor 注册容器提供者，用于扩展containerprovidor
func RegistryProvidor(name string, providor NewContainerFn) {
	if len(strings.TrimSpace(name)) == 0 || providor == nil {
		panic(fmt.Errorf("name was empty or providor was nil"))
	}
	_, ok := containerProvidors[name]
	if ok {
		panic(fmt.Errorf("the container providor:%s was already existed", name))
	}
	containerProvidors[name] = providor
}
