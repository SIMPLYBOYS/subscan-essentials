package service

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/itering/substrate-api-rpc/websocket"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/common/log"
)

type DbStorage struct {
	db     *gorm.DB
	Prefix string
}

func NewDbStorage(d *gorm.DB) *DbStorage {
	return &DbStorage{
		db: d,
	}
}

var protectedTables []string

func (d *DbStorage) SetPrefix(prefix string) {
	d.Prefix = prefix
}

func (d *DbStorage) GetPrefix() string {
	return d.Prefix
}

func (d *DbStorage) SpecialMetadata(spec int) string {
	var raw model.RuntimeVersion
	if query := d.db.Where("spec_version = ?", spec).First(&raw); query.RecordNotFound() {
		return ""
	}
	return raw.RawData
}

func (d *DbStorage) getModelTableName(model interface{}) string {
	return d.db.Unscoped().NewScope(model).TableName()
}

func (d *DbStorage) checkProtected(model interface{}) error {
	if util.StringInSlice(d.getModelTableName(model), protectedTables) {
		return errors.New("protected tables")
	}
	return nil
}

func (d *DbStorage) RPCPool() *websocket.PoolConn {
	conn, _ := websocket.Init()
	return conn
}

func (d *DbStorage) DbBegin() *model.GormDB {
	txn := d.db.Begin()
	if txn.Error != nil {
		panic(txn.Error)
	}
	return &model.GormDB{txn, false}
}

func (d *DbStorage) DbRollback(c *model.GormDB) {
	if c.GdbDone {
		return
	}
	tx := c.Rollback()
	c.GdbDone = true
	if err := tx.Error; err != nil && err != sql.ErrTxDone {
		log.Error("Fatal error DbRollback", err)
	}
}

func (d *DbStorage) DbCommit(c *model.GormDB) {
	if c.GdbDone {
		return
	}
	tx := c.Commit()
	c.GdbDone = true
	if err := tx.Error; err != nil && err != sql.ErrTxDone {
		log.Error("Fatal error DbCommit", err)
	}
}

func (d *DbStorage) getPluginPrefixTableName(instant interface{}) string {
	tableName := d.getModelTableName(instant)

	if util.StringInSlice(tableName, protectedTables) {
		return tableName
	}
	return fmt.Sprintf("%s_%s", d.GetPrefix(), tableName)
}

func (d *DbStorage) FindBy(record interface{}, query interface{}, option *model.Option) error {
	tx := d.db
	switch v := query.(type) {
	case []string:
		for i, q := range v {
			if i == 0 {
				tx = tx.Where(q)
			} else {
				tx = tx.Or(q)
			}
		}
	case map[string]interface{}:
		tx = tx.Where(query)
	default:
		// handle unknown type
	}

	// plugin prefix table
	if option != nil && option.PluginPrefix != "" {
		tx = tx.Table(fmt.Sprintf("%s_%s", option.PluginPrefix, d.getModelTableName(record)))
		if (option.Page > 0) && (option.PageSize > 0) {
			tx = tx.Limit(option.PageSize).Offset((option.Page - 1) * option.PageSize)
		}
		if option.Order != "" {
			tx = tx.Order(option.Order)
		}
	}

	// rows count
	// tx.Count(&count) TODO count should be get from account table

	// pagination
	if option != nil {
		// default page limit 1000
		if option.PageSize == 0 {
			option.PageSize = 1000
		}
		tx = tx.Offset(option.Page * option.PageSize).Limit(option.PageSize)
	}

	tx = tx.Find(record)
	return tx.Error
}

func (d *DbStorage) AutoMigration(model interface{}) error {
	log.Info("--- AutoMigration ---", d.getPluginPrefixTableName(model))
	if d.checkProtected(model) == nil {
		tx := d.db.Table(d.getPluginPrefixTableName(model)).Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(model)
		return tx.Error
	}
	return nil
}

func (d *DbStorage) AddIndex(model interface{}, indexName string, columns ...string) error {
	if d.checkProtected(model) == nil {
		tx := d.db.Table(d.getPluginPrefixTableName(model)).AddIndex(indexName, columns...)
		return tx.Error
	}
	return nil
}

func (d *DbStorage) Create(txn *model.GormDB, record interface{}) *model.GormDB {
	if err := d.checkProtected(record); err == nil {
		tx := txn.Table(d.getPluginPrefixTableName(record)).Create(record)
		return &model.GormDB{tx, false}
	} else {
		log.Error(err)
		return nil
	}
}

func (d *DbStorage) Update(txn *model.GormDB, record interface{}, query interface{}, attr map[string]interface{}) *model.GormDB {
	if err := d.checkProtected(record); err == nil {
		tx := txn.Table(d.getPluginPrefixTableName(record))
		switch v := query.(type) {
		case []string:
			for _, q := range v {
				tx = tx.Where(q)
			}
		case string:
			tx.Where(query)
		default:
			// handle unknown type
		}
		tx.Updates(attr)
		return &model.GormDB{tx, false}
	} else {
		log.Error(err)
		return nil
	}
}

func (d *DbStorage) Delete(model interface{}, query interface{}) error {
	if err := d.checkProtected(model); err == nil {
		tx := d.db.Table(d.getPluginPrefixTableName(model)).Where(query).Delete(model)
		return tx.Error
	} else {
		return err
	}
}

func (d *DbStorage) AddUniqueIndex(model interface{}, indexName string, columns ...string) error {
	if d.checkProtected(model) == nil {
		tx := d.db.Table(d.getPluginPrefixTableName(model)).AddUniqueIndex(indexName, columns...)
		return tx.Error
	}
	return nil
}
