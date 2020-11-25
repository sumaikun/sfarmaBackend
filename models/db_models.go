package models

import "gopkg.in/mgo.v2/bson"

//User representation on mongo
type User struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	Name          string        `bson:"name" json:"name"`
	State         string        `bson:"state" json:"state"`
	Password      string        `bson:"password" json:"password"`
	Email         string        `bson:"email" json:"email"`
	LastName      string        `bson:"lastName" json:"lastName"`
	Role          string        `bson:"role" json:"role"`
	Laboratory    int           `bson:"laboratory" json:"laboratory"`
	Picture       string        `bson:"picture" json:"picture"`
	ResetPassword bool          `bson:"resetPassword" json:"resetPassword"`
	Conditions    bool          `bson:"conditions" json:"conditions"`
	Date          string        `bson:"date" json:"date"`
	UpdateDate    string        `bson:"update_date" json:"update_date"`
}

//Product representation on mongo
type Product struct {
	ID                       bson.ObjectId `bson:"_id" json:"id"`
	Name                     string        `bson:"name" json:"name"`
	User                     bson.ObjectId `bson:"user,omitempty" json:"user,omitempty"`
	State                    string        `bson:"state" json:"state"`
	Description              string        `bson:"description" json:"description"`
	ExternalBoxDesc          string        `bson:"externalBoxDesc" json:"externalBoxDesc"`
	InternalBoxDesc          string        `bson:"internalBoxDesc" json:"internalBoxDesc"`
	SubClassification        string        `bson:"subClassification" json:"subClassification"`
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
	Picture2                 string        `bson:"picture2" json:"picture2"`
	Picture3                 string        `bson:"picture3" json:"picture3"`
	CustomerBenefit          string        `bson:"customerBenefit" json:"customerBenefit"`
	PrepakCondition          string        `bson:"prepakCondition" json:"prepakCondition"`
	SustanceCompose          string        `bson:"sustanceCompose" json:"sustanceCompose"`
	RegisterInvima           string        `bson:"registerInvima" json:"registerInvima"`
	BarCodeRegular           string        `bson:"barCodeRegular" json:"barCodeRegular"`
	AmountByReference        string        `bson:"amountByReference" json:"amountByReference"`
	ShooperClassification    interface{}   `bson:"shooperClassification" json:"shooperClassification"`
	MarketSegment            interface{}   `bson:"marketSegment" json:"marketSegment"`
	RejectJutification       string        `bson:"rejectJutification" json:"rejectJutification"`
	RecommendedPrice         string        `bson:"recommendedPrice" json:"recommendedPrice"`
	ShopDefaultReference     string        `bson:"shopDefaultReference" json:"shopDefaultReference"`
	PrestashopID             string        `bson:"prestashopId" json:"prestashopId"`
	DefaultImageID           string        `bson:"defaultImageID" json:"defaultImageID"`
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

//Laboratories representation in mongo
type Laboratories struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	PrestashopID string        `bson:"prestashopId" json:"prestashopId"`
	Name         string        `bson:"name" json:"name"`
	Date         string        `bson:"date" json:"date"`
}

//Suppliers representation in mongo
type Suppliers struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	PrestashopID string        `bson:"prestashopId" json:"prestashopId"`
	Name         string        `bson:"name" json:"name"`
	Date         string        `bson:"date" json:"date"`
}

//Categories representation in mongo
type Categories struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	PrestashopID string        `bson:"prestashopId" json:"prestashopId"`
	Name         string        `bson:"name" json:"name"`
	Date         string        `bson:"date" json:"date"`
}
