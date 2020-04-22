package models

import "gopkg.in/mgo.v2/bson"

//User representation on mongo
type User struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	Name       string        `bson:"name" json:"name"`
	Password   string        `bson:"password" json:"password"`
	Email      string        `bson:"email" json:"email"`
	LastName   string        `bson:"lastName" json:"lastName"`
	Role       string        `bson:"role" json:"role"`
	Laboratory int           `bson:"laboratory" json:"laboratory"`
	Picture    string        `bson:"picture" json:"picture"`
	Date       string        `bson:"date" json:"date"`
	UpdateDate string        `bson:"update_date" json:"update_date"`
}

//Product representation on mongo
type Product struct {
	ID                       bson.ObjectId `bson:"_id" json:"id"`
	Name                     string        `bson:"name" json:"name"`
	User                     bson.ObjectId `bson:"user" json:"user"`
	State                    string        `bson:"state" json:"state"`
	Description              string        `bson:"description" json:"description"`
	ExternalBoxDesc          string        `bson:"externalBoxDesc" json:"externalBoxDesc"`
	InternalBoxDesc          string        `bson:"internalBoxDesc" json:"internalBoxDesc"`
	CodeCopidrogas           string        `bson:"codeCopidrogas" json:"codeCopidrogas"`
	InternalManufacturerCode string        `bson:"internalManufacturerCode" json:"internalManufacturerCode"`
	MedicineType             string        `bson:"medicineType" json:"medicineType"`
	Appearance               string        `bson:"appearance" json:"appearance"`
	Laboratory               string        `bson:"laboratory" json:"laboratory"`
	Dimens                   string        `bson:"dimens" json:"dimens"`
	Weight                   string        `bson:"weight" json:"weight"`
	MesaureUnit              string        `bson:"measureUnit" json:"measureUnit"`
	AmountSized              string        `bson:"amountSized" json:"amountSized"`
	Indications              string        `bson:"indications" json:"indications"`
	ContraIndications        string        `bson:"contraIndications" json:"contraIndications"`
	Precautions              string        `bson:"precautions" json:"precautions"`
	AdministrationWay        string        `bson:"administrationWay" json:"administrationWay"`
	Category                 string        `bson:"category" json:"category"`
	Picture                  string        `bson:"picture" json:"picture"`
	Date                     string        `bson:"date" json:"date"`
	UpdateDate               string        `bson:"update_date" json:"update_date"`
}

//Transfer representation in mongo
type Transfer struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	Product    bson.ObjectId `bson:"product" json:"product"`
	User       bson.ObjectId `bson:"user" json:"user"`
	Reference  string        `bson:"reference" json:"reference"`
	Price      string        `bson:"price" json:"price"`
	ProductID  string        `bson:"productID" json:"productID"`
	Date       string        `bson:"date" json:"date"`
	UpdateDate string        `bson:"update_date" json:"update_date"`
}
