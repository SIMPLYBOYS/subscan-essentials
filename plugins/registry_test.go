package plugins

import (
	"testing"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/itering/subscan-plugin/router"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type TPlugin struct{}

func (a *TPlugin) InitDao(d model.Dao) {}

func (a *TPlugin) InitHttp() []router.Http { return nil }

func (a *TPlugin) ProcessExtrinsic(block *model.Block, extrinsic *model.Extrinsic, events []model.Event) error {
	return nil
}

func (a *TPlugin) ProcessEvent(block *model.Block, event *model.Event, fee decimal.Decimal) error {
	return nil
}

func (a *TPlugin) Migrate() {}

func (a *TPlugin) Version() string { return "0.1" }

func (a *TPlugin) SubscribeExtrinsic() []string { return nil }

func (a *TPlugin) SubscribeEvent() []string { return nil }

func TestRegister(t *testing.T) {
	register("test", &TPlugin{})
	register("test2", nil)
	register("test", &TPlugin{})
	assert.NotNil(t, RegisteredPlugins["test"])
	assert.Nil(t, RegisteredPlugins["test2"])
}

func TestList(t *testing.T) {
	assert.Equal(t, len(List()), len(RegisteredPlugins))
}
