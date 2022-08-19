package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/boltdb/bolt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var sillyGirl Bucket = NewBucket("sillyGirl")
var db *bolt.DB

type Bucket string

var Buckets = []Bucket{}

func initStore() {
	var err error
	if runtime.GOOS == "darwin" {
		db, err = bolt.Open("./sillyGirl.cache", 0600, nil)
		logs.Info("打开数据库", `db=`, db, `err=`, err)
	} else {
		db, err = bolt.Open(dataHome+"/sillyGirl.cache", 0600, nil)
		logs.Info("打开数据库", `db=`, db, `err=`, err)
	}
	if err != nil
		panic(err)
	}
}

func (b Bucket) String() string {
	logs.Info(string(b))
	return string(b)
}



func NewBucket(name string) Bucket {
	logs.Info("NewBucket", `name=`, name)
	b := Bucket(name)
	Buckets = append(Buckets, b)
	logs.Info("NewBucket-re", `Buckets=`, Buckets, `b=`, b)
	return b
}

func GetDB() *bolt.DB {
	return db
}



func (bucket Bucket) Set(key interface{}, value interface{}) error {
	logs.Info("Set", `table=`, bucket, `key=`, key, `value=`, value)
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		k := fmt.Sprint(key)
		if _, ok := value.([]byte); !ok {
			v := fmt.Sprint(value)
			if v == "" {
				if err := b.Delete([]byte(k)); err != nil {
					return err
				}
			} else {
				if err := b.Put([]byte(k), []byte(v)); err != nil {
					return err
				}
			}
		} else {
			if len(value.([]byte)) == 0 {
				if err := b.Delete([]byte(k)); err != nil {
					return err
				}

			} else {
				if err := b.Put([]byte(k), value.([]byte)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (bucket Bucket) Push2Array(key, value string) {
	logs.Info("Push2Array", `table=`, bucket, `key=`, key, `value=`, value)
	bucket.Set(key, strings.Join(append(strings.Split(bucket.GetString(key), ","), value), ","))
}

func (bucket Bucket) GetArray(key string) []string {
	logs.Info("GetArray", `table=`, bucket, `key=`, key)
	return strings.Split(bucket.GetString(key), ",")
}
func (bucket Bucket) Get(kv ...interface{}) string {
	logs.Info("Get", `table=`, bucket, `kv=`, kv)
	return bucket.GetString(kv)
}
func (bucket Bucket) GetString(kv ...interface{}) string {
	logs.Info("GetString", `table=`, bucket, `kv=`, kv)
	var key, value string
	for i := range kv {
		if i == 0 {
			key = fmt.Sprint(kv[0])
		} else {
			value = fmt.Sprint(kv[1])
		}
	}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if v := string(b.Get([]byte(key))); v != "" {
			value = v
		}
		return nil
	})
	return value
}

func (bucket Bucket) GetBytes(key string) []byte {
	logs.Info("GetBytes", `table=`, bucket, `key=`, key)
	var value []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if v := b.Get([]byte(key)); v != nil {
			value = v
		}
		return nil
	})
	return value
}

func (bucket Bucket) GetInt(key interface{}, vs ...int) int {
	logs.Info("GetInt", `table=`, bucket, `key=`, key, `vs=`, vs)
	var value int
	if len(vs) != 0 {
		value = vs[0]
	}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		v := Int(string(b.Get([]byte(fmt.Sprint(key)))))
		if v != 0 {
			value = v
		}
		return nil
	})
	return value
}

func (bucket Bucket) GetBool(key interface{}, vs ...bool) bool {
	logs.Info("GetBool", `table=`, bucket, `key=`, key, `vs=`, vs)
	var value bool
	if len(vs) != 0 {
		value = vs[0]
	}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		v := string(b.Get([]byte(fmt.Sprint(key))))
		if v == "true" {
			value = true
		} else if v == "false" {
			value = false
		}
		return nil
	})
	return value
}

func (bucket Bucket) Foreach(f func(k, v []byte) error) {
	logs.Info("Foreach", `table=`, bucket, `f=`, f)
	var bs = [][][]byte{}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b != nil {
			err := b.ForEach(func(k, v []byte) error {
				bs = append(bs, [][]byte{k, v})
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	for i := range bs {
		f(bs[i][0], bs[i][1])
	}
}

var Int = func(s interface{}) int {
	logs.Info("Int", `s=`, s)
	i, _ := strconv.Atoi(fmt.Sprint(s))
	return i
}

var Int64 = func(s interface{}) int64 {
	logs.Info("Int64", `s=`, s)
	i, _ := strconv.Atoi(fmt.Sprint(s))
	return int64(i)
}

func (bucket Bucket) Create(i interface{}) error {
	logs.Info("Create", `i=`, i)
	s := reflect.ValueOf(i).Elem()
	id := s.FieldByName("ID")
	sequence := s.FieldByName("Sequence")
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		if _, ok := id.Interface().(int); ok {
			key := id.Int()
			sq, err := b.NextSequence()
			if err != nil {
				return err
			}
			if key == 0 {
				key = int64(sq)
				id.SetInt(key)
			}
			if sequence != reflect.ValueOf(nil) {
				sequence.SetInt(int64(sq))
			}
			buf, err := json.Marshal(i)
			if err != nil {
				return err
			}
			return b.Put(itob(uint64(key)), buf)
		} else {
			key := id.String()
			sq, err := b.NextSequence()
			if err != nil {
				return err
			}
			if key == "" {
				key = fmt.Sprint(sq)
				id.SetString(key)
			}
			if sequence != reflect.ValueOf(nil) {
				sequence.SetInt(int64(sq))
			}
			buf, err := json.Marshal(i)
			if err != nil {
				return err
			}
			return b.Put([]byte(key), buf)
		}
	})
}

func itob(i uint64) []byte {
	logs.Info("12334", `i=`, i)
	return []byte(fmt.Sprint(i))
}

func (bucket Bucket) First(i interface{}) error {
	logs.Info("First", `i=`, i)
	var err error
	s := reflect.ValueOf(i).Elem()
	id := s.FieldByName("ID")
	if v, ok := id.Interface().(int); ok {
		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			if b == nil {
				err = errors.New("bucket not find")
				return nil
			}
			data := b.Get([]byte(fmt.Sprint(v)))
			if len(data) == 0 {
				err = errors.New("record not find")
				return nil
			}
			return json.Unmarshal(data, i)
		})
	} else {
		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			if b == nil {
				err = errors.New("bucket not find")
				return nil
			}
			data := b.Get([]byte(id.Interface().(string)))
			if len(data) == 0 {
				err = errors.New("record not find")
				return nil
			}
			return json.Unmarshal(data, i)
		})
	}
	return err
}