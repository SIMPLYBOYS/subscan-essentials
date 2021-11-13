package transfers

import (
	"fmt"
	"strings"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/http"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/model"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/repository"
	"github.com/CoolBitX-Technology/subscan/plugins/transfers/service"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/itering/subscan-plugin/router"
	"github.com/prometheus/common/log"
	"github.com/shopspring/decimal"
)

var srv model.TransferService

type Transfer struct {
	d m.Dao
}

func New() *Transfer {
	return &Transfer{}
}

func (a *Transfer) InitDao(d m.Dao) {
	s := repository.NewsqlTransferRepository(d)
	srv = service.New(s)
	a.d = d
	a.Migrate()
}

func (a *Transfer) InitHttp() []router.Http {
	return http.Router(srv)
}

func (a *Transfer) ProcessExtrinsic(b *m.Block, e *m.Extrinsic, p []m.Event) error {
	log.Info("=== Transfer ProcessExtrinsic ===")
	var err error
	var paramExtrinsic []m.ExtrinsicParam
	util.UnmarshalAny(&paramExtrinsic, e.Params)
	c := fmt.Sprintf("%s-%s", strings.ToLower(e.CallModule), strings.ToLower(e.CallModuleFunction))
	log.Info(c)
	switch c {
	case strings.ToLower("Balances-Transfer_Keep_Alive"), strings.ToLower("Balances-Transfer"), strings.ToLower("Balances-Transfer_All"):
		log.Info(e.Success)
		// TODO e.Fee == 0 case
		if err = srv.BalancesTransaction(b, e, paramExtrinsic); err != nil {
			log.Error(err)
		}
		break
	}

	if err != nil {
		return err
	}

	return nil
}

func (a *Transfer) ProcessEvent(block *m.Block, event *m.Event, fee decimal.Decimal) error {
	return nil
}

// Plugins version
func (a *Transfer) Version() string {
	return "0.1"
}

func (a *Transfer) TransferList(page int, row int, address string) ([]model.Transfer, error) {
	list, err := srv.GetTransfersListJson(page, row, address)

	if err != nil {
		return nil, err
	}

	for i, tx := range list {
		list[i].FromAddr = ss58.Encode(tx.FromAddr, util.StringToInt(util.AddressType))
		list[i].ToAddr = ss58.Encode(tx.ToAddr, util.StringToInt(util.AddressType))
	}

	return list, err
}

// Subscribe Extrinsic with special module
func (a *Transfer) SubscribeExtrinsic() []string {
	return []string{"sudo", "system", "balances", "utility"}
}

// Subscribe Events with special module
func (a *Transfer) SubscribeEvent() []string {
	return []string{"balances"}
}

func (a *Transfer) Migrate() {
	log.Info("=== Transfer Migrate ===")
	var e error
	if e = a.d.AutoMigration(&model.Transfer{}); e != nil {
		log.Error(e)
	}
	if e = a.d.AddUniqueIndex(&model.Transfer{}, "extrinsic_index", "extrinsic_index"); e != nil {
		log.Error(e)
	}
	if e = a.d.AddIndex(&model.Transfer{}, "idx_num_from_to", "block_num", "from_addr", "to_addr"); e != nil {
		log.Error(e)
	}
}
