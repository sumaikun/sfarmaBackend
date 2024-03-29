package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"

	Models "github.com/sumaikun/sfarma-rest-api/models"

	Helpers "github.com/sumaikun/sfarma-rest-api/helpers"

	"github.com/clbanning/mxj"
)

var (
	lastTracked float64
)

func init() {
	fmt.Println("init executed")
	lastTracked = 0
}

//-----------------------------  Auth functions --------------------------------------------------

func authentication(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	response := &Models.TokenResponse{Token: "", User: nil}

	var creds Models.Credentials
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&creds)

	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the expected password from our in memory map
	expectedPassword, ok := Models.Users[creds.Username]

	//fmt.Println("expectedpassword " + expectedPassword)

	//fmt.Println(creds.Password)

	//fmt.Println(Helpers.HashPassword(creds.Password))

	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if !ok || !Helpers.CheckPasswordHash(creds.Password, expectedPassword) {

		user, err := dao.FindOneByKEY("users", "email", creds.Username)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {

			match := Helpers.CheckPasswordHash(creds.Password, user.(bson.M)["password"].(string))

			if !match {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			log.Println("user found login", user)

			response.User = user.(bson.M)

		}

	}

	//log.Println("responseUser", response.User)

	if response.User["state"] == "pending" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Declare the expiration time of the token
	// here, we have kept it as 5 minutes
	expirationTime := time.Now().Add(8 * time.Hour)
	// Create the JWT claims, which includes the username and expiry time
	claims := &Models.Claims{
		Username: creds.Username,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	w.Header().Set("Content-type", "application/json")

	//Generate json response for get the token
	response.Token = tokenString

	json.NewEncoder(w).Encode(response)
}

func authUserCheck(w http.ResponseWriter, r *http.Request) {

	user := context.Get(r, "user")

	userParsed := user.(bson.M)

	log.Println("user after context", userParsed)

}

func updateConditions(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)

	user, err := dao.FindByID("users", params["id"])

	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}

	parsedData := user.(bson.M)

	parsedData["conditions"] = true

	if err := dao.Update("users", parsedData["_id"].(bson.ObjectId), parsedData); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func signUp(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	w.Header().Set("Content-type", "application/json")

	err, user := signUpValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	user.ID = bson.NewObjectId()
	user.Role = "distributors"
	user.State = "pending"
	user.Date = time.Now().String()
	user.UpdateDate = time.Now().String()

	if len(user.Password) != 0 {
		user.Password, _ = Helpers.HashPassword(user.Password)
	}

	if err := dao.Insert("users", user, []string{"email"}); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusCreated, user)

}

// ----------Prestashop Integration ----------------------------------------

/*
func testCreateProduct(w http.ResponseWriter, r *http.Request) {

	xml := returnXML()

	var xmlStr = []byte(xml)

	req, err := http.NewRequest("POST", "https://sfarmadroguerias.com/api/products?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV", bytes.NewBuffer(xmlStr))
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Output-Format", "JSON")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)


	product, _ := result["product"].(map[string]interface{})

	log.Println("product id", product["id"])

}*/

func testAddFile(filename string, productID string) {

	// Open the file
	file, err := os.Open("./files/" + filename)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	// Close the file later
	defer file.Close()

	log.Println(file)

	// Buffer to store our request body as bytes
	var requestBody bytes.Buffer

	// Create a multipart writer
	multiPartWriter := multipart.NewWriter(&requestBody)

	// Initialize the file field
	fileWriter, err := multiPartWriter.CreateFormFile("image", filename)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Copy the actual file content to the field field's writer
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	multiPartWriter.Close()

	req, err := http.NewRequest("POST", "https://sfarmadroguerias.com/api/images/products/"+productID+"?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV", &requestBody)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	log.Println(result)
}

func createPrestaShopProduct(w http.ResponseWriter, r *http.Request) {

	//fmt.Println("prestashop product")

	var createProduct Models.CreateProduct
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&createProduct)

	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//log.Println(createProduct)

	product, err := dao.FindByID("products", createProduct.Product)
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	//log.Println(product)

	parsedProduct := product.(bson.M)

	/*if parsedProduct["state"] == "sended" {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Product Already Sended")
		return
	}*/

	if parsedProduct["state"] != "inShop" && parsedProduct["picture"] == "" {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Product Need Image")
		return
	}

	var parsedProductM Models.Product

	bsonBytes, _ := bson.Marshal(product)
	bson.Unmarshal(bsonBytes, &parsedProductM)

	//xml := returnXML(parsedProduct["prestashopId"].(string), createProduct.Reference, parsedProduct["laboratory"].(string), createProduct.Price, parsedProduct["category"].(string), parsedProduct["description"].(string), parsedProduct["name"].(string), parsedProduct["ean13"].(string), parsedProduct["width"].(string), parsedProduct["height"].(string), parsedProduct["unity"].(string), parsedProduct["metaTitle"].(string), parsedProduct["metaDescription"].(string), parsedProduct["metaKeywords"].(string), parsedProduct["descriptionShort"].(string))

	xml := returnXML(parsedProductM.PrestashopID, createProduct.Reference, parsedProductM.Laboratory, createProduct.Price, parsedProductM.Category, parsedProductM.Description, parsedProductM.Name, parsedProductM.ExternalBoxDesc, parsedProductM.Width, parsedProductM.Height, parsedProductM.Unity, parsedProductM.MetaTitle, parsedProductM.MetaDescription, parsedProductM.MetaKeywords, parsedProductM.DescriptionShort)

	log.Println("xml", xml)

	var xmlStr = []byte(xml)

	req, err := http.NewRequest("POST", "https://sfarmadroguerias.com/api/products?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV", bytes.NewBuffer(xmlStr))
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "error generating request")
		return
	}

	if len(parsedProduct["prestashopId"].(string)) > 0 {
		//fmt.Println("update product url:", "https://sfarmadroguerias.com/api/products/"+parsedProduct["prestashopId"].(string)+"?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")
		fmt.Println("update product xml:", xml)

		req, err = http.NewRequest("PUT", "https://sfarmadroguerias.com/api/products/"+parsedProduct["prestashopId"].(string)+"?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV", bytes.NewBuffer(xmlStr))
		if err != nil {
			Helpers.RespondWithError(w, http.StatusBadRequest, "error generating request")
			return
		}
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Output-Format", "JSON")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "error making request")
		return
	}

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	fmt.Println("result", result)

	productG, _ := result["product"].(map[string]interface{})

	log.Println("PRESTASHOP product id generated", productG["id"])

	user := context.Get(r, "user")

	userParsed := user.(bson.M)

	var transfer Models.Transfer
	transfer.ID = bson.NewObjectId()
	transfer.Product = bson.ObjectIdHex(createProduct.Product)
	transfer.User = userParsed["_id"].(bson.ObjectId)
	transfer.Reference = createProduct.Reference
	transfer.Price = createProduct.Price
	transfer.ProductID = productG["id"].(string)
	transfer.Date = time.Now().String()
	transfer.UpdateDate = time.Now().String()

	if err := dao.Insert("transfers", transfer, nil); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go testAddFile(parsedProduct["picture"].(string), productG["id"].(string))

	if len(parsedProduct["picture2"].(string)) != 0 {
		go testAddFile(parsedProduct["picture2"].(string), productG["id"].(string))
	}

	if len(parsedProduct["picture3"].(string)) != 0 {
		go testAddFile(parsedProduct["picture3"].(string), productG["id"].(string))
	}

	parsedProduct["state"] = "sended"

	parsedProduct["prestashopId"] = productG["id"].(string)

	parsedProduct["recommendedPrice"] = string(createProduct.Price)

	parsedProduct["shopDefaultReference"] = string(createProduct.Reference)

	if err := dao.Update("products", parsedProduct["_id"], parsedProduct); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

}

