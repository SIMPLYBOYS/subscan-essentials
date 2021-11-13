package service

import (
	"strings"

	"github.com/prometheus/common/log"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/itering/substrate-api-rpc/metadata"
	"github.com/itering/substrate-api-rpc/rpc"
)

type runtimeService struct {
	SqlRepository model.SqlRepository
}

type RuntimeConfig struct {
	SqlRepository model.SqlRepository
}

func NewRunTimeService(c *RuntimeConfig) model.RuntimeService {
	return &runtimeService{
		SqlRepository: c.SqlRepository,
	}
}

var (
	runtimeSpecs []int
)

func (r *runtimeService) RegCodecMetadata(hash ...string) (coded string, err error) {
	// for i := 0; i < retry; i++ {
	// 	if coded, err = rpc.GetMetadataByHash(nil, hash...); err == nil {
	// 		log.Info("hash: ", hash)
	// 		break
	// 	}
	// 	fmt.Fprintf(os.Stderr, "Request GetMetadataByHash error: %+v\n", err)
	// 	fmt.Fprintf(os.Stderr, "Retrying GetMetadataByHash in %v\n", 10*time.Second)
	// 	time.Sleep(10 * time.Second)
	// }
	coded, err = rpc.GetMetadataByHash(nil, hash...)
	if err != nil {
		return "", err
	}
	return coded, nil
}

func (r *runtimeService) SubstrateRuntimeList() []model.RuntimeVersion {
	return r.SqlRepository.RuntimeVersionList()
}

func (r *runtimeService) GetMetadataInstant(spec int, hash string) (metadataInstant *metadata.Instant, err error) {
	metadataInstant, ok := metadata.RuntimeMetadata[spec]
	// log.Info("=== GetMetadataInstant ===")
	// log.Info("ok: ", ok)
	if !ok {
		raw := r.SqlRepository.RuntimeVersionRaw(spec)
		if raw.Raw == "" {
			if raw.Raw, err = r.RegCodecMetadata(hash); err == nil && raw.Raw != "" {
				affected := r.SqlRepository.SetRuntimeData(spec, "", raw.Raw)
				log.Info("\n\n affected: ", affected, " \n\n")
			} else {
				return nil, err
			}
		}
		// log.Info("raw.Raw: ", raw.Raw)
		metadataInstant = metadata.Process(raw)
	}

	return
}

func (r *runtimeService) SubstrateRuntimeInfo(spec int) *metadata.Instant {
	if metadataInstant, ok := metadata.RuntimeMetadata[spec]; ok {
		return metadataInstant
	}
	runtime := metadata.Process(r.SqlRepository.RuntimeVersionRaw(spec))
	if runtime == nil {
		return metadata.Latest(nil)
	}
	return runtime
}

func (r *runtimeService) RegRuntimeVersion(name string, spec int, hash ...string) error {
	if util.IntInSlice(spec, runtimeSpecs) {
		return nil
	}
	if affected := r.SqlRepository.CreateRuntimeVersion(name, spec); affected > 0 {
		if coded, err := r.RegCodecMetadata(hash...); coded != "" && err == nil {
			runtime := metadata.RegNewMetadataType(spec, coded)
			r.SetRuntimeData(spec, runtime, coded)
		} else {
			log.Error(err)
			panic("get runtime metadata error")
		}
	}
	runtimeSpecs = append(runtimeSpecs, spec)
	return nil
}

func (r *runtimeService) SetRuntimeData(spec int, runtime *metadata.Instant, rawData string) int64 {
	var modules []string
	for _, value := range runtime.Metadata.Modules {
		modules = append(modules, value.Name)
	}
	return r.SqlRepository.SetRuntimeData(spec, strings.Join(modules, "|"), rawData)
}
