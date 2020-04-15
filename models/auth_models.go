package models

import (
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/mgo.v2/bson"
)

// Credentials is the request body of credential input
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// Users are test users for generate jwt token
var Users = map[string]string{
	"ventas.javc@gmail.com": "$2a$14$LhvKvjNkvoVKyUMAzle0DexAZCoM7RgHW.0yVaDOBG3O1psbC4XTG",
	"kotomivega@gmail.com":  "$2a$14$LhvKvjNkvoVKyUMAzle0DexAZCoM7RgHW.0yVaDOBG3O1psbC4XTG",
}

// Claims represents the struct of jwt token
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// TokenResponse represents json response after succesfully auth
type TokenResponse struct {
	Token string `json:"token"`
	User  bson.M `json:"user"`
}

// JwtKey is the sample jwt secret
//var JwtKey = []byte()
