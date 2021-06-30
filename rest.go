package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-xorm/xorm"
)

type modelAndController struct {
	Controller, Model interface{}
	Extra             []string
	NotCopy           []string
	HiddenFiled       []string
}

var modelAndControllers []*modelAndController

func init() {
	modelAndControllers = make([]*modelAndController, 0)
}

// model:model struct controller:conttoller struct hiddenField:not copy field g: subrouting,bind route path
func Register(model, controller interface{}, hiddenField []string, g ...string) {
	if model == nil || controller == nil {
		return
	}
	t := &modelAndController{
		Controller:  controller,
		Model:       model,
		Extra:       g,
		HiddenFiled: hiddenField,
	}
	modelAndControllers = append(modelAndControllers, t)
}

func NewAPIBackend(g *gin.RouterGroup, x *xorm.Engine, relativePath string) {
	group := g.Group(relativePath)
	// bind routes
	for _, mc := range modelAndControllers {
		modelT := reflect.TypeOf(mc.Model)
		if modelT.Kind() == reflect.Ptr {
			modelT = modelT.Elem()
		}
		controllerT := reflect.TypeOf(mc.Controller)
		if controllerT.Kind() == reflect.Ptr {
			controllerT = controllerT.Elem()
		}
		rest := NewRest(x, modelT, controllerT, RouteTypeALL, mc.HiddenFiled)
		subrouting := strings.ToLower(controllerT.Name())
		if len(mc.Extra) > 0 && strings.TrimSpace(mc.Extra[0]) != "" {
			subrouting = strings.TrimSpace(mc.Extra[0])
		}
		rest.bind(group.Group(subrouting), mc.Controller)
	}
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
	RouteTypeALL    = RouteTypeNew | RouteTypeList | RouteTypeGet | RouteTypeUpdate | RouteTypePatch | RouteTypeDelete
	// RouteTypeALL    = RouteTypeNew | RouteTypeList | RouteTypeGet | RouteTypeUpdate | RouteTypeDelete
)

func NewRest(e *xorm.Engine, modelT, controllerT reflect.Type, r RouteType, hiddenField []string) *Rest {
	return &Rest{
		modelType:      modelT,
		controllerType: controllerT,
		routes:         r,
		engine:         e,
		HiddenField:    hiddenField,
		NotCopy:        []string{"ID", "Created", "Updated"},
	}
}

type Rest struct {
	modelType      reflect.Type
	controllerType reflect.Type
	engine         *xorm.Engine
	routes         RouteType
	HiddenField    []string
	NotCopy        []string
}

type handlerBefore interface {
	Before(*gin.Context, *xorm.Engine) bool
}

type handlerAfter interface {
	After(*gin.Context, *xorm.Engine) bool
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

func (b *Rest) bind(g *gin.RouterGroup, h interface{}) {
	route := b.routes
	if (route & RouteTypeNew) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerNew)
		if ok {
			g.POST("", handler.New)
		} else {
			g.POST("", b.New)
		}
	}
	if (route & RouteTypeList) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerList)
		if ok {
			g.GET("", handler.List)
		} else {
			g.GET("", b.List)
		}
	}
	if (route & RouteTypeGet) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerGet)
		if ok {
			g.GET("/:id", handler.Get)
		} else {
			g.GET("/:id", b.Get)
		}
	}
	if (route & RouteTypeUpdate) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerUpdate)
		if ok {
			g.PUT("/:id", handler.Update)
		} else {
			g.PUT("/:id", b.Update)
		}
	}
	if (route & RouteTypePatch) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerPatch)
		if ok {
			g.PATCH("/:id", handler.Patch)
		} else {
			g.PATCH("/:id", b.Patch)
		}
	}
	if (route & RouteTypeDelete) != 0 {
		reflectVal := reflect.ValueOf(h)
		t := reflect.Indirect(reflectVal).Type()
		newObj := reflect.New(t)
		handler, ok := newObj.Interface().(handlerDelete)
		if ok {
			g.DELETE("/:id", handler.Delete)
		} else {
			g.DELETE("/:id", b.Delete)
		}
	}
}

func (b *Rest) New(c *gin.Context) {
	content := reflect.New(b.controllerType).Interface()
	err := c.BindJSON(content)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	model := reflect.New(b.modelType).Interface()
	err = copyField(model, content, b.NotCopy)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	_, err = b.engine.Insert(model)
	if err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	data := reflect.New(b.controllerType).Interface()
	err = copyField(data, model, b.HiddenField)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	c.JSON(http.StatusCreated, data)
}

func (b *Rest) List(c *gin.Context) {
	content := reflect.New(b.controllerType).Interface()
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(b.modelType)), 0, 0)
	slicePtr := reflect.New(slice.Type())
	sliceVal := slicePtr.Elem()
	err := b.engine.Find(slicePtr.Interface())
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
	}
	contentSlice := make([]interface{}, 0, sliceVal.Len())
	for i := 0; i < sliceVal.Len(); i++ {
		content := reflect.New(b.controllerType).Interface()
		err = copyField(content, sliceVal.Index(i).Interface(), b.HiddenField)
		if err != nil {
			c.AbortWithError(http.StatusUnprocessableEntity, err)
			return
		}

		contentSlice = append(contentSlice, content)
	}
	c.JSON(http.StatusOK, contentSlice)
}

func (b *Rest) Get(c *gin.Context) {
	content := reflect.New(b.controllerType).Interface()
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	inst := reflect.New(b.modelType).Interface()
	has, err := b.engine.ID(id).Get(inst)
	if !has {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	err = copyField(content, inst, b.HiddenField)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	c.JSON(200, content)
}

func (b *Rest) Update(c *gin.Context) {
	// get content
	content := reflect.New(b.controllerType).Interface()
	err := c.BindJSON(content)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	// get model
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	inst := reflect.New(b.modelType).Interface()
	has, err := b.engine.ID(id).Exist(inst)
	if !has {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	err = copyField(inst, content, b.NotCopy)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	_, err = b.engine.ID(id).AllCols().Update(inst)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	content = reflect.New(b.controllerType).Interface()
	err = copyField(content, inst, b.HiddenField)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	c.JSON(http.StatusOK, content)
}

func (b *Rest) Patch(c *gin.Context) {
	// get content
	content := reflect.New(b.controllerType).Interface()
	err := c.BindJSON(content)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	// get model
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	inst := reflect.New(b.modelType).Interface()
	has, err := b.engine.ID(id).Exist(inst)
	if !has {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	err = copyField(inst, content, b.NotCopy)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	_, err = b.engine.ID(id).Update(inst)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	content = reflect.New(b.controllerType).Interface()
	err = copyField(content, inst, b.HiddenField)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	c.JSON(http.StatusOK, content)
}

func (b *Rest) Delete(c *gin.Context) {
	content := reflect.New(b.controllerType).Interface()
	h, ok := content.(handlerBefore)
	if ok {
		if !h.Before(c, b.engine) {
			return
		}
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	inst := reflect.New(b.modelType).Interface()
	has, err := b.engine.ID(id).Exist(inst)
	if !has {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", c.Param("id")))
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	_, err = b.engine.ID(id).Delete(inst)
	if err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}
	c.Status(http.StatusNoContent)
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
