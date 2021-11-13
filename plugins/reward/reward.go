package reward

import (
	"fmt"
	"strings"

	m "github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/model"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/repository"
	"github.com/CoolBitX-Technology/subscan/plugins/reward/service"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/CoolBitX-Technology/subscan/util/ss58"
	"github.com/itering/subscan-plugin/router"
	"github.com/prometheus/common/log"
	"github.com/shopspring/decimal"
)

var srv model.RewardService

type Reward struct {
	d m.Dao
}

func New() *Reward {
	return &Reward{}
}

func (r *Reward) InitDao(d m.Dao) {
	s := repository.NewsqlRewardRepository(d)
	srv = service.New(s)
	r.d = d
	r.Migrate()
}

func (r *Reward) Migrate() {
	var e error
	if e = r.d.AutoMigration(&model.Reward{}); e != nil {
		log.Error(e)
	}

	// if e = r.d.AutoMigration(&model.Account{}); e != nil {
	// 	log.Error(e)
	// }

	if e = r.d.AddUniqueIndex(&model.Reward{}, "reward_index", "event_index", "event_idx"); e != nil {
		log.Error(e)
	}

	if e = r.d.AddIndex(&model.Reward{}, "account_w_event", "account_id", "event_index"); e != nil {
		log.Error(e)
	}
}

func (r *Reward) InitHttp() []router.Http {
	return nil
}

func (r *Reward) RewardList(page int, row int, address string) (rewardList []model.Reward, nonce int, err error) {
	rewardList, err = srv.GetRewardListJson(page, row, address)
	nonce, err = srv.GetAccountNonce(address)

	if err != nil {
		return nil, 0, err
	}

	for i, reward := range rewardList {
		rewardList[i].AccountId = ss58.Encode(reward.AccountId, util.StringToInt(util.AddressType))
	}

	return rewardList, nonce, err
}

func (r *Reward) ProcessExtrinsic(block *m.Block, e *m.Extrinsic, p []m.Event) error {
	return nil
}

func (r *Reward) ProcessEvent(block *m.Block, e *m.Event, fee decimal.Decimal) error {
	var err error
	var paramEvent []m.EventParam
	util.UnmarshalAny(&paramEvent, e.Params)
	c := fmt.Sprintf("%s-%s", strings.ToLower(e.ModuleId), strings.ToLower(e.EventId))
	log.Info(c)
	switch c {
	case strings.ToLower("Staking-Reward"), strings.ToLower("Staking-Slash"):
		if err = srv.NewRewardEvent(block, e, paramEvent); err != nil {
			log.Error(err)
		}
		break
	}
	return err
}

func (r *Reward) Version() string {
	return "0.1"
}

func (r *Reward) SubscribeExtrinsic() []string {
	return nil
}

func (r *Reward) SubscribeEvent() []string {
	return []string{"staking"}
}
