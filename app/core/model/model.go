package model

import (
	"github.com/dulumao/Guten-framework/app/core/adapter/database"
	"github.com/dulumao/Guten-utils/paginater"
	"github.com/jinzhu/gorm"
)

type IModel interface {
	TableName() string
}

type Model struct {
	instance IModel
}

func (self *Model) With(instance IModel) *Model {
	self.instance = instance

	return self
}

func (self *Model) GetDB(transaction ...bool) *gorm.DB {
	return database.DB
}

func (self *Model) Create() error {
	return self.GetDB().Table(self.instance.TableName()).Create(self.instance).Error
}

func (self *Model) FirstOrCreate(out interface{}) error {
	/*if err := self.First(out); err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return self.GetDB().Create(out).Error
		}
	}

	return nil*/
	return database.DB.Where(self.instance).FirstOrCreate(out).Error
}

func (self *Model) FirstOrInit(out interface{}) error {
	return database.DB.Where(self.instance).FirstOrInit(out).Error
}

func (self *Model) Update(attrs ...interface{}) error {
	return self.GetDB().Table(self.instance.TableName()).Where(self.instance).Update(attrs...).Error
}

func (self *Model) Updates(attrs interface{}) error {
	return self.GetDB().Table(self.instance.TableName()).Where(self.instance).Updates(attrs).Error
}

func (self *Model) Delete(unscoped ...bool) error {
	var query = self.GetDB().Table(self.instance.TableName())

	if len(unscoped) > 0 {
		if unscoped[0] {
			query = query.Unscoped()
		}
	}

	return query.Delete(self.instance).Error
}

// func (self *Model) Delete(wheres ...[]func(*gorm.DB) *gorm.DB) error {
// 	/*var query = self.GetDB().Table(self.instance.TableName()).Where(self.instance)
//
// 	if len(unscoped) > 0 {
// 		if unscoped[0] {
// 			query = query.Unscoped()
// 		}
// 	}
//
// 	return query.Delete(nil).Error*/
// 	var query = self.GetDB().Table(self.instance.TableName())
//
// 	if len(wheres) > 0 {
// 		for _, scope := range wheres[0] {
// 			query = query.Scopes(scope)
// 		}
// 	}
//
// 	return query.Delete(self.instance).Error
// }

func (self *Model) Count(wheres ...[]func(*gorm.DB) *gorm.DB) int {
	var count int
	var query = self.GetDB().Table(self.instance.TableName())

	if len(wheres) > 0 {
		for _, scope := range wheres[0] {
			query = query.Scopes(scope)
		}
	}

	query.Count(&count)

	return count
}

func (self *Model) First(out interface{}) error {
	if err := self.GetDB().Table(self.instance.TableName()).Where(self.instance).First(out).Error; err != nil {
		return err
	}

	return nil
}

func (self *Model) Get(out interface{}, wheres ...[]func(*gorm.DB) *gorm.DB) error {
	var query = self.GetDB().Table(self.instance.TableName())

	if len(wheres) > 0 {
		for _, scope := range wheres[0] {
			query = query.Scopes(scope)
		}
	}

	if err := query.Find(out).Error; err != nil {
		return err
	}

	return nil
}

func (self *Model) Transaction(callback func(tx *gorm.DB)) error {
	var err error

	tx := database.DB.Begin()

	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			err = tx.Rollback().Error
		}
	}()

	callback(tx)

	err = tx.Commit().Error

	return err
}

func (self *Model) Paginate(pagingNum, current, numPages int, wheres ...[]func(*gorm.DB) *gorm.DB) *paginater.Paginater {
	return paginater.New(self.Count(wheres ...), pagingNum, current, numPages)
}
