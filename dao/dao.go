package dao

import (
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database

//MongoConnector struct for connections access
type MongoConnector struct {
	Server   string
	Database string
}

//Connect golang to mongo sb
func (mongo *MongoConnector) Connect() {
	session, err := mgo.Dial(mongo.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(mongo.Database)
}

//FindAll from repository
func (mongo *MongoConnector) FindAll(collection string) ([]interface{}, error) {
	var data []interface{}
	err := db.C(collection).Find(bson.M{}).All(&data)
	return data, err
}

//Insert into repository
func (mongo *MongoConnector) Insert(collection string, data interface{}, uniqueKeys []string) error {

	for _, key := range uniqueKeys {
		index := mgo.Index{
			Key:    []string{key},
			Unique: true,
		}
		if err := db.C(collection).EnsureIndex(index); err != nil {
			return err
		}
	}

	err := db.C(collection).Insert(&data)
	return err
}

//FindByID in repository
func (mongo *MongoConnector) FindByID(collection string, id string) (interface{}, error) {

	//fmt.Println(collection, id)

	var data interface{}
	err := db.C(collection).FindId(bson.ObjectIdHex(id)).One(&data)
	return data, err
}

// DeleteByID by id on repository
func (mongo *MongoConnector) DeleteByID(collection string, id string) error {

	err := db.C(collection).RemoveId(bson.ObjectIdHex(id))
	//err := db.C(COLLECTION).Remove(&movie)
	return err
}

// Update an existing movie
func (mongo *MongoConnector) Update(collection string, id interface{}, data interface{}) error {

	err := db.C(collection).UpdateId(id, &data)
	return err
}

//FindOneByKEY with key and value specified in repository
func (mongo *MongoConnector) FindOneByKEY(collection string, key string, value string) (interface{}, error) {
	var data interface{}
	err := db.C(collection).Find(bson.M{key: value}).One(&data)
	return data, err
}
