# go restful api generator

## Introduce
Only need to define the model, automatically generate add delete modify query interface

## Usage
`
// main func
func main() {
	g := gin.Default()
    x := xorm.NewEngine(...)
	rest.NewAPIBackend(g, x, "/api")
	log.Fatal(g.Run("127.0.0.1:9996"))
}

// api 
func init(){
    rest.RegisterModelAndController(&User{}, &UserController{}, nil, "user")
	rest.RegisterModelAndController(&Admin{}, &AdminController{}, []string{"Password"}, "admin")
}
type UserController struct{
    ID int64
    Name string
}
// set Before
func (u *UserController) Before(g *gin.Context, x *xorm.Engine) bool {
	if u.Name != "zhao"{
        return false
    }
	return true
}

type AdminController struct{
    ID int64
    Name string
    Password string
}
// overwrite Before
func (u *UserController)  New(c *gin.Context) {
	err := c.BindJSON(u)
	if err! = nil{
        c.AbortWithError(http.StatusBadRequest, err)
		return
    }
    if u.ID < 200 {
        c.AbortWithError(http.StatusBadRequest, errors.New("ID error"))
		return
    }
	c.JSON(200, nil)
}

type User struct{
    ID int64
    Name string
}
type Admin struct{
    ID int64
    Name string
    Password string
}
`