package main

import (
	"github.com/kroderdev/vcluster-vnode-plugin/hooks"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	_ = plugin.MustInit()
	plugin.MustRegister(hooks.NewVNodePodHook())
	plugin.MustStart()
}
