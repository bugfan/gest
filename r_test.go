package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	_ "github.com/go-sql-driver/mysql"

	"github.com/bugfan/rest"
	"github.com/gin-gonic/gin"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

type Config struct {
	Object  string
	ConnStr string
	Log     string
}

func newEngine(config *Config) (*xorm.Engine, error) {
	x, err := xorm.NewEngine(config.Object, config.ConnStr)
	if err != nil {
		return nil, err
	}
	x.SetMapper(core.GonicMapper{})

	if config.Log != "" {
		f, err := os.Create(config.Log)
		if err != nil {
			return nil, fmt.Errorf("Fail to create xorm.log: %v", err)
		}
		x.SetLogger(xorm.NewSimpleLogger(f))
	}
	x.ShowSQL(true)
	return x, nil
}

func SetEngine(config *Config) (*xorm.Engine, error) {
	var err error
	if x, err = newEngine(config); err != nil {
		return nil, err
	}

	if err = x.StoreEngine("InnoDB").Sync2(&User2{}); err != nil {
		return nil, fmt.Errorf("sync database struct error: %v\n", err)
	}
	return x, nil
}

var x *xorm.Engine

func main() {

	g := gin.Default()
	_, err := SetEngine(&Config{"mysql", "root:123456@tcp(127.0.0.1:3306)/ums?charset=utf8&parseTime=true", "./xorm.log"})
	fmt.Println("init orm:", err)
	rest.NewAPIBackend(g, x, "/api")
	fmt.Println("ok")
	log.Fatal(g.Run("127.0.0.1:9996"))
}

func MyCopy(to, from interface{}, excepts []string) {
	fromValue := reflect.ValueOf(from)
	if fromValue.Kind() == reflect.Ptr {
		fromValue = fromValue.Elem()
	}
	toValue := reflect.ValueOf(to)
	if toValue.Kind() == reflect.Ptr {
		toValue = toValue.Elem()
	}
	toType := toValue.Type()
	toTypeFieldNum := toType.NumField()
	// set field
	for i := 0; i < toTypeFieldNum; i++ {
		toFiledName := toType.Field(i).Name
		if excepts != nil && stringInSlice(toFiledName, excepts) {
			continue
		}
		toValueField := toValue.Field(i)
		if !toValueField.CanSet() {
			continue
		}
		if fromValueField := fromValue.FieldByName(toFiledName); fromValueField.IsValid() && fromValueField.Type() == toValueField.Type() {
			toValueField.Set(fromValueField)
			continue
		}
		fmt.Println("t:", toType.Field(i))
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func init() {
	rest.RegisterModelAndController(&User2{}, &userController{}, "/user", nil)
}

type userController struct {
	ID   int64
	Name string
}

func (u *userController) Before(g *gin.Context, x *xorm.Engine) bool {
	fmt.Println("CC:", u.ID, u.Name)
	return true
}

func (*userController) New2(c *gin.Context) {
	os.Exit(2)
	c.JSON(311, 78)
}

type User2 struct {
	ID   int64
	Name string
}