func getAllRequest(url string) (map[string][]interface{}, error) {
	// By now our original request body should have been populated, so let's just use it with our custom request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	//fmt.Println("req", req)

	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Output-Format", "JSON")
	req.Header.Set("Connection", "keep-alive")

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	//fmt.Println("body", response)

	var result map[string][]interface{}

	if response != nil {
		json.NewDecoder(response.Body).Decode(&result)
	}

	return result, nil

}

func getRequest(url string) (map[string]interface{}, error) {
	// By now our original request body should have been populated, so let's just use it with our custom request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Output-Format", "JSON")

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	//fmt.Println(response.Body)

	var result map[string]interface{}

	if response != nil {
		json.NewDecoder(response.Body).Decode(&result)
	}

	return result, nil

}

func proccessPrestaShopDistributors() {

	fmt.Println("execute distributors")

	result, err := getAllRequest("https://sfarmadroguerias.com/api/manufacturers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	if err != nil {

		return
	}

	var slice []interface{}

	for _, element := range result["manufacturers"] {

		md, _ := element.(map[string]interface{})

		subelement, err := constructDistributors(fmt.Sprintf("%g", md["id"]))

		if err != nil {

			return
		}

		if subelement["active"] == "1" {
			slice = append(slice, subelement)
		}

	}

	//fmt.Println("Slice Result ", slice)

	for _, element := range slice {
		//fmt.Println("Key:", key, "=>", "Element:", element)

		var laboratory Models.Laboratories

		parsedElm := element.(map[string]interface{})

		if parsedElm["active"] == "1" {

			//fmt.Println(parsedElm["id"], int(parsedElm["id"].(float64)))

			prestaShopID := int(parsedElm["id"].(float64))

			//fmt.Println(strconv.Itoa(prestaShopID), string(prestaShopID))

			laboratory.PrestashopID = strconv.Itoa(prestaShopID)

			laboratory.Name = parsedElm["name"].(string)

			laboratory.Date = time.Now().String()

			exists, err := dao.FindManyByKEY("laboratories", "prestashopId", strconv.Itoa(prestaShopID))
			if err != nil {
				return
			}

			//fmt.Println("len", len(exists.([]interface{})))

			if len(exists.([]interface{})) == 0 {
				//fmt.Println("laboratory not exist")
				laboratory.ID = bson.NewObjectId()
				if err := dao.Insert("laboratories", laboratory, nil); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				//fmt.Println("laboratory exist")
				parsedExist := exists.([]interface{})[0].(bson.M)
				laboratory.ID = parsedExist["_id"].(bson.ObjectId)
				if err := dao.Update("laboratories", laboratory.ID, laboratory); err != nil {
					fmt.Println(err)
					return
				}
			}

			//fmt.Println(laboratory)

		}

	}

}

func getPrestaShopDistributors(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	w.Header().Set("Content-type", "application/json")

	laboratories, err := dao.FindAll("laboratories")
	if err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, laboratories)

}

func constructDistributors(id string) (map[string]interface{}, error) {

	result, err := getRequest("https://sfarmadroguerias.com/api/manufacturers/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	if err != nil {

		return nil, err
	}
	//log.Println(result)

	if result != nil {
		manufacturer, _ := result["manufacturer"].(map[string]interface{})

		//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

		return manufacturer, nil

	}

	return nil, nil

}

func proccessPrestaShopProductcategories() {

	result, err := getAllRequest("https://sfarmadroguerias.com/api/categories?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&output_format=JSON&display=[id,name,active]")

	if err != nil {

		return
	}

	//fmt.Println("Key:", key, "=>", "Element:", element)
	var category Models.Categories

	for _, element := range result["categories"] {
		md, _ := element.(map[string]interface{})
		//fmt.Println("Key:", key, "=>", "Element:", md)

		if md["active"] == "1" {

			//fmt.Println(parsedElm["id"], int(parsedElm["id"].(float64)))

			prestaShopID := int(md["id"].(float64))

			//fmt.Println(strconv.Itoa(prestaShopID), string(prestaShopID))

			category.PrestashopID = strconv.Itoa(prestaShopID)

			category.Name = md["name"].(string)

			category.Date = time.Now().String()

			exists, err := dao.FindManyByKEY("categories", "prestashopId", strconv.Itoa(prestaShopID))
			if err != nil {
				return
			}

			//fmt.Println("len", len(exists.([]interface{})))

			if len(exists.([]interface{})) == 0 {
				//fmt.Println("supplier not exist")
				category.ID = bson.NewObjectId()
				if err := dao.Insert("categories", category, nil); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				//fmt.Println("supplier exist")
				parsedExist := exists.([]interface{})[0].(bson.M)
				category.ID = parsedExist["_id"].(bson.ObjectId)
				if err := dao.Update("categories", category.ID, category); err != nil {
					fmt.Println(err)
					return
				}
			}

			//fmt.Println(laboratory)

		}

	}

}

func getPrestaShopProductcategories(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	w.Header().Set("Content-type", "application/json")

	categories, err := dao.FindAll("categories")
	if err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, categories)

}

