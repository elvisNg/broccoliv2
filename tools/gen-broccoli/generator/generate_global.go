package generator

func GenerateGlobal(PD *Generator, rootdir string) (err error) {
	err = genGlobal(PD, rootdir)
	if err != nil {
		return
	}
	err = genGlobalInit(PD, rootdir)
	if err != nil {
		return
	}

	return
}

func genGlobalInit(PD *Generator, rootdir string) error {
	header := _defaultHeader
	context := `package global

import (
	"log"

	"github.com/elvisNg/broccoliv2/config"
	"github.com/elvisNg/broccoliv2/engine"
	"github.com/elvisNg/broccoliv2/service"
)

var ng engine.Engine
var ServiceOpts = []service.Option{
	service.WithLoadEngineFnOption(func(ng engine.Engine) {
		log.Println("WithLoadEngineFnOption: SetNG success.")
		SetNG(ng)
        loadEngineSuccess(ng)
	}),
}

func init() {
	// load engine
	//loadEngineFnOpt := service.WithLoadEngineFnOption(func(ng engine.Engine) {
	//	log.Println("WithLoadEngineFnOption: SetNG success.")
	//	SetNG(ng)
	//	loadEngineSuccess(ng)
	//})
	processChangeFnOpt := service.WithProcessChangeFnOption(func(event interface{}) {
		processChange(event)
	})
	ServiceOpts = append(ServiceOpts, processChangeFnOpt)

	// // server wrap
	// ServiceOpts = append(ServiceOpts, service.WithGoMicroServerWrapGenerateFnOption(gomicro.GenerateServerLogWrap))
}

// GetNG ...
func GetNG() engine.Engine {
	return ng
}

// SetNG ...
func SetNG(n engine.Engine) {
	ng = n
}

// GetConfig ...
func GetConfig() (conf *config.AppConf) {
	c, err := ng.GetConfiger()
	if err != nil {
		log.Println("global.GetConfig err:", err)
		return
	}
	conf = c.Get()
	return
}

`
	fn := GetTargetFileName(PD, "global.init", rootdir)
	return writeContext(fn, header, context, true)
}

func genGlobal(PD *Generator, rootdir string) error {
	header := ``
	context := `package global
import (
    "github.com/elvisNg/broccoliv2/config"
    "github.com/elvisNg/broccoliv2/engine"
)

func loadConfig(conf *config.AppConf) {
    // 加载配置
    // TODO: do something here
}

func loadEngineSuccess(ng engine.Engine) {
    loadConfig(GetConfig())
    // 加载engine成功
    // TODO: do something here
}

func processChange(event interface{}) {
    loadConfig(GetConfig())
    // 配置变更
    // TODO: do something here
}

`
	fn := GetTargetFileName(PD, "global", rootdir)
	return writeContext(fn, header, context, false)
}
