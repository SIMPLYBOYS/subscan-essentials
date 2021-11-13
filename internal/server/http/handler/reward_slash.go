package handler

import (
	"net/http"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins"
	b "github.com/CoolBitX-Technology/subscan/plugins/reward/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (h *Handler) rewardlist(c *gin.Context) {
	p := new(struct {
		Row     int    `json:"row" validate:"min=1,max=100"`
		Page    int    `json:"page" validate:"min=0"`
		Adderss string `json:"address" validate:"omitempty,len=48"`
	})

	if err := c.MustBindWith(p, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, model.R{
			Message:     err.Error(),
			GeneratedAt: time.Now().UTC().Unix(),
			Code:        model.QueryBindingError,
		})
		return
	}

	if p.Adderss == "" || ss58.Decode(p.Adderss, util.StringToInt(util.AddressType)) == "" {
		c.JSON(http.StatusBadRequest, model.R{
			Message:     "Invalid address",
			GeneratedAt: time.Now().UTC().Unix(),
			Code:        model.AddressValidateError,
		})
		return
	}

	list, nonce, e := plugins.RegisteredPlugins["reward"].(b.RewardDelivery).RewardList(p.Page, p.Row, p.Adderss)

	if e != nil {
		c.JSON(http.StatusInternalServerError, model.R{
			Message:     e.Error(),
			GeneratedAt: time.Now().UTC().Unix(),
			Code:        model.DataBaseError,
			Data:        e,
		})
		return
	}

	data := map[string]interface{}{
		"list":  list,
		"count": nonce,
	}

	c.JSON(http.StatusOK, model.R{
		Message:     "Success",
		GeneratedAt: time.Now().UTC().Unix(),
		Code:        model.Ok,
		Data:        data,
	})
}