func constructCategory(id string) map[string]interface{} {

	//fmt.Println("https://sfarmadroguerias.com/api/categories/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	result, err := getRequest("https://sfarmadroguerias.com/api/categories/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	if err != nil {

		return nil
	}

	//log.Println(result)

	category, _ := result["category"].(map[string]interface{})

	//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

	return category
}

// Get products from prestashop

func proccessPrestashopProducts() {

	fmt.Println("proccess prestashop products")

	result, err := getAllRequest("https://sfarmadroguerias.com/api/products?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[id,name,reference,price,id_manufacturer,description,id_category_default,active,id_supplier,id_default_image]&output_format=JSON")
	//,width,height,ean13,meta_title,meta_description,meta_keywords,id_default_image,unity,description_short
	if err != nil {

		fmt.Println("getting products error", err)

		return
	}

	//fmt.Println("result", result)

	for _, element := range result["products"] {
		md, _ := element.(map[string]interface{})
		//fmt.Println("Key:", key, "=>", "Element:", md["name"])
		//if md["active"] == "1" {

		if md["id_manufacturer"] == "0" && md["id_supplier"] != "0" {
			//fmt.Println("supplier", md)
			supplier, _ := dao.FindOneByKEY("suppliers", "prestashopId", md["id_supplier"].(string))
			//fmt.Println("supplier", supplier)
			if supplier != nil {
				parsedSupplier := supplier.(bson.M)
				manufacturer, _ := dao.FindOneLikeKEY("laboratories", "name", parsedSupplier["name"].(string))
				if manufacturer != nil {
					parsedManufacturer := manufacturer.(bson.M)
					var localProduct Models.Product
					localProduct.Name = md["name"].(string)
					localProduct.Category = md["id_category_default"].(string)
					localProduct.Description = md["description"].(string)
					localProduct.Laboratory = parsedManufacturer["prestashopId"].(string)
					localProduct.RecommendedPrice = md["price"].(string)
					localProduct.ShopDefaultReference = md["reference"].(string)

					if md["active"] == "1" {
						localProduct.State = "inShop"
					} else {
						localProduct.State = "inShopRejected"
					}
					prestaShopID := int(md["id"].(float64))
					localProduct.PrestashopID = strconv.Itoa(prestaShopID)
					localProduct.Date = time.Now().String()
					localProduct.DefaultImageID = md["id_default_image"].(string)
					//xml := returnXML(createProduct.Reference, parsedProduct["laboratory"].(string), createProduct.Price, parsedProduct["category"].(string), parsedProduct["description"].(string), parsedProduct["name"].(string))

					exists, err := dao.FindManyByKEY("products", "prestashopId", strconv.Itoa(prestaShopID))
					if err != nil {
						return
					}

					//fmt.Println("len", len(exists.([]interface{})))

					if len(exists.([]interface{})) == 0 {
						//fmt.Println("product not exist")
						localProduct.ID = bson.NewObjectId()
						if err := dao.Insert("products", localProduct, nil); err != nil {
							fmt.Println(err)
							//return
						}
					} else {
						//fmt.Println("product exist")
						parsedExist := exists.([]interface{})[0].(bson.M)

						parsedExist["name"] = localProduct.Name

						parsedExist["description"] = localProduct.Description

						parsedExist["recommendedPrice"] = localProduct.RecommendedPrice

						parsedExist["reference"] = localProduct.ShopDefaultReference

						localProduct.ID = parsedExist["_id"].(bson.ObjectId)
						if err := dao.Update("products", localProduct.ID, parsedExist); err != nil {
							fmt.Println(err)
							//return
						}
					}

					//fmt.Println("localProduct", localProduct)
				}
			}

		}
		//}

	}
	fmt.Println("finish execute get products")
	proccessPrestashopProducts2()
}

func proccessPrestashopProducts2() {

	fmt.Println("proccess prestashop products 2")

	result, err := getAllRequest("https://sfarmadroguerias.com/api/products?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[id,width,height,ean13,meta_title,meta_description,meta_keywords,unity,description_short]&output_format=JSON")
	//
	if err != nil {

		fmt.Println("getting products error", err)

		return
	}

	//fmt.Println("result", result)

	for _, element := range result["products"] {
		md, _ := element.(map[string]interface{})
		//fmt.Println("Key:", key, "=>", "Element:", md["unity"])

		prestaShopID := int(md["id"].(float64))

		exists, err := dao.FindManyByKEY("products", "prestashopId", strconv.Itoa(prestaShopID))
		if err != nil {
			return
		}

		if len(exists.([]interface{})) > 0 {

			parsedExist := exists.([]interface{})[0].(bson.M)

			parsedExist["width"] = md["width"].(string)
			parsedExist["height"] = md["height"].(string)
			parsedExist["externalBoxDesc"] = md["ean13"].(string)
			parsedExist["metaTitle"] = md["meta_title"].(string)
			parsedExist["metaDescription"] = md["meta_description"].(string)
			parsedExist["metaKeywords"] = md["meta_keywords"].(string)
			parsedExist["unity"] = md["unity"].(string)
			parsedExist["descriptionShort"] = md["description_short"].(string)

			ID := parsedExist["_id"].(bson.ObjectId)
			if err := dao.Update("products", ID, parsedExist); err != nil {
				fmt.Println(err)
				//return
			}
		}

	}

}

// Get suppliers from prestashop

func proccessPrestashopSuppliers() {

	result, err := getAllRequest("https://sfarmadroguerias.com/api/suppliers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&output_format=JSON&display=[id,name,active]")

	if err != nil {

		return
	}

	var supplier Models.Suppliers

	for _, element := range result["suppliers"] {
		md, _ := element.(map[string]interface{})
		//fmt.Println("Key:", key, "=>", "Element:", md)

		if md["active"] == "1" {

			//fmt.Println(parsedElm["id"], int(parsedElm["id"].(float64)))

			prestaShopID := int(md["id"].(float64))

			//fmt.Println(strconv.Itoa(prestaShopID), string(prestaShopID))

			supplier.PrestashopID = strconv.Itoa(prestaShopID)

			supplier.Name = md["name"].(string)

			supplier.Date = time.Now().String()

			exists, err := dao.FindManyByKEY("suppliers", "prestashopId", strconv.Itoa(prestaShopID))
			if err != nil {
				return
			}

			//fmt.Println("len", len(exists.([]interface{})))

			if len(exists.([]interface{})) == 0 {
				//fmt.Println("supplier not exist")
				supplier.ID = bson.NewObjectId()
				if err := dao.Insert("suppliers", supplier, nil); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				//fmt.Println("supplier exist")
				parsedExist := exists.([]interface{})[0].(bson.M)
				supplier.ID = parsedExist["_id"].(bson.ObjectId)
				if err := dao.Update("suppliers", supplier.ID, supplier); err != nil {
					fmt.Println(err)
					return
				}
			}

			//fmt.Println(laboratory)

		}

	}

}

