package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
)

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
		log.Fatalln(err)
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
		log.Fatalln(err)
	}

	// Copy the actual file content to the field field's writer
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		log.Fatalln(err)
	}

	multiPartWriter.Close()

	req, err := http.NewRequest("POST", "https://sfarmadroguerias.com/api/images/products/"+productID+"?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV", &requestBody)
	if err != nil {
		log.Fatalln(err)
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	log.Println(result)
}

func createPrestaShopProduct(w http.ResponseWriter, r *http.Request) {

	fmt.Println("prestashop product")

	var createProduct Models.CreateProduct
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&createProduct)

	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(createProduct)

	product, err := dao.FindByID("products", createProduct.Product)
	if err != nil {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	//log.Println(product)

	parsedProduct := product.(bson.M)

	if parsedProduct["state"] == "sended" {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Product Already Sended")
		return
	}

	if parsedProduct["picture"] == "" {
		Helpers.RespondWithError(w, http.StatusBadRequest, "Product Need Image")
		return
	}

	xml := returnXML(createProduct.Reference, parsedProduct["laboratory"].(string), createProduct.Price, parsedProduct["category"].(string), parsedProduct["description"].(string), parsedProduct["name"].(string))

	//log.Println("xml", xml)

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

	if err := dao.Update("products", parsedProduct["_id"], parsedProduct); err != nil {
		Helpers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	Helpers.RespondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})

}

func getAllRequest(url string) map[string][]interface{} {
	// By now our original request body should have been populated, so let's just use it with our custom request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Output-Format", "JSON")

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	//fmt.Println(response.Body)

	var result map[string][]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	return result

}

func getRequest(url string) map[string]interface{} {
	// By now our original request body should have been populated, so let's just use it with our custom request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	// We need to set the content type from the writer, it includes necessary boundary as well
	req.Header.Set("Output-Format", "JSON")

	// Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	//fmt.Println(response.Body)

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	return result

}

func proccessPrestaShopDistributors() {

	result := getAllRequest("https://sfarmadroguerias.com/api/manufacturers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	var slice []interface{}

	for _, element := range result["manufacturers"] {

		md, _ := element.(map[string]interface{})

		subelement := constructDistributors(fmt.Sprintf("%g", md["id"]))

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

func constructDistributors(id string) map[string]interface{} {

	result := getRequest("https://sfarmadroguerias.com/api/manufacturers/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	//log.Println(result)

	manufacturer, _ := result["manufacturer"].(map[string]interface{})

	//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

	return manufacturer

}

func proccessPrestaShopProductcategories() {

	result := getAllRequest("https://sfarmadroguerias.com/api/categories?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&output_format=JSON&display=[id,name,active]")
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

	result := getRequest("https://sfarmadroguerias.com/api/categories/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	//log.Println(result)

	category, _ := result["category"].(map[string]interface{})

	//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

	return category
}

// Get products from prestashop

func proccessPrestashopProducts() {

	result := getAllRequest("https://sfarmadroguerias.com/api/products?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&display=[id,name,reference,price,id_manufacturer,description,id_category_default,id_default_image,active,id_supplier]&output_format=JSON")

	for _, element := range result["products"] {
		md, _ := element.(map[string]interface{})
		//fmt.Println("Key:", key, "=>", "Element:", md["name"])
		if md["active"] == "1" {

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
						localProduct.State = "inShop"
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
								return
							}
						} else {
							//fmt.Println("product exist")
							parsedExist := exists.([]interface{})[0].(bson.M)
							localProduct.ID = parsedExist["_id"].(bson.ObjectId)
							if err := dao.Update("products", localProduct.ID, localProduct); err != nil {
								fmt.Println(err)
								return
							}
						}

						//fmt.Println("localProduct", localProduct)
					}
				}

			}
		}

	}

}

// Get suppliers from prestashop

func proccessPrestashopSuppliers() {

	result := getAllRequest("https://sfarmadroguerias.com/api/suppliers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV&output_format=JSON&display=[id,name,active]")

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
