package zmysql

import (
	"github.com/elvisNg/broccoliv2/config"
	"github.com/jinzhu/gorm"
)

type Mysql interface {
	Reload(cfg *config.Mysql)
	GetCli() *gorm.DB
}
