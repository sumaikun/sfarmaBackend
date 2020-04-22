package main

import (
	"bytes"
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

			//log.Println("user found login", user)

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

	xml := returnXML(createProduct.Reference, createProduct.Price, parsedProduct["category"].(string), parsedProduct["description"].(string), parsedProduct["name"].(string))

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

func getPrestaShopDistributors(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	result := getAllRequest("https://sfarmadroguerias.com/api/manufacturers?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	//log.Println(result)

	var slice []interface{}

	for _, element := range result["manufacturers"] {
		//fmt.Println("Key:", key, "=>", "Element:", element)
		md, _ := element.(map[string]interface{})
		//fmt.Println(md)
		//fmt.Println(md["id"])
		subelement := constructDistributors(fmt.Sprintf("%g", md["id"]))

		//fmt.Println(subelement["active"])

		if subelement["active"] == "1" {
			//fmt.Println(subelement)
			slice = append(slice, subelement)
		}

	}

	//fmt.Println("Slice Result ", slice)

	Helpers.RespondWithJSON(w, http.StatusOK, slice)

}

func constructDistributors(id string) map[string]interface{} {

	result := getRequest("https://sfarmadroguerias.com/api/manufacturers/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	//log.Println(result)

	manufacturer, _ := result["manufacturer"].(map[string]interface{})

	//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

	return manufacturer

}

func getPrestaShopProductcategories(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	result := getRequest("https://sfarmadroguerias.com/api/categories/2?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	category, _ := result["category"].(map[string]interface{})

	//fmt.Println(category["associations"])

	categories := category["associations"].(map[string]interface{})

	//fmt.Println(categories["categories"])

	selectedCategories := categories["categories"].([]interface{})

	var slice []interface{}

	for _, element := range selectedCategories {
		//fmt.Println(element)
		md, _ := element.(map[string]interface{})

		//fmt.Println(md["id"])
		subelement := constructCategory(md["id"].(string))
		//fmt.Println(subelement)
		if subelement["active"] == "1" {
			//fmt.Println(subelement)
			slice = append(slice, subelement)
		}
	}

	Helpers.RespondWithJSON(w, http.StatusOK, slice)

}

func constructCategory(id string) map[string]interface{} {

	//fmt.Println("https://sfarmadroguerias.com/api/categories/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	result := getRequest("https://sfarmadroguerias.com/api/categories/" + id + "?ws_key=ITEBHIEURLT922QIBK8WRYLXS589QDPV")

	//log.Println(result)

	category, _ := result["category"].(map[string]interface{})

	//fmt.Println("Key:", manufacturer["id"], "=>", "Element:", manufacturer["name"], " ", "active", manufacturer["active"])

	return category
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

	if userParsed["role"] != "admin" {
		product.User = userParsed["_id"].(bson.ObjectId)
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

	if parsedData["user"] == nil {
		user := context.Get(r, "user")

		userParsed := user.(bson.M)

		product.User = userParsed["_id"].(bson.ObjectId)
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
