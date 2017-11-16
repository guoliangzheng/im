package gate

/* gateway启动主模块 */
import (
	"log"
	"time"

	"im.zgl/gomqtt/uuid"
)

type Gate struct {
}

func New() *Gate {
	return &Gate{}
}

func (g *Gate) Start(isStatic bool) {

	// init configurations
	err := loadConfig(isStatic)
	if err != nil {
		log.Fatal("config load error: %v", err)
	}

	// wait for rpc connections inited
	time.Sleep(6 * time.Second)

	// start uuid service
	uuid.Init(Conf.Gateway.ServerId, "ts16", Logger)

	// init providers
	providersStart()

	// init addmin service
	// go adminStart()

	// start the monitors
	// monitorsStart()

	Logger.Info("gate started!!")
}
