package rest

import (
	"reflect"

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
		s := NewRest(nil, mc.M, mc.C, RouteTypeALL)
		// e := exports(mc.C)
		// fmt.Println("Q:", e)
		// tt := reflect.TypeOf(mc.C)
		// t := reflect.New(tt)
		// id, err := x.Insert(t)
		// fmt.Println("EE:", err, id)

		// t := reflect.TypeOf(mc.C)
		// ot := t.Elem()
		// instValue := reflect.New(ot)
		// ifce := instValue.Interface()
		setExports(mc.M, mc.C)
		// ctl, _ := ifce.(Controller)
		// ctl.Register(r)

		// t := reflect.TypeOf(mc.C)
		// ot := t.Elem()
		// instValue := reflect.New(ot)
		// ifce := instValue.Interface()
		// setExports(ifce)

		s.register(group.Group(mc.Group), mc.C)

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
