package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (h *Handler) blocks(c *gin.Context) {
	p := new(struct {
		Row  int `json:"row" validate:"min=1,max=100"`
		Page int `json:"page" validate:"min=0"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	blockNum, err := h.BlockService.GetCurrentBlockNum(context.TODO())
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
	}
	blks := h.BlockService.GetBlocksSampleByNums(p.Page, p.Row)
	c.JSON(http.StatusOK, map[string]interface{}{
		"blocks": blks, "current": blockNum,
	})
}

func (h *Handler) block(c *gin.Context) {
	p := new(struct {
		BlockNum  int    `json:"block_num" validate:"omitempty,min=0"`
		BlockHash string `json:"block_hash" validate:"omitempty,len=66"`
	})
	if err := c.MustBindWith(p, binding.JSON); err != nil {
		return
	}
	if p.BlockHash == "" {
		c.JSON(http.StatusOK, h.BlockService.GetBlockByNum(p.BlockNum))
	} else {
		c.JSON(http.StatusOK, h.BlockService.GetBlockByHashJson(p.BlockHash))
	}
}