// Enums --------------------------------------------------------------------

func userRoles(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-type", "application/json")

	x := [2]string{"admin", "distributors"}

	Helpers.RespondWithJSON(w, http.StatusOK, x)
}

//-----------------------------  Users functions --------------------------------------------------

func allUsersEndPoint(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-type", "application/json")

	users, err := dao.FindAll("users")
	if err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, users)
}

func createUsersEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	w.Header().Set("Content-type", "application/json")

	err, user := userValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	user.ID = bson.NewObjectId()
	user.Date = time.Now().String()
	user.UpdateDate = time.Now().String()

	if len(user.Password) != 0 {
		user.Password, _ = Helpers.HashPassword(user.Password)
	}

	if err := dao.Insert("users", user, []string{"email"}); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusCreated, user)

}

func findUserEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	user, err := dao.FindByID("users", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, user)

}

func removeUserEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	err := dao.DeleteByID("users", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, nil)

}

func updateUserEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	params := mux.Vars(r)

	w.Header().Set("Content-type", "application/json")

	err, user := userValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	prevUser, err2 := dao.FindByID("users", params["id"])
	if err2 != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}

	parsedData := prevUser.(bson.M)

	user.ID = parsedData["_id"].(bson.ObjectId)

	user.Date = parsedData["date"].(string)

	user.UpdateDate = time.Now().String()

	if len(user.Password) == 0 {
		user.Password = parsedData["password"].(string)
	} else {
		user.Password, _ = Helpers.HashPassword(user.Password)
	}

	if err := dao.Update("users", user.ID, user); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

}

//-------------------------------------- Products Functions ----------------------------------

func allProductsEndPoint(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-type", "application/json")

	user := context.Get(r, "user")

	userParsed := user.(bson.M)

	if userParsed["role"] == "admin" {
		products, err := dao.FindAll("products")
		if err != nil {
			Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		Helpers.RespondWithJSON(w, http.StatusOK, products)

	} else {
		products, err := dao.FindManyByKEY("products", "laboratory", strconv.Itoa(userParsed["laboratory"].(int)))
		if err != nil {
			Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		Helpers.RespondWithJSON(w, http.StatusOK, products)
	}

}

func createProductEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	w.Header().Set("Content-type", "application/json")

	err, product := productValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	log.Println(product)

	product.ID = bson.NewObjectId()
	product.Date = time.Now().String()
	product.UpdateDate = time.Now().String()

	user := context.Get(r, "user")

	userParsed := user.(bson.M)

	product.User = userParsed["_id"].(bson.ObjectId)

	if userParsed["role"] != "admin" {

		product.Laboratory = strconv.Itoa(userParsed["laboratory"].(int))
		if err := dao.Insert("products", product, []string{"name"}); err != nil {
			Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		Helpers.RespondWithJSON(w, http.StatusCreated, product)
	} else {

		if err := dao.Insert("products", product, []string{"name"}); err != nil {
			Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		Helpers.RespondWithJSON(w, http.StatusCreated, product)

	}

}

func findProductEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	product, err := dao.FindByID("products", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, product)

}

func removeProductEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	err := dao.DeleteByID("products", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, nil)

}

func updateProductEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	params := mux.Vars(r)

	w.Header().Set("Content-type", "application/json")

	err, product := productValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	prevData, err2 := dao.FindByID("products", params["id"])
	if err2 != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	parsedData := prevData.(bson.M)

	user := context.Get(r, "user")

	userParsed := user.(bson.M)

	if parsedData["user"] == nil {

		product.User = userParsed["_id"].(bson.ObjectId)
	} else {
		product.User = parsedData["user"].(bson.ObjectId)
	}

	product.ID = parsedData["_id"].(bson.ObjectId)

	if len(product.State) == 0 {
		product.State = parsedData["state"].(string)
	}

	product.Date = parsedData["date"].(string)

	product.UpdateDate = time.Now().String()

	if err := dao.Update("products", product.ID, product); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

}

//-------------------------------------- Transfers Functions ----------------------------------

func allTransfersEndPoint(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-type", "application/json")

	products, err := dao.FindAll("transfers")
	if err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, products)
}

func createTransferEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	w.Header().Set("Content-type", "application/json")

	err, transfer := transferValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	transfer.ID = bson.NewObjectId()
	transfer.Date = time.Now().String()
	transfer.UpdateDate = time.Now().String()

	if err := dao.Insert("transfers", transfer, nil); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusCreated, transfer)

}

func findTransferEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	transfer, err := dao.FindByID("transfers", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Transfer ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, transfer)

}

func removeTransferEndpoint(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	err := dao.DeleteByID("transfers", params["id"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Transfer ID")
		return
	}
	Helpers.RespondWithJSON(w, http.StatusOK, nil)

}

func updateTransferEndPoint(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	params := mux.Vars(r)

	w.Header().Set("Content-type", "application/json")

	err, transfer := productValidator(r)

	if len(err["validationError"].(url.Values)) > 0 {
		//fmt.Println(len(e))
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	prevData, err2 := dao.FindByID("transfers", params["id"])
	if err2 != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Transfer ID")
		return
	}

	parsedData := prevData.(bson.M)

	transfer.ID = parsedData["_id"].(bson.ObjectId)

	transfer.Date = parsedData["date"].(string)

	transfer.UpdateDate = time.Now().String()

	if err := dao.Update("transfers", transfer.ID, transfer); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

}

//------------------------------------- Testing xml -----------------------------------------

func testingXML(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	params := mux.Vars(r)
	product, err := dao.FindByID("products", params["idProduct"])
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	var parsedProduct Models.Product

	bsonBytes, _ := bson.Marshal(product)
	bson.Unmarshal(bsonBytes, &parsedProduct)

	xml := returnXML(parsedProduct.PrestashopID, parsedProduct.ShopDefaultReference, parsedProduct.Laboratory, parsedProduct.RecommendedPrice, parsedProduct.Category, parsedProduct.Description, parsedProduct.Name, parsedProduct.ExternalBoxDesc, parsedProduct.Width, parsedProduct.Height, parsedProduct.Unity, parsedProduct.MetaTitle, parsedProduct.MetaDescription, parsedProduct.MetaKeywords, parsedProduct.DescriptionShort)

	w.Write([]byte(xml))
}

//-------------------------------------- file Upload -----------------------------------------

func fileUpload(w http.ResponseWriter, r *http.Request) {

	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		Helpers.RespondWithJSON(w, http.StatusBadRequest, err)
		return
	}

	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	var extension = filepath.Ext(handler.Filename)

	fmt.Printf("Extension: %+v\n", extension)

	tempFile, err := ioutil.TempFile("files", "upload-*"+extension)

	if err != nil {
		fmt.Println(err)
		Helpers.RespondWithJSON(w, http.StatusInternalServerError, err)
	}

	var tempName = strings.Trim(tempFile.Name(), "files/")

	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		Helpers.RespondWithJSON(w, http.StatusInternalServerError, err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"filename": tempName})

}

