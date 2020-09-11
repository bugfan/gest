package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-xorm/xorm"
)

type modelAndController struct {
	C, M    interface{}
	Group   string
	NotCopy []string
}

var modelAndControllers []*modelAndController

func init() {
	modelAndControllers = make([]*modelAndController, 0)
}

func RegisterModelAndController(m, c interface{}, g string, notCopy []string) {
	if m == nil || c == nil {
		return
	}
	t := &modelAndController{
		C:       c,
		M:       m,
		Group:   g,
		NotCopy: notCopy,
	}
	modelAndControllers = append(modelAndControllers, t)
}

func NewAPIBackend(g *gin.Engine, x *xorm.Engine, relativePath string) error {
	group := g.Group(relativePath)
	// register routes
	for _, mc := range modelAndControllers {
		modelT := reflect.TypeOf(mc.M)
		if modelT.Kind() == reflect.Ptr {
			modelT = modelT.Elem()
		}
		contentT := reflect.TypeOf(mc.C)
		if contentT.Kind() == reflect.Ptr {
			contentT = contentT.Elem()
		}
		rest := NewRest(x, modelT, contentT, RouteTypeALL)
		rest.register(group.Group(mc.Group))
	}
	return nil
}

func exports(b interface{}) map[string]interface{} {
	fieldVal := make(map[string]interface{})
	ptrObjVal := reflect.ValueOf(b)
	objVal := ptrObjVal.Elem()
	objType := objVal.Type()
	fieldNum := objType.NumField()
	for i := 0; i < fieldNum; i++ {
		sf := objType.Field(i)
		valField := objVal.Field(i)
		if valField.CanInterface() {
			fieldVal[sf.Name] = valField.Interface()
		}
	}
	return fieldVal
}

func setExports(to, from interface{}) {
	exports := exports(from)
	for name, val := range exports {
		valVal := reflect.ValueOf(val)
		ptrObjVal := reflect.ValueOf(to)
		objVal := ptrObjVal.Elem()
		fieldVal := objVal.FieldByName(name)
		if fieldVal.IsValid() && fieldVal.Type() == valVal.Type() {
			fieldVal.Set(valVal)
		}
	}
}

type RouteType int

const (
	RouteTypeNew    = 1 << iota
	RouteTypeList   // get query all
	RouteTypeGet    // get query one
	RouteTypeUpdate // put
	RouteTypePatch  // patch
	RouteTypeDelete // delete
	RouteTypeALL    = RouteTypeNew | RouteTypeList | RouteTypeGet | RouteTypeUpdate | RouteTypeDelete
)

func NewRest(e *xorm.Engine, modelT, contentT reflect.Type, r RouteType) *Rest {
	return &Rest{
		modelType:   modelT,
		contentType: contentT,
		routes:      r,
		engine:      e,
	}
}

type Rest struct {
	model       interface{}
	modelType   reflect.Type
	content     interface{}
	contentType reflect.Type
	engine      *xorm.Engine
	routes      RouteType
	HiddenField []string
	NotCopy     []string
}

type handlerBefore interface {
	Before(*gin.Context, *xorm.Engine) bool
}

type handlerNew interface {
	New(*gin.Context)
}
type handlerGet interface {
	Get(*gin.Context)
}
type handlerList interface {
	List(*gin.Context)
}
type handlerUpdate interface {
	Update(*gin.Context)
}
type handlerPatch interface {
	Patch(*gin.Context)
}
type handlerDelete interface {
	Delete(*gin.Context)
}

func (b *Rest) register(g *gin.RouterGroup) {
	route := b.routes
	if (route & RouteTypeNew) != 0 {
		g.POST("", b.New)
	}
}

func (b *Rest) New(c *gin.Context) {
	obj := reflect.New(b.contentType).Interface()
	err := c.BindJSON(obj)
	data, err2 := json.Marshal(obj)
	fmt.Println("DATA:", err, err2, string(data))
	model := reflect.New(b.modelType).Interface()
	err = copyField(model, obj, nil)
	fmt.Println("DATA2:", err, model)
	id, err := b.engine.Insert(model)
	fmt.Println("DATA3:", err, id)

	c.JSON(http.StatusCreated, nil)
}

func copyField(to interface{}, from interface{}, excepts []string) error {
	toVal := reflect.ValueOf(to)
	if toVal.Kind() == reflect.Ptr {
		toVal = toVal.Elem()
	}
	fromVal := reflect.ValueOf(from)
	if fromVal.Kind() == reflect.Ptr {
		fromVal = fromVal.Elem()
	}
	// to fileld
	toType := toVal.Type()
	fieldNum := toType.NumField()
	for i := 0; i < fieldNum; i++ {
		toField := toType.Field(i)
		if excepts != nil && stringInSlice(toField.Name, excepts) {
			continue
		}
		toValField := toVal.Field(i)
		if !toValField.CanSet() {
			continue
		}
		if fromValField := fromVal.FieldByName(toField.Name); fromValField.IsValid() && fromValField.Type() == toValField.Type() {
			toValField.Set(fromValField)
			continue
		}
		if fromFunc := fromVal.Addr().MethodByName(toField.Name); fromFunc.IsValid() &&
			fromFunc.Type().NumOut() >= 1 &&
			fromFunc.Type().Out(0) == toValField.Type() &&
			fromFunc.Type().NumIn() == 0 {
			res := fromFunc.Call(make([]reflect.Value, 0))
			if len(res) > 1 {
				last := res[len(res)-1]
				if last.CanInterface() && !last.IsNil() {
					if err, ok := last.Interface().(error); ok {
						return err
					}
				}

			}
			toValField.Set(res[0])
			continue
		}
	}
	// to func

	toVal = toVal.Addr()
	toType = toVal.Type()
	funcNum := toType.NumMethod()
	for i := 0; i < funcNum; i++ {
		// method from type
		toMethod := toType.Method(i)
		if !strings.HasPrefix(toMethod.Name, "Set") {
			// only SetXXX methods
			continue
		}

		name := strings.TrimPrefix(toMethod.Name, "Set")
		// skip excepts
		if excepts != nil && stringInSlice(name, excepts) {
			continue
		}

		// func from value
		toFunc := toVal.MethodByName(toMethod.Name)
		argType := toFunc.Type().In(0)

		// from field
		if fromValField := fromVal.FieldByName(name); fromValField.IsValid() && fromValField.Type() == argType {
			res := toFunc.Call([]reflect.Value{fromValField})
			if len(res) > 0 {
				last := res[len(res)-1]
				if last.CanInterface() && !last.IsNil() {
					if err, ok := last.Interface().(error); ok {
						return err
					}
				}

			}
			continue
		}
		// from func

		if fromFunc := fromVal.Addr().MethodByName(name); fromFunc.IsValid() &&
			fromFunc.Type().NumOut() >= 1 &&
			fromFunc.Type().Out(0) == argType &&
			fromFunc.Type().NumIn() == 0 {
			res := fromFunc.Call(make([]reflect.Value, 0))
			if len(res) > 1 {
				last := res[len(res)-1]

				if last.CanInterface() && !last.IsNil() {
					if err, ok := last.Interface().(error); ok {
						return err
					}
				}

			}

			res = toFunc.Call([]reflect.Value{res[0]})
			if len(res) > 0 {
				last := res[len(res)-1]
				if last.CanInterface() && !last.IsNil() {
					if err, ok := last.Interface().(error); ok {
						return err
					}
				}

			}
			continue
		}

	}
	return nil
}
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
