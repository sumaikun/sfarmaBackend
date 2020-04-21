package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/gorilla/mux"

	Config "github.com/sumaikun/sfarma-rest-api/config"
	middleware "github.com/sumaikun/sfarma-rest-api/middlewares"

	Dao "github.com/sumaikun/sfarma-rest-api/dao"
)

var (
	port   string
	jwtKey []byte
)

var dao = Dao.MongoConnector{}

//Dynamic types

var typeRegistry = make(map[string]reflect.Type)

func registerType(typedNil interface{}) {
	t := reflect.TypeOf(typedNil).Elem()
	typeRegistry[t.PkgPath()+"."+t.Name()] = t
}

func makeInstance(name string) interface{} {
	return reflect.New(typeRegistry[name]).Elem().Interface()
}

// CORSRouterDecorator applies CORS headers to a mux.Router
type CORSRouterDecorator struct {
	R *mux.Router
}

// ServeHTTP wraps the HTTP server enabling CORS headers.
// For more info about CORS, visit https://www.w3.org/TR/cors/
func (c *CORSRouterDecorator) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	//fmt.Println("I am on serve HTTP")

	rw.Header().Set("Access-Control-Allow-Origin", "*")

	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	rw.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Authorization, X-Requested-With")

	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		//fmt.Println("I am in options")
		rw.WriteHeader(http.StatusOK)
		return
	}

	c.R.ServeHTTP(rw, req)
}

//-------------------

func init() {

	var config = Config.Config{}
	config.Read()
	//fmt.Println(config.Jwtkey)
	jwtKey = []byte(config.Jwtkey)
	port = config.Port

	dao.Server = config.Server
	dao.Database = config.Database
	dao.Connect()
}

func main() {
	//initEvents()
	fmt.Println("start server in port " + port)
	router := mux.NewRouter().StrictSlash(true)

	/* Authentication */
	router.HandleFunc("/auth", authentication).Methods("POST")

	/* enums */
	router.Handle("/userRoles", middleware.AuthMiddleware(http.HandlerFunc(userRoles))).Methods("GET")

	/* prestashop Services */
	router.Handle("/getPrestaShopDistributors", middleware.AuthMiddleware(http.HandlerFunc(getPrestaShopDistributors))).Methods("GET")
	router.Handle("/getPrestaShopProductcategories", middleware.AuthMiddleware(http.HandlerFunc(getPrestaShopProductcategories))).Methods("GET")

	/* Users Routes */
	router.Handle("/users", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(createUsersEndPoint))))).Methods("POST")
	router.Handle("/users", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(allUsersEndPoint))))).Methods("GET")
	router.Handle("/users/{id}", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(findUserEndpoint))))).Methods("GET")
	router.Handle("/users/{id}", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(removeUserEndpoint))))).Methods("DELETE")
	router.Handle("/users/{id}", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(updateUserEndPoint))))).Methods("PUT")

	/* Products Routes */
	router.Handle("/products", middleware.AuthMiddleware(middleware.UserMiddleware(http.HandlerFunc(createProductEndPoint)))).Methods("POST")
	router.Handle("/products", middleware.AuthMiddleware(middleware.UserMiddleware(http.HandlerFunc(allProductsEndPoint)))).Methods("GET")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(findProductEndpoint))).Methods("GET")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(removeProductEndpoint))).Methods("DELETE")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(updateProductEndPoint))).Methods("PUT")

	/* fileUpload */

	router.Handle("/fileUpload", middleware.AuthMiddleware(http.HandlerFunc(fileUpload))).Methods("POST")
	router.HandleFunc("/serveImage/{image}", serveImage).Methods("GET")

	log.Fatal(http.ListenAndServe(":"+port, &CORSRouterDecorator{router}))

}