func serveImage(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)

	var fileName = params["image"]

	if !strings.Contains(fileName, "png") && !strings.Contains(fileName, "jpg") && !strings.Contains(fileName, "jpeg") && !strings.Contains(fileName, "gif") {
		Helpers.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"result": "invalid file extension"})
		return
	}

	img, err := os.Open("./files/" + params["image"])
	if err != nil {
		//log.Fatal(err) // perhaps handle this nicer
		Helpers.RespondWithJSON(w, http.StatusInternalServerError, err)
		return
	}
	defer img.Close()
	w.Header().Set("Content-Type", "image/jpeg") // <-- set the content-type header
	io.Copy(w, img)

}

func downloadFormat(w http.ResponseWriter, r *http.Request) {

	format, err := os.Open("./format/importFormat.csv")

	if err != nil {
		//log.Fatal(err) // perhaps handle this nicer
		Helpers.RespondWithJSON(w, http.StatusInternalServerError, err)
		return
	}

	defer format.Close()

	w.Header().Set("Content-Type", "text/csv") // <-- set the content-type header

	w.Header().Set("Content-Disposition: attachment", "filename=format.csv")

	// Write the body to file
	_, err = io.Copy(w, format)
}

//Others

func readCsv() {
	csvfile, err := os.Open("input.csv")
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}

	// Parse the file
	r := csv.NewReader(csvfile)
	//r := csv.NewReader(bufio.NewReader(csvfile))

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Question: %s Answer %s\n", record[0], record[1])
	}
}

func massiveUpload(w http.ResponseWriter, r *http.Request) {

	var products []Models.Product

	defer r.Body.Close()
	log.Println(r.Body)

	user := context.Get(r, "user")

	log.Println("user", user)

	err := json.NewDecoder(r.Body).Decode(&products)

	if err != nil {
		log.Println(err)
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, element := range products {
		log.Println("element", element)
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

/*********** Commerssia  *********/

func checkProductQuantityCommerssia(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	log.Println(r.Body)

	var v interface{}
	err := json.NewDecoder(r.Body).Decode(&v)

	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		fmt.Println("err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	parsedV := v.(map[string]interface{})

	fmt.Println("v", parsedV["reference"])

	validation := makeCommerssiaRequest(parsedV["reference"].(string))

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]int{"message": validation})
}

func makeCommerssiaRequest(reference string) int {

	//var text = "&lt;DATOS&gt;&lt;USUARIO&gt;624154454F2912704B06435E06425001703E0BE3AFBC&lt;/USUARIO&gt;&lt;CLAVE&gt;624154454F2912704B06435E06425001706657A2F2&lt;/CLAVE&gt;&lt;NOMBRE&gt;CONSULTAINVENTARIOREFERENCIA&lt;/NOMBRE&gt;&lt;REFCODIGO&gt;00020&lt;/REFCODIGO&gt;&lt;ALMCODIGO&gt;&lt;/ALMCODIGO&gt;&lt;IDEMP&gt;SFARMA&lt;/IDEMP&gt;&lt;/DATOS&gt;"

	//var xml = "<?xml version=\"1.0\" ?><S:Envelope xmlns:S=\"http://schemas.xmlsoap.org/soap/envelope/\"><S:Body><wm_Reporte xmlns=\"http://tempuri.org/\"><pi_sEntrada>&lt;DATOS&gt;&lt;USUARIO&gt;624154454F2912704B06435E06425001703E0BE3AFBC&lt;/USUARIO&gt;&lt;CLAVE&gt;624154454F2912704B06435E06425001706657A2F2&lt;/CLAVE&gt;&lt;NOMBRE&gt;CONSULTAINVENTARIOREFERENCIA&lt;/NOMBRE&gt;&lt;REFCODIGO&gt;00020&lt;/REFCODIGO&gt;&lt;ALMCODIGO&gt;&lt;/ALMCODIGO&gt;&lt;IDEMP&gt;SFARMA&lt;/IDEMP&gt;&lt;/DATOS&gt;</pi_sEntrada></wm_Reporte></S:Body></S:Envelope>"

	var xml = "<?xml version=\"1.0\" ?><S:Envelope xmlns:S=\"http://schemas.xmlsoap.org/soap/envelope/\"><S:Body><wm_Reporte xmlns=\"http://tempuri.org/\"><pi_sEntrada>&lt;DATOS&gt;&lt;USUARIO&gt;624154454F2912704B06435E06425001703E0BE3AFBC&lt;/USUARIO&gt;&lt;CLAVE&gt;624154454F2912704B06435E06425001706657A2F2&lt;/CLAVE&gt;&lt;NOMBRE&gt;CONSULTAINVENTARIOREFERENCIA&lt;/NOMBRE&gt;&lt;REFCODIGO&gt;" + reference + "&lt;/REFCODIGO&gt;&lt;ALMCODIGO&gt;&lt;/ALMCODIGO&gt;&lt;IDEMP&gt;SFARMA&lt;/IDEMP&gt;&lt;/DATOS&gt;</pi_sEntrada></wm_Reporte></S:Body></S:Envelope>"

	//fmt.Println("xml", xml)

	var xmlStr = []byte(xml)

	req, err := http.NewRequest("POST", "http://auditoria.comerssia.com/PDPIntegracion/wsintegracion.asmx?op=wm_Reporte", bytes.NewBuffer(xmlStr))
	if err != nil {
		fmt.Println(err.Error())
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("SOAPAction", "http://tempuri.org/wm_Reporte")
	req.Header.Set("Content-Type", "text/xml")

	// Do the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	fresp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}
	resp.Body.Close()
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}
	//fmt.Println(string(f))

	var parsedResponse = strings.Replace(string(fresp), "<?xml version=\"1.0\" encoding=\"utf-8\"?><soap:Envelope xmlns:soap=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\"><soap:Body><wm_ReporteResponse xmlns=\"http://tempuri.org/\"><wm_ReporteResult>", "", -1)

	parsedResponse = strings.Replace(parsedResponse, "</wm_ReporteResult></wm_ReporteResponse></soap:Body></soap:Envelope>", "", -1)

	//fmt.Println("parsedResponse", parsedResponse)

	dec, err := base64.StdEncoding.DecodeString(parsedResponse)
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	f, err := os.Create("./files/response.zip")
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}
	defer f.Close()

	if _, err := f.Write(dec); err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}
	if err := f.Sync(); err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	files, err := Unzip("./files/response.zip", "output-folder")
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	//fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))

	//fmt.Println("files[0]" + files[0])

	contentF, err := ioutil.ReadFile(files[0])
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	// Convert []byte to string and print to screen
	textF := string(contentF)
	//fmt.Println(textF)

	e := os.Remove(files[0])
	if e != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	m, err := mxj.NewMapXmlSeq([]byte(textF))
	if err != nil {
		fmt.Println("err:", err.Error())
		return -1
	}

	//fmt.Println("m", len(m["RESPUESTA"].(map[string]interface{})))

	respuesta := m["RESPUESTA"].(map[string]interface{})

	//m["RESPUESTA"]

	var amount int

	amount = 0

	if respuesta != nil {
		//fmt.Println(respuesta["REGISTROS"].(map[string]interface{}))

		registersParent, ok := respuesta["REGISTROS"].(map[string]interface{})

		if ok == true {

			registers, ok := registersParent["REGISTRO"].([]interface{})

			if ok == true {
				for index, _ := range registers {
					//fmt.Println("f", f)
					//fmt.Println("index", index)

					register := registers[index].(map[string]interface{})

					INVTotal := register["INVTotal"].(map[string]interface{})

					INVTotalData := INVTotal["#text"].(string)

					//fmt.Println("INVTotalData", INVTotalData)

					f, _ := strconv.ParseFloat(INVTotalData, 64)

					//fmt.Println("f", f)

					s := int(f)

					//fmt.Println("s", s)

					amount = amount + s

				}
			}

			register, ok := registersParent["REGISTRO"].(map[string]interface{})

			if ok == true {
				INVTotal := register["INVTotal"].(map[string]interface{})

				INVTotalData := INVTotal["#text"].(string)

				f, _ := strconv.ParseFloat(INVTotalData, 64)

				s := int(f)

				amount = amount + s
			}
		}

		//fmt.Println("inventory qua:", amount)

		return amount
	}

	return -1

}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

