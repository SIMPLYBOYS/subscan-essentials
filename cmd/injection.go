package main

import (
	"log"

	"github.com/CoolBitX-Technology/subscan/internal/repository"
	"github.com/CoolBitX-Technology/subscan/internal/service"
	"github.com/CoolBitX-Technology/subscan/model"
)

func inject(d *dataSources) (error, model.CommonService, model.BlockService, model.ExtrinsicService, model.EventService, model.RuntimeService, model.PluginService, model.SubscribeService, model.RepairService, model.RedisRepository, model.SqlRepository) {
	log.Println("Injecting data sources")
	redisRepository := repository.NewRedisRepository(d.Redis)
	sqlRepository := repository.NewSqlRepository(d.DB)
	DbStorage := service.NewDbStorage(d.DB)
	done := make(chan struct{})

	runtimeService := service.NewRunTimeService(&service.RuntimeConfig{
		SqlRepository: sqlRepository,
	})

	commonService := service.NewCommonService(&service.CommonConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
	}, runtimeService)

	pluginService := service.NewPluginService(&service.PluginConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
		DbStorage:       DbStorage,
	}, commonService)

	extrinsicService := service.NewExtrinsicService(&service.ExtrinsicConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
	}, pluginService)

	eventService := service.NewEventService(&service.EventConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
	}, pluginService)

	blockService := service.NewBlockService(&service.BlockConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
	}, runtimeService, extrinsicService, eventService, commonService)

	subscribeService := service.NewSubscribeService(&service.SubscribeConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
	}, done, commonService, runtimeService, blockService)

	repairService := service.NewRepairService(&service.RepairConfig{
		RedisRepository: redisRepository,
		SqlRepository:   sqlRepository,
		DbStorage:       DbStorage,
	}, done, commonService, runtimeService, blockService, pluginService)

	return nil, commonService, blockService, extrinsicService, eventService, runtimeService, pluginService, subscribeService, repairService, redisRepository, sqlRepository
}
