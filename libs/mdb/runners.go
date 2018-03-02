package mdb

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (self *MdbSession) DB(s *mgo.Session) *mgo.Database {
	return s.DB(self.db)
}

func (self *MdbSession) WithC(collection string, job func(*mgo.Collection) error) error {
	s := self.session.New()
	defer s.Close()
	return job(s.DB(self.db).C(collection))
}

func (self *MdbSession) Upsert(collection string, selector interface{}, change interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		_, err := c.Upsert(selector, change)
		return err
	})
}

func (self *MdbSession) UpdateId(collection string, id interface{}, change interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.UpdateId(id, change)
	})
}
func (self *MdbSession) Update(collection string, selector, change interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Update(selector, change)
	})
}
func (self *MdbSession) UpdateAll(collection string, selector, change interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		_, err := c.UpdateAll(selector, change)
		return err
	})
}

func (self *MdbSession) Insert(collection string, data ...interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Insert(data...)
	})
}

func (self *MdbSession) All(collection string, query interface{}, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(query).All(result)
	})
}

// 返回所有复合 query 条件的item， 并且被 projection 限制返回的fields
func (self *MdbSession) AllSelect(collection string, query interface{}, projection interface{}, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(query).Select(projection).All(result)
	})
}

func (self *MdbSession) One(collection string, query interface{}, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(query).One(result)
	})
}

func (self *MdbSession) OneSelect(collection string, query interface{}, projection interface{}, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(query).Select(projection).One(result)
	})
}

//等效于: self.One(collection,bson.M{"_id":id},result)
func (self *MdbSession) FindId(collection string, id interface{}, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(bson.M{"_id": id}).One(result)
	})
}

func (self *MdbSession) RemoveId(collection string, id interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		err := c.RemoveId(id)
		return err
	})
}
func (self *MdbSession) Remove(collection string, selector interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		err := c.Remove(selector)
		return err
	})
}
func (self *MdbSession) RemoveAll(collection string, selector interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		_, err := c.RemoveAll(selector)
		return err
	})
}

func (self *MdbSession) CountId(collection string, id interface{}) (n int) {
	self.WithC(collection, func(c *mgo.Collection) error {
		var err error
		n, err = c.FindId(id).Count()
		return err
	})
	return n
}
func (self *MdbSession) Count(collection string, query interface{}) (n int) {
	self.WithC(collection, func(c *mgo.Collection) error {
		var err error
		n, err = c.Find(query).Count()
		return err
	})
	return n
}
func (self *MdbSession) Exist(collection string, query interface{}) bool {
	return self.Count(collection, query) != 0
}
func (self *MdbSession) ExistId(collection string, id interface{}) bool {
	return self.CountId(collection, id) != 0
}

func (self *MdbSession) Page(collection string, query interface{}, offset int, limit int, result interface{}) error {
	return self.WithC(collection, func(c *mgo.Collection) error {
		return c.Find(query).Skip(offset).Limit(limit).All(result)
	})
}

//获取页面数据和“所有”符合条件的记录“总共”的条数
func (self *MdbSession) PageAndCount(collection string, query interface{}, offset int, limit int, result interface{}) (total int, err error) {
	err = self.WithC(collection, func(c *mgo.Collection) error {
		total, err = c.Find(query).Count()
		if err != nil {
			return err
		}
		return c.Find(query).Skip(offset).Limit(limit).All(result)
	})
	return total, err
}

func (self *MdbSession) PageSortAndCount(collection string, query interface{}, sort string, offset int, limit int, result interface{}) (total int, err error) {
	err = self.WithC(collection, func(c *mgo.Collection) error {
		total, err = c.Find(query).Count()
		if err != nil {
			return err
		}
		return c.Find(query).Sort(sort).Skip(offset).Limit(limit).All(result)
	})
	return total, err
}

//等同与UpdateId(collection,id,bson.M{"$set":change})
func (self *MdbSession) SetId(collection string, id interface{}, change interface{}) error {
	return self.UpdateId(collection, id, bson.M{"$set": change})
}