//------------------------------------ commerssia transaction

func commerssiaTransactionString(totalRefs string,
	idNumber string,
	name string,
	lastName string,
	lastName2 string,
	phone string,
	address string,
	email string,
	consecutive string,
	items []interface{}) string {
	t := time.Now()
	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?><transaccion><USUARIO>624154454F2912704B06435E06425001703E0BE3AFBC</USUARIO><CLAVE>624154454F2912704B06435E06425001706657A2F2</CLAVE>")
	b.WriteString("<encabezado>")
	b.WriteString("<ENCDescripcion>Pedido Web</ENCDescripcion>")
	b.WriteString("<GMVCodigo>ADM</GMVCodigo>")
	b.WriteString("<MOVCodigo>PEDWEB</MOVCodigo>")
	b.WriteString("<movimiento>Standar</movimiento>")
	b.WriteString("<USUCodigo>PW</USUCodigo>")
	b.WriteString("<CAJCodigo>ADM0101</CAJCodigo>")
	b.WriteString("<IDEMP>SFARMA</IDEMP>")
	b.WriteString("<ALMCodigo>ADM01</ALMCodigo>")
	b.WriteString("<ALMNombre>ADMINISTRATIVO</ALMNombre>")
	b.WriteString("<MONCodigo>1</MONCodigo>")
	b.WriteString("<ENCFechaTrx>" + fmt.Sprintf("%d/%02d/%02d",
		t.Year(), t.Month(), t.Day(),
	) + "</ENCFechaTrx>")
	b.WriteString("<ENCHoraTrx>" + fmt.Sprintf("%02d:%02d:%02d",
		t.Hour(), t.Minute(), t.Second()) + "</ENCHoraTrx>")
	b.WriteString("<ENCModo>L-C</ENCModo>")
	b.WriteString("<ENCTipoProc>Standar</ENCTipoProc>")
	// strconv.Itoa(rand.Intn(99999-1+1)+1)
	b.WriteString("<ENCConsTrx>P0" + consecutive + "</ENCConsTrx>")
	b.WriteString("<ENCTasaConversion>1</ENCTasaConversion>")
	b.WriteString("<ENCTotalReferencias>" + totalRefs + "</ENCTotalReferencias>")
	b.WriteString("<ENCBruto>0</ENCBruto>")
	b.WriteString("<ENCDescuento>0</ENCDescuento>")
	b.WriteString("<ENCPagNoVenta>0</ENCPagNoVenta>")
	b.WriteString("<ENCVenta>0</ENCVenta>")
	b.WriteString("<ENCImpuestos>0</ENCImpuestos>")
	b.WriteString("<ENCComision>0</ENCComision>")
	b.WriteString("<ENCNeto>0</ENCNeto>")
	b.WriteString("<ENCRecaudo>0</ENCRecaudo>")
	b.WriteString("<ENCImpuestoAsumido>0</ENCImpuestoAsumido>")
	b.WriteString("<ENCPuntos>0</ENCPuntos>")
	b.WriteString("<ENCEstadoLinea>L</ENCEstadoLinea>")
	b.WriteString("<ENCRespuesta>OK</ENCRespuesta>")
	b.WriteString("<ENCDescRespuesta>NO APLICA</ENCDescRespuesta>")
	b.WriteString("</encabezado>")
	b.WriteString("<detalle>")
	b.WriteString("<items>")

	//default items
	var defaultItems []map[string]string

	itemD := map[string]string{
		"nitem":           "1",
		"ICPPresentacion": "C" + idNumber,
		"ICPDescripcion":  "Cedula/Nit",
		"ICPCadena":       "C" + idNumber,
		"ICPLetra":        "CLI",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "2",
		"ICPPresentacion": name,
		"ICPDescripcion":  "Nombre",
		"ICPCadena":       name,
		"ICPLetra":        "NOM",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "3",
		"ICPPresentacion": lastName,
		"ICPDescripcion":  "Primer Apellido",
		"ICPCadena":       lastName,
		"ICPLetra":        "APE",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "4",
		"ICPPresentacion": lastName2,
		"ICPDescripcion":  "Segundo Apellido",
		"ICPCadena":       lastName2,
		"ICPLetra":        "SEGAP",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "5",
		"ICPPresentacion": phone,
		"ICPDescripcion":  "Telefono",
		"ICPCadena":       phone,
		"ICPLetra":        "TEL",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "6",
		"ICPPresentacion": address,
		"ICPDescripcion":  "Direccion",
		"ICPCadena":       address,
		"ICPLetra":        "DIR",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "7",
		"ICPPresentacion": email,
		"ICPDescripcion":  "Correo Electronico",
		"ICPCadena":       email,
		"ICPLetra":        "MAIL",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "8",
		"ICPPresentacion": "EFECTIVO",
		"ICPDescripcion":  "Forma de pago",
		"ICPCadena":       "EFECTIVO",
		"ICPLetra":        "FORMPA",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "9",
		"ICPPresentacion": "IMPLE",
		"ICPDescripcion":  "Vendedor",
		"ICPCadena":       "IMPLE",
		"ICPLetra":        "VEN",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "10",
		"ICPPresentacion": consecutive,
		"ICPDescripcion":  "Consecutivo Pedido Web",
		"ICPCadena":       consecutive,
		"ICPLetra":        "NUMPEDWEB",
	}

	defaultItems = append(defaultItems, itemD)

	itemD = map[string]string{
		"nitem":           "11",
		"ICPPresentacion": "ClienteWeb",
		"ICPDescripcion":  "Tipo Cliente Pedido Web",
		"ICPCadena":       "1",
		"ICPLetra":        "TCL",
	}

	defaultItems = append(defaultItems, itemD)

	normalItems := comerssiaTransactionNomalItem(defaultItems)

	b.WriteString(normalItems)

	transacItems := []map[string]string{}

	for n := 0; n < len(items); n++ {
		transacItem := map[string]string{}
		parsedItem := items[n].(map[string]interface{})
		transacItem["reference"] = parsedItem["product_reference"].(string)
		transacItem["name"] = parsedItem["product_name"].(string)
		transacItem["price"] = parsedItem["product_price"].(string)
		transacItem["quantity"] = parsedItem["product_quantity"].(string)
		transacItems = append(transacItems, transacItem)
	}

	//fmt.Println("transacItems", transacItems)

	productItems, _ := comerssiaTransactionProductItem(transacItems, 12)

	b.WriteString(productItems)

	b.WriteString("</items>")
	b.WriteString("</detalle>")
	b.WriteString("</transaccion>")
	return b.String()
}

