package bond

import (
	"fmt"
	"strings"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/model"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/repository"
	"github.com/CoolBitX-Technology/subscan/plugins/bond/service"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	ui "github.com/itering/subscan-plugin"
	"github.com/itering/subscan-plugin/router"
	"github.com/prometheus/common/log"
	"github.com/shopspring/decimal"
)

var srv model.BondService

type Bond struct {
	d m.Dao
}

func New() *Bond {
	return &Bond{}
}

func (b *Bond) InitDao(d m.Dao) {
	s := repository.NewsqlBondRepository(d)
	srv = service.New(s)
	b.d = d
	b.Migrate()
}

func (b *Bond) Migrate() {
	var e error
	if e = b.d.AutoMigration(&model.Bond{}); e != nil {
		log.Error(e)
	}
	if e = b.d.AddUniqueIndex(&model.Bond{}, "extrinsic_index", "extrinsic_index"); e != nil {
		log.Error(e)
	}
	if e = b.d.AddIndex(&model.Bond{}, "account_w_start_at", "account", "start_at"); e != nil {
		log.Error(e)
	}
}

func (b *Bond) InitHttp() []router.Http {
	return nil
}

func (b *Bond) BondList(page, row int, addr string, status string, locked int) ([]model.Bond, error) {
	bondlist, err := srv.GetBondListJson(page, row, addr, status, locked)

	if err != nil {
		return nil, err
	}

	for i, bond := range bondlist {
		bondlist[i].Account = ss58.Encode(bond.Account, util.StringToInt(util.AddressType))
	}

	return bondlist, err
}

func (b *Bond) ProcessExtrinsic(block *m.Block, e *m.Extrinsic, p []m.Event) error {
	var err error
	var paramExtrinsic []m.ExtrinsicParam
	util.UnmarshalAny(&paramExtrinsic, e.Params)
	c := fmt.Sprintf("%s-%s", strings.ToLower(e.CallModule), strings.ToLower(e.CallModuleFunction))
	log.Info(c)
	switch c {
	case strings.ToLower("Staking-Bond"), strings.ToLower("Staking-Bond_Extra"), strings.ToLower("Staking-Rebond"):
		if err = srv.NewBondExtrinsic(block, e, paramExtrinsic, "bonded"); err != nil {
			log.Error(err)
		}
		break
	case strings.ToLower("Staking-Unbond"):
		if err = srv.NewBondExtrinsic(block, e, paramExtrinsic, "unbonding"); err != nil {
			log.Error(err)
		}
		break
	}

	if err != nil {
		return err
	}

	return nil
}

func (b *Bond) ProcessEvent(block *m.Block, event *m.Event, fee decimal.Decimal) error {
	return nil
}

func (b *Bond) Version() string {
	return "0.1"
}

func (b *Bond) UiConf() *ui.UiConfig {
	return nil
}

func (b *Bond) SubscribeExtrinsic() []string {
	return []string{"staking"}
}

func (b *Bond) SubscribeEvent() []string {
	return []string{"balances"}
}
