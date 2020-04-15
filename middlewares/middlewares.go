package middleware

import (
	"log"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"

	C "github.com/sumaikun/sfarma-rest-api/config"
)

// AuthMiddleware verify
func AuthMiddleware(next http.Handler) http.Handler {

	var config = C.Config{}
	config.Read()

	var JwtKey = []byte(config.Jwtkey)

	if len(JwtKey) == 0 {
		log.Fatal("HTTP server unable to start, expected an APP_KEY for JWT auth")
	}
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(JwtKey), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
	return jwtMiddleware.Handler(next)

}