func comerssiaTransactionNomalItem(items []map[string]string) string {
	var b bytes.Buffer

	for n := 0; n < len(items); n++ {
		parsedItem := items[n]
		b.WriteString("<item nitem=\"" + parsedItem["nitem"] + "\" Tipo=\"Letra\" Visible=\"True\" Imprime=\"True\">")
		b.WriteString("<ICPPresentacion>" + parsedItem["ICPPresentacion"] + "</ICPPresentacion>")
		b.WriteString("<ICPDescripcion>" + parsedItem["ICPDescripcion"] + "</ICPDescripcion>")
		b.WriteString("<ICPCadena>" + parsedItem["ICPCadena"] + "</ICPCadena>")
		b.WriteString("<ICPLetra>" + parsedItem["ICPLetra"] + "</ICPLetra>")
		b.WriteString("</item>")
	}

	return b.String()
}

func comerssiaTransactionProductItem(items []map[string]string, j int) (string, int) {
	var b bytes.Buffer

	for n := 0; n < len(items); n++ {
		parsedItem := items[n]
		b.WriteString("<item nitem=\"" + strconv.Itoa(j) + "\" Tipo=\"Referencia\" Tiporef=\"normal\" Visible=\"True\">")
		b.WriteString("<REFCodClasificacion></REFCodClasificacion>")
		b.WriteString("<REFCodigo1>" + parsedItem["reference"] + "</REFCodigo1>")
		b.WriteString("<REFCodigo2>" + parsedItem["reference"] + "</REFCodigo2>")
		b.WriteString("<REFNombreLargo>" + parsedItem["name"] + "</REFNombreLargo>")
		b.WriteString("<CARCodigo1></CARCodigo1>")
		b.WriteString("<REFPrecioLista>" + parsedItem["price"] + "</REFPrecioLista>")
		b.WriteString("<IRFBruto>" + parsedItem["price"] + "</IRFBruto>")
		b.WriteString("<IRFDescuento>0</IRFDescuento>")
		b.WriteString("<IRFPago>" + parsedItem["price"] + "</IRFPago>")
		b.WriteString("<IRFCantidad>" + parsedItem["quantity"] + "</IRFCantidad>")
		b.WriteString("<IRFValorImpuesto>0</IRFValorImpuesto>")
		b.WriteString("<IRFImpuesto>0</IRFImpuesto>")
		b.WriteString("<REFEsCombo>0</REFEsCombo>")
		b.WriteString("<REFUltimoCosto>0</REFUltimoCosto>")
		b.WriteString("<PRVCodigo>CODE</PRVCodigo>")
		b.WriteString("<REFCapturaSerial>false</REFCapturaSerial>")
		b.WriteString("<REFManejaLotes>false</REFManejaLotes>")
		b.WriteString("<REFFactorConversion>1</REFFactorConversion>")
		b.WriteString("<REFInventario></REFInventario>")
		b.WriteString("<REFEsParaVenta></REFEsParaVenta>")
		b.WriteString("<estado>ACTIVO</estado>")
		b.WriteString("<IRFPagNoVenta>0</IRFPagNoVenta>")
		b.WriteString("<IRFVenta>0</IRFVenta>")
		b.WriteString("<IRFValorImpuestoNeto>0</IRFValorImpuestoNeto>")
		b.WriteString("<IRFComision>0</IRFComision>")
		b.WriteString("<IRFImpuestoAsumido>0</IRFImpuestoAsumido>")
		b.WriteString("<IRFNeto>0</IRFNeto>")
		b.WriteString("<REFPuntos>0</REFPuntos>")
		b.WriteString("</item>")
		j++
	}

	return b.String(), j
}

