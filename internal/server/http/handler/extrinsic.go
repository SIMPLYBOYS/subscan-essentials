package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (h *Handler) extrinsics(c *gin.Context) {
	p := new(struct {
		Row     int    `json:"row" validate:"min=1,max=100"`
		Page    int    `json:"page" validate:"min=0"`
		Signed  string `json:"signed" validate:"omitempty"`
		Address string `json:"address" validate:"omitempty"`
		Module  string `json:"module" validate:"omitempty"`
		Call    string `json:"call" validate:"omitempty"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	var query []string
	if p.Module != "" {
		query = append(query, fmt.Sprintf("call_module = '%s'", p.Module))
	}
	if p.Call != "" {
		query = append(query, fmt.Sprintf("call_module_function = '%s'", p.Call))
	}

	if p.Signed == "signed" {
		query = append(query, "is_signed = 1")
	}

	if p.Address != "" {
		account := ss58.Decode(p.Address, util.StringToInt(util.AddressType))
		if account == "" {
			c.JSON(http.StatusBadRequest, model.R{
				Message:     "Success",
				GeneratedAt: time.Now().UTC().Unix(),
				Data:        util.InvalidAccountAddress,
			})
			return
		}
		query = append(query, fmt.Sprintf("is_signed = 1 and account_id = '%s'", account))
	}

	extrinsics, count := h.ExtrinsicService.GetExtrinsicList(p.Page, p.Row, "desc", query...)

	c.JSON(http.StatusOK, model.R{
		Message:     "Success",
		GeneratedAt: time.Now().UTC().Unix(),
		Data: map[string]interface{}{
			"extrinsics": extrinsics, "count": count,
		},
	})
}

func (h *Handler) extrinsic(c *gin.Context) {

	p := new(struct {
		ExtrinsicIndex string `json:"extrinsic_index" validate:"omitempty"`
		Hash           string `json:"hash" validate:"omitempty,len=66"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	if p.ExtrinsicIndex == "" && p.Hash == "" {
		c.JSON(http.StatusBadRequest, util.ParamsError)
		return
	}

	if p.ExtrinsicIndex != "" {
		c.JSON(http.StatusOK, model.R{
			Message:     "Success",
			GeneratedAt: time.Now().UTC().Unix(),
			Data:        h.ExtrinsicService.GetExtrinsicByIndex(p.ExtrinsicIndex),
		})
	} else {
		c.JSON(http.StatusOK, model.R{
			Message:     "Success",
			GeneratedAt: time.Now().UTC().Unix(),
			Data:        h.ExtrinsicService.GetExtrinsicDetailByHash(p.Hash),
		})
	}
}
