package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	"github.com/thedevsaddam/govalidator"

	Config "github.com/sumaikun/sfarma-rest-api/config"
	middleware "github.com/sumaikun/sfarma-rest-api/middlewares"

	Dao "github.com/sumaikun/sfarma-rest-api/dao"

	Helpers "github.com/sumaikun/sfarma-rest-api/helpers"
)

var (
	port   string
	jwtKey []byte
)

var dao = Dao.MongoConnector{}

//xml as string

func returnXML(id string, reference string, manufacturer string, price string, category string, description string, name string, ean13 string, width string, height string, unity string, meta_title string, meta_description string, meta_keywords string, description_short string) string {
	var xml string

	if len(id) == 0 {
		xml = "<?xml version='1.0' encoding='UTF-8'?><prestashop xmlns:xlink='http://www.w3.org/1999/xlink'><product><type notFilterable='true'>simple</type><reference>" + reference + "</reference><id_manufacturer>" + manufacturer + "</id_manufacturer><price>" + price + "</price><active>1</active><state>1</state><id_type_redirected>0</id_type_redirected><available_for_order>1</available_for_order><ean13>" + ean13 + "</ean13><width>" + width + "</width><height>" + height + "</height><unity>" + unity + "</unity><id_category_default xlink:href='https://sfarmadroguerias.com/api/categories/" + category + "'>" + category + "</id_category_default><condition>new</condition><show_price>1</show_price><indexed>1</indexed><visibility>both</visibility><meta_title>" + meta_title + "</meta_title><meta_keywords>" + meta_keywords + "</meta_keywords><meta_description><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + meta_description + "</language></meta_description><meta_keywords><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>Farmacia, online, droguería, Bogotá, Colombia, Domicilio, complemento, suplemento, dieta, niños</language></meta_keywords><meta_title><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + name + "</language></meta_title><link_rewrite><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + name + "</language></link_rewrite><name><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + name + "</language></name><description><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + description + "</language></description><description_short>" + description_short + "</description_short><associations><categories nodeType='category' api='categories'><category xlink:href='https://sfarmadroguerias.com/api/categories/" + category + "'><id>" + category + "</id></category></categories></associations></product></prestashop>"
	} else {
		xml = "<?xml version='1.0' encoding='UTF-8'?><prestashop xmlns:xlink='http://www.w3.org/1999/xlink'><product><id>" + id + "</id><type notFilterable='true'>simple</type><reference>" + reference + "</reference><id_manufacturer>" + manufacturer + "</id_manufacturer><price>" + price + "</price><active>1</active><state>1</state><id_type_redirected>0</id_type_redirected><available_for_order>1</available_for_order><ean13>" + ean13 + "</ean13><width>" + width + "</width><height>" + height + "</height><unity>" + unity + "</unity><id_category_default xlink:href='https://sfarmadroguerias.com/api/categories/" + category + "'>" + category + "</id_category_default><condition>new</condition><show_price>1</show_price><indexed>1</indexed><visibility>both</visibility><meta_title>" + meta_title + "</meta_title><meta_keywords>" + meta_keywords + "</meta_keywords><meta_description><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + meta_description + "</language></meta_description><meta_keywords><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>Farmacia, online, droguería, Bogotá, Colombia, Domicilio, complemento, suplemento, dieta, niños</language></meta_keywords><meta_title>" + name + "</meta_title><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + name + "</language><name><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + name + "</language></name><description><language id='2' xlink:href='https://sfarmadroguerias.com/api/languages/2'>" + description + "</language></description><description_short>" + description_short + "</description_short><associations><categories nodeType='category' api='categories'><category xlink:href='https://sfarmadroguerias.com/api/categories/" + category + "'><id>" + category + "</id></category></categories></associations></product></prestashop>"
	}

	return xml
}

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

	govalidator.AddCustomRule("stateEnum", func(field string, rule string, message string, value interface{}) error {

		x := []string{"sended", "rejected", "inShop", "inShopToApprove", "inShopRejected", "", "inactive"}

		val := Helpers.Contains(x, value.(string))

		if val != true {
			return fmt.Errorf("The %s field must be a valid value for state Enum", field)
		}
		return nil
	})
}

func main() {

	//makeCommerssiaRequestTransaction("6723")

	//makeCommerssiaRequest("00020")

	//go proccessPrestaShopDistributors()

	//go proccessPrestashopSuppliers()

	//go proccessPrestashopProducts()

	//go proccessPrestaShopProductcategories()

	//checkPaymentsToSend()

	c := cron.New()
	//c.AddFunc("*/5 * * * *", func() {
	c.AddFunc("@hourly", func() {
		fmt.Println("Executed each hour")
		go proccessPrestaShopDistributors()

		go proccessPrestashopSuppliers()

		go proccessPrestashopProducts()

		go proccessPrestaShopProductcategories()
	})

	c.AddFunc("@every 5m", func() {
		fmt.Println("executing each 9 minutes")
		checkPaymentsToSend()
	})

	c.Start()

	//initEvents()
	fmt.Println("start server in port " + port)
	router := mux.NewRouter().StrictSlash(true)

	/* Authentication */
	router.HandleFunc("/auth", authentication).Methods("POST")
	router.HandleFunc("/updateConditions/{id}", updateConditions).Methods("GET")

	/* Sign Up */
	router.HandleFunc("/signUp", signUp).Methods("POST")

	/* enums */
	router.Handle("/userRoles", middleware.AuthMiddleware(http.HandlerFunc(userRoles))).Methods("GET")

	/* prestashop Services */
	router.HandleFunc("/getPrestaShopDistributors", getPrestaShopDistributors).Methods("GET")
	router.Handle("/getPrestaShopProductcategories", middleware.AuthMiddleware(http.HandlerFunc(getPrestaShopProductcategories))).Methods("GET")
	//router.HandleFunc("/testCreateProduct", testCreateProduct).Methods("GET")
	//router.HandleFunc("/testAddFile", testAddFile).Methods("GET")
	router.Handle("/createPrestaShopProduct", middleware.UserMiddleware(middleware.OnlyAdminMiddleware(middleware.AuthMiddleware(http.HandlerFunc(createPrestaShopProduct))))).Methods("POST")

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
	router.Handle("/products/{id}", middleware.AuthMiddleware(middleware.UserMiddleware(http.HandlerFunc(updateProductEndPoint)))).Methods("PUT")

	router.HandleFunc("/testingXML/{idProduct}", testingXML).Methods("GET")
	/* Transfer Routes  */

	router.Handle("/transfers", middleware.AuthMiddleware(middleware.UserMiddleware(middleware.OnlyAdminMiddleware(http.HandlerFunc(allTransfersEndPoint))))).Methods("GET")

	/* fileUpload */

	router.Handle("/fileUpload", middleware.AuthMiddleware(http.HandlerFunc(fileUpload))).Methods("POST")
	router.HandleFunc("/serveImage/{image}", serveImage).Methods("GET")

	/* excel format */

	router.Handle("/downloadFormat", middleware.AuthMiddleware(http.HandlerFunc(downloadFormat))).Methods("GET")
	router.Handle("/massiveUpload", middleware.AuthMiddleware(middleware.UserMiddleware(http.HandlerFunc(massiveUpload)))).Methods("POST")

	/**  commerssias connection services  */
	router.HandleFunc("/checkProductQuantityCommerssia", checkProductQuantityCommerssia).Methods("POST")

	http.DefaultClient.Timeout = time.Minute * 5

	log.Fatal(http.ListenAndServe(":"+port, &CORSRouterDecorator{router}))

}
