package mysql

import (
	"database/sql/driver"
	"fmt"
	"testing"
)

type C struct {
	Contact *Contact `gorm:"column:contact"`
}

func (C) TableName() string {
	return "c"
}

//联系方式结构
type Contact struct {
	Phone string `json:"phone"` //电话号码
	Email string `json:"email"` //邮箱
}

func (j Contact) Value() (driver.Value, error) {
	return Value(j)
}

func (j *Contact) Scan(src interface{}) error {
	return Scan(j, src)
}

func TestContact(t *testing.T) {
	dbConn := New(Config{
		Url: "root:123@/test?charset=utf8&parseTime=True&loc=Local",
	})
	if err := dbConn.Create(&C{
		Contact: &Contact{
			Phone: "123",
			Email: "a@b.com",
		},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := dbConn.Create(&C{
		Contact: nil, //插入NULL
	}).Error; err != nil {
		t.Fatal(err)
	}
	var cs []C
	if err := dbConn.Find(&cs).Error; err != nil {
		t.Fatal(err)
	}
	fmt.Println(cs)
}
