package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/itering/substrate-api-rpc"
	"github.com/itering/substrate-api-rpc/metadata"
	"github.com/itering/substrate-api-rpc/rpc"
	"github.com/itering/substrate-api-rpc/storage"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/prometheus/common/log"
)

var onceToken sync.Once

type commonService struct {
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
	RunTimeService  model.RuntimeService
}

type CommonConfig struct {
	RedisRepository model.RedisRepository
	SqlRepository   model.SqlRepository
}

func NewCommonService(c *CommonConfig, r model.RuntimeService) model.CommonService {
	return &commonService{
		RedisRepository: c.RedisRepository,
		SqlRepository:   c.SqlRepository,
		RunTimeService:  r,
	}
}

func (s *commonService) ReadTypeRegistry() ([]byte, error) {
	var configPath string
	for i, arg := range os.Args {
		if arg == "--conf" {
			configPath = os.Args[i+1]
		}
	}
	if configPath == "" {
		configPath = "./configs" // default path of config file
	}
	return ioutil.ReadFile(fmt.Sprintf("%s/source/%s.json", configPath, util.NetworkNode))
}

// read custom registry from local or remote
func (s *commonService) InitSubRuntimeLatest() {
	// reg network custom type
	defer func() {
		go s.unknownToken()
		if c, err := s.ReadTypeRegistry(); err == nil {
			substrate.RegCustomTypes(c)
			if unknown := metadata.Decoder.CheckRegistry(); len(unknown) > 0 {
				log.Warn("Found unknown type ", strings.Join(unknown, ", "))
			}
		} else {
			if os.Getenv("TEST_MOD") != "true" {
				panic(err)
			}
		}
	}()

	// find db
	if recent := s.SqlRepository.RuntimeVersionRecent(); recent != nil && strings.HasPrefix(recent.RawData, "0x") {
		metadata.Latest(&metadata.RuntimeRaw{Spec: recent.SpecVersion, Raw: recent.RawData})
		return
	}
	// find metadata for blockChain

	if raw, err := s.RunTimeService.RegCodecMetadata(); strings.HasPrefix(raw, "0x") && err == nil {
		metadata.Latest(&metadata.RuntimeRaw{Spec: 1, Raw: raw})
		return
	}
	panic("Can not find chain metadata, please check network")
}

func (s *commonService) Close() {
	s.RedisRepository.Close()
	s.SqlRepository.Close()
}

func (s *commonService) DaemonHealth(ctx context.Context) map[string]bool {
	return s.RedisRepository.DaemonHealth(ctx)
}

func (s *commonService) Metadata() (map[string]string, error) {
	c := context.TODO()
	m, err := s.RedisRepository.GetMetadata(c)
	m["networkNode"] = util.NetworkNode
	m["commissionAccuracy"] = util.CommissionAccuracy
	m["balanceAccuracy"] = util.BalanceAccuracy
	m["addressType"] = util.AddressType
	return m, err
}

func (s *commonService) SetHeartBeat(action string) {
	ctx := context.TODO()
	_ = s.RedisRepository.SetHeartBeatNow(ctx, action)
}

func (s *commonService) unknownToken() {
	websocket.SetEndpoint(util.WSEndPoint)
	onceToken.Do(func() {
		if p, _ := rpc.GetSystemProperties(nil); p != nil {
			util.AddressType = util.IntToString(p.Ss58Format)
			util.BalanceAccuracy = util.IntToString(p.TokenDecimals)
		}
	})
}

func (s *commonService) Ping(ctx context.Context, e *empty.Empty) (*empty.Empty, error) {
	return &empty.Empty{}, s.RedisRepository.Ping(ctx)
}

func (s *commonService) ValidatorsList(conn websocket.WsConn, hash string) (validatorList []string) {
	validatorsRaw, _ := rpc.ReadStorage(conn, "Session", "Validators", hash)
	for _, addr := range validatorsRaw.ToStringSlice() {
		validatorList = append(validatorList, util.TrimHex(addr))
	}
	return
}

func (s *commonService) EmitLog(txn *model.GormDB, blockNum int, l []storage.DecoderLog, finalized bool, validatorList []string) (validator string, err error) {
	s.SqlRepository.DropLogsNotFinalizedData(blockNum, finalized)
	for index, logData := range l {
		dataStr := util.ToString(logData.Value)

		ce := model.ChainLog{
			LogIndex:  fmt.Sprintf("%d-%d", blockNum, index),
			BlockNum:  blockNum,
			LogType:   logData.Type,
			Data:      dataStr,
			Finalized: finalized,
		}
		if err = s.SqlRepository.CreateLog(txn, &ce); err != nil {
			return "", err
		}

		// check validator TODO check validator blank case
		if strings.EqualFold(ce.LogType, "PreRuntime") {
			validator = substrate.ExtractAuthor([]byte(dataStr), validatorList)
		}

	}
	return validator, err
}

func (s *commonService) GetCurrentRuntimeSpecVersion(blockNum int) int {
	if util.CurrentRuntimeSpecVersion != 0 {
		return util.CurrentRuntimeSpecVersion
	}
	if block := s.SqlRepository.GetNearBlock(blockNum); block != nil {
		return block.SpecVersion
	}
	return -1
}
