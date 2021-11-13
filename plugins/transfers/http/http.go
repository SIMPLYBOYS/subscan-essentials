package http

import (
	"encoding/json"
	"net/http"

	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/CoolBitX-Technology/subscan/util/validator"
	"github.com/itering/subscan-plugin/router"
	"github.com/pkg/errors"
)

var (
	svc model.TransferService
)

func Router(s model.TransferService) []router.Http {
	svc = s
	return []router.Http{
		{"transfers", transfers},
	}
}

func transfers(w http.ResponseWriter, r *http.Request) error {
	p := new(struct {
		Row     int    `json:"row" validate:"min=1,max=100"`
		Page    int    `json:"page" validate:"min=0"`
		Adderss string `json:"address"`
	})
	if err := validator.Validate(r.Body, p); err != nil {
		toJson(w, 10001, nil, err)
		return err
	}
	if p.Adderss == "" || ss58.Decode(p.Adderss, util.StringToInt(util.AddressType)) == "" {
		toJson(w, 10001, nil, nil)
		return nil
	}
	list, count := svc.GetTransfersListJson(p.Page, p.Row, p.Adderss)
	var f, t string
	for i, tx := range list {
		f = ss58.Encode(tx.FromAddr, util.StringToInt(util.AddressType))
		t = ss58.Encode(tx.ToAddr, util.StringToInt(util.AddressType))
		list[i].FromAddr = f
		list[i].ToAddr = t
	}
	toJson(w, 0, map[string]interface{}{
		"transfers": list, "count": count,
	}, nil)
	return nil
}

type J struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	TTL     int         `json:"ttl"`
	Data    interface{} `json:"data,omitempty"`
}

func (j J) Render(w http.ResponseWriter) error {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
	}
	return nil
}

func (j J) WriteContentType(w http.ResponseWriter) {
	var (
		jsonBytes []byte
		err       error
	)
	_ = j.Render(w)
	if jsonBytes, err = json.Marshal(j); err != nil {
		_ = errors.WithStack(err)
		return
	}
	if _, err = w.Write(jsonBytes); err != nil {
		_ = errors.WithStack(err)
	}
}

func toJson(w http.ResponseWriter, code int, data interface{}, err error) {
	j := J{
		Message: "Success",
		TTL:     1,
		Data:    data,
	}
	if err != nil {
		j.Message = err.Error()
	}
	if code != 0 {
		j.Code = code
	}
	j.WriteContentType(w)
	_ = j.Render(w)
}
