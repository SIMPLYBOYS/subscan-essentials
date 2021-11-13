package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins"
	b "github.com/CoolBitX-Technology/subscan/plugins/bond/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type Handler struct {
	CommonService    model.CommonService
	BlockService     model.BlockService
	ExtrinsicService model.ExtrinsicService
	EventService     model.EventService
	RuntimeService   model.RuntimeService
	BondService      b.BondDelivery
}

type Config struct {
	R                *gin.Engine
	CommonService    model.CommonService
	BlockService     model.BlockService
	ExtrinsicService model.ExtrinsicService
	EventService     model.EventService
	RuntimeService   model.RuntimeService
	BondService      b.BondDelivery
}

func NewHandler(c *Config) {
	h := &Handler{
		CommonService:    c.CommonService,
		BlockService:     c.BlockService,
		ExtrinsicService: c.ExtrinsicService,
		EventService:     c.EventService,
		RuntimeService:   c.RuntimeService,
		BondService:      c.BondService,
	}

	f := c.R.Group("/")
	{
		f.GET("health", h.systemHealth)
	}

	g := c.R.Group("/api")
	{
		g.POST("/now", h.now)
		g.GET("/system/status", h.systemHealth)
		s := g.Group("/scan")
		{
			s.POST("metadata", h.metadata)
			s.POST("blocks", h.blocks)
			s.POST("block", h.block)
			s.POST("extrinsics", h.extrinsics)
			s.POST("extrinsic", h.extrinsic)
			s.POST("events", h.events)
			s.POST("check_hash", h.checkSearchHash)
			s.POST("runtime/metadata", h.runtimeMetadata)
			s.POST("runtime/list", h.runtimeList)
			s.POST("account/reward_slash", h.rewardlist)
			s.POST("plugins", h.pluginList)
			s.POST("transfers", h.transfers) // not include utility.batch event transfer records yet
			s.POST("bond_list", h.bondlist)
		}
		j := g.Group("open/account")
		{
			j.POST("extrinsics", h.extrinsics)
		}
		k := g.Group("wallet")
		{
			k.POST("bond_list", h.bondlist)
		}
		pluginRouter(g)
	}
}

func pluginRouter(g *gin.RouterGroup) {
	for name, plugin := range plugins.RegisteredPlugins {
		for _, r := range plugin.InitHttp() {
			g.Group("plugin").Group(name).POST(r.Router, func(context *gin.Context) {
				_ = r.Handle(context.Writer, context.Request)
			})
		}
	}
}

func (h *Handler) systemHealth(c *gin.Context) {
	status := h.CommonService.DaemonHealth(c)
	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}

func (h *Handler) metadata(c *gin.Context) {
	metadata, err := h.CommonService.Metadata()
	if err == nil {
		c.JSON(http.StatusOK, model.R{
			Message:     "Success",
			GeneratedAt: time.Now().UTC().Unix(),
			Code:        model.Ok,
			Data:        metadata,
		})
	} else {
		c.JSON(http.StatusBadRequest, err)
	}
}

func (h *Handler) now(c *gin.Context) {
	c.JSON(http.StatusOK, time.Now().Unix())
}

func (h *Handler) events(c *gin.Context) {
	p := new(struct {
		Row    int    `json:"row" validate:"min=1,max=100"`
		Page   int    `json:"page" validate:"min=0"`
		Module string `json:"module" validate:"omitempty"`
		Call   string `json:"call" validate:"omitempty"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	var query []string
	if p.Module != "" {
		query = append(query, fmt.Sprintf("module_id = '%s'", p.Module))
	}
	if p.Call != "" {
		query = append(query, fmt.Sprintf("event_id = '%s'", p.Call))
	}
	events, count := h.EventService.RenderEvents(p.Page, p.Row, "desc", query...)
	c.JSON(http.StatusOK, map[string]interface{}{
		"events": events, "count": count,
	})
}

func (h *Handler) checkSearchHash(c *gin.Context) {
	p := new(struct {
		Hash string `json:"hash" validate:"len=66"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	if block := h.BlockService.GetBlockByHash(p.Hash); block != nil {
		c.JSON(http.StatusOK, map[string]string{"hash_type": "block"})
		return
	}
	if extrinsic := h.ExtrinsicService.GetExtrinsicByHash(p.Hash); extrinsic != nil {
		c.JSON(http.StatusOK, map[string]string{"hash_type": "extrinsic"})
		return
	}
	c.JSON(http.StatusOK, util.RecordNotFound)
}

func (h *Handler) runtimeList(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"list": h.RuntimeService.SubstrateRuntimeList(),
	})
}

func (h *Handler) runtimeMetadata(c *gin.Context) {
	p := new(struct {
		Spec int `json:"spec"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	if info := h.RuntimeService.SubstrateRuntimeInfo(p.Spec); info == nil {
		c.JSON(http.StatusOK, map[string]interface{}{"info": nil})
	} else {
		c.JSON(http.StatusOK, map[string]interface{}{
			"info": info.Metadata.Modules,
		})
	}
}

func (h *Handler) pluginList(c *gin.Context) {
	c.JSON(http.StatusOK, plugins.List())
}
