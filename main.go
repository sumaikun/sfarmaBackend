package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/gorilla/handlers"
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
	router.Handle("/users", middleware.AuthMiddleware(http.HandlerFunc(createUsersEndPoint))).Methods("POST")
	router.Handle("/users", middleware.AuthMiddleware(http.HandlerFunc(allUsersEndPoint))).Methods("GET")
	router.Handle("/users/{id}", middleware.AuthMiddleware(http.HandlerFunc(findUserEndpoint))).Methods("GET")
	router.Handle("/users/{id}", middleware.AuthMiddleware(http.HandlerFunc(removeUserEndpoint))).Methods("DELETE")
	router.Handle("/users/{id}", middleware.AuthMiddleware(http.HandlerFunc(updateUserEndPoint))).Methods("PUT")

	/* Products Routes */
	router.Handle("/products", middleware.AuthMiddleware(http.HandlerFunc(createProductEndPoint))).Methods("POST")
	router.Handle("/products", middleware.AuthMiddleware(http.HandlerFunc(allProductsEndPoint))).Methods("GET")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(findProductEndpoint))).Methods("GET")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(removeProductEndpoint))).Methods("DELETE")
	router.Handle("/products/{id}", middleware.AuthMiddleware(http.HandlerFunc(updateProductEndPoint))).Methods("PUT")

	/* fileUpload */

	router.Handle("/fileUpload", middleware.AuthMiddleware(http.HandlerFunc(fileUpload))).Methods("POST")
	router.HandleFunc("/serveImage/{image}", serveImage).Methods("GET")

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With"})
	originsOk := handlers.AllowedOrigins([]string{os.Getenv("ORIGIN_ALLOWED")})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	//log.Fatal(http.ListenAndServe(":"+port, router))

	log.Fatal(http.ListenAndServe(":"+port, handlers.CORS(originsOk, headersOk, methodsOk)(router)))
}