func makeCommerssiaRequestTransaction(id_order string) int {
	//http://auditoria.comerssia.com/PDPIntegracion/wsintegracion.asmx?op=wm_EnvioTransacciones

	// ------------  services proccess

	resultOrders, err := getAllRequest("https://sfarmadroguerias.com/api/orders?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[id,id_cart,id_customer,payment,current_state,total_discounts,total_paid,total_products,id_address_delivery]&output_format=JSON&filter[id]=" + id_order)

	if err != nil {

		return 0
	}

	ordersObject := resultOrders["orders"]

	ordersObjectDetail := ordersObject[0].(map[string]interface{})

	time.Sleep(2 * time.Second)

	resultAddress, err2 := getAllRequest("https://sfarmadroguerias.com/api/addresses?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[dni,lastname,firstname,address1,phone]&output_format=JSON&filter[id]=" + ordersObjectDetail["id_address_delivery"].(string))

	if err2 != nil {

		return 0
	}

	addressObject := resultAddress["addresses"]

	addressObjectDetail := addressObject[0].(map[string]interface{})

	//fmt.Println("resultAddress", resultAddress)

	time.Sleep(2 * time.Second)

	resultCustomer, err3 := getAllRequest("https://sfarmadroguerias.com/api/customers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[email]&output_format=JSON&sort=id_DESC&filter[id]=" + ordersObjectDetail["id_customer"].(string))

	if err3 != nil {

		return 0
	}

	customersObject := resultCustomer["customers"]

	customersObjectDetail := customersObject[0].(map[string]interface{})

	//fmt.Println("resultCustomer", resultCustomer)

	time.Sleep(2 * time.Second)

	resultProducts, err4 := getAllRequest("https://sfarmadroguerias.com/api/order_details?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[product_name,product_quantity,product_price,product_reference]&output_format=JSON&filter[id_order]=" + id_order)

	if err4 != nil {

		return 0
	}

	//fmt.Println("resultProducts", len(resultProducts["order_details"]))

	totalRefs := strconv.Itoa(len(resultProducts["order_details"]))
	idNumber := addressObjectDetail["dni"].(string)
	name := addressObjectDetail["firstname"].(string)
	lastName := addressObjectDetail["lastname"].(string)
	lastName2 := ""
	phone := addressObjectDetail["phone"].(string)
	address := addressObjectDetail["address1"].(string)
	email := customersObjectDetail["email"].(string)
	consecutive := id_order

	// ------------  services proccess

	xmlTest := commerssiaTransactionString(totalRefs, idNumber, name, lastName, lastName2, phone, address, email, consecutive, resultProducts["order_details"])

	// title := "proccess" + strconv.Itoa(rand.Intn(99999-1+1)+1)
	title := "proccess"
	file, err := os.Create("transactions/" + title + ".xml")
	if err != nil {
		fmt.Println(err)
	} else {
		file.WriteString(xmlTest)
	}

	defer file.Close()

	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	file2, err2 := os.OpenFile("transactions/"+title+".zip", flags, 0644)

	if err2 != nil {
		fmt.Println("err2", err2)
	}

	zipw := zip.NewWriter(file2)

	file3, err3 := os.Open("transactions/" + title + ".xml")

	if err3 != nil {
		fmt.Println("err3", err3)
	}

	defer file3.Close()

	wr, err4 := zipw.Create(title + ".xml")

	if err4 != nil {
		fmt.Println("err4", err4)
	}

	if _, err5 := io.Copy(wr, file3); err5 != nil {
		fmt.Println("err5", err5)
	}

	defer zipw.Close()

	generateBase64 := func(title string) {

		time.Sleep(3 * time.Second)

		bytesE, err6 := ioutil.ReadFile("transactions/" + title + ".zip")

		//fmt.Println("bytes", bytes)

		if err6 != nil {
			fmt.Println("err6", err6)
		}

		base64Encoding := base64.StdEncoding.EncodeToString(bytesE)

		testRequestSend(base64Encoding)
	}

	go generateBase64(title)

	return 0
}

func RandStringBytes(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func checkPaymentsToSend() {
	resultPayments, _ := getAllRequest("https://sfarmadroguerias.com/api/order_payments?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[id,order_reference,amount,date_add]&output_format=JSON&sort=id_DESC&limit=20")
	paymentsObject := resultPayments["order_payments"]
	paymentsObjectDetail := paymentsObject[0].(map[string]interface{})

	lastID := paymentsObjectDetail["id"].(float64)

	if lastID > lastTracked {
		for n := 0; n < len(paymentsObject); n++ {
			parsedPayment := paymentsObject[n].(map[string]interface{})
			if parsedPayment["id"].(float64) > lastTracked {
				orderReference := parsedPayment["order_reference"].(string)
				pureReference := strings.Replace(orderReference, "ORD-", "", 1)
				pureReferenceInt, _ := strconv.Atoi(pureReference)
				pureReferenceStr := strconv.Itoa(pureReferenceInt)
				makeCommerssiaRequestTransaction(pureReferenceStr)
				time.Sleep(5 * time.Second)
			}
		}

		lastTracked = lastID

	}

}

func testRequestSend(base string) {
	var b bytes.Buffer
	b.WriteString("<soap:Envelope xmlns:soap=\"http://www.w3.org/2003/05/soap-envelope\"	xmlns:tem=\"http://tempuri.org/\">")
	b.WriteString("<soap:Header/>")
	b.WriteString("<soap:Body>")
	b.WriteString("<tem:wm_EnvioTransaccionesCompletarTrx>")
	b.WriteString("<tem:pi_sIdemp>")
	b.WriteString("SFARMA")
	b.WriteString("</tem:pi_sIdemp>")
	b.WriteString("<tem:pi_sEnvio>")
	b.WriteString(base)
	b.WriteString("</tem:pi_sEnvio>")
	b.WriteString("</tem:wm_EnvioTransaccionesCompletarTrx>")
	b.WriteString("</soap:Body>")
	b.WriteString("</soap:Envelope>")

	var xmlStr = []byte(b.String())

	req, err := http.NewRequest("POST", "http://auditoria.comerssia.com/PDPIntegracion/wsintegracion.asmx", bytes.NewBuffer(xmlStr))
	if err != nil {
		fmt.Println(err.Error())
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", "text/xml")

	// Do the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	// fmt.Println("resp", resp)

	body, err := ioutil.ReadAll(resp.Body)
	bodyString := string(body)
	resp.Body.Close()

	fmt.Println("bodyString", bodyString)
}
