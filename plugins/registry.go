package plugins

import (
	"reflect"
	"strings"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond"
	"github.com/CoolBitX-Technology/subscan/plugins/reward"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers"
	"github.com/prometheus/common/log"
)

type PluginFactory model.Plugin

var RegisteredPlugins = make(map[string]PluginFactory)

// register local plugin
func init() {
	// registerNative(balance.New())
	// registerNative(system.New())
	registerNative(transfers.New())
	registerNative(bond.New())
	registerNative(reward.New())
}

func register(name string, f interface{}) {
	log.Info("register plugins: ", name)
	name = strings.ToLower(name)
	if f == nil {
		return
	}

	if _, ok := RegisteredPlugins[name]; ok {
		return
	}

	if _, ok := f.(PluginFactory); ok {
		RegisteredPlugins[name] = f.(PluginFactory)
	}
}

func registerNative(p interface{}) {
	register(reflect.ValueOf(p).Type().Elem().Name(), p)
}

type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Ui      bool   `json:"ui"`
}

func List() []PluginInfo {
	plugins := make([]PluginInfo, 0, len(RegisteredPlugins))
	for name, plugin := range RegisteredPlugins {
		plugins = append(plugins, PluginInfo{Name: name, Version: plugin.Version()})
	}
	return plugins
}
