package main

import (
	"net/http"

	Models "github.com/sumaikun/sfarma-rest-api/models"
	"github.com/thedevsaddam/govalidator"
)

func signUpValidator(r *http.Request) (map[string]interface{}, Models.User) {

	var user Models.User

	rules := govalidator.MapData{
		"name":       []string{"required"},
		"email":      []string{"required", "email"},
		"lastName":   []string{"required"},
		"laboratory": []string{"required"},
	}

	opts := govalidator.Options{
		Request:         r,
		Data:            &user,
		Rules:           rules,
		RequiredDefault: true,
	}

	v := govalidator.New(opts)
	e := v.ValidateJSON()
	//fmt.Println(user)

	err := map[string]interface{}{"validationError": e}

	return err, user
}

func userValidator(r *http.Request) (map[string]interface{}, Models.User) {

	var user Models.User

	rules := govalidator.MapData{
		"name":       []string{"required"},
		"email":      []string{"required", "email"},
		"lastName":   []string{"required"},
		"laboratory": []string{"required"},
		"role":       []string{"required"},
		//"picture": []string{"url"},
	}

	opts := govalidator.Options{
		Request:         r,
		Data:            &user,
		Rules:           rules,
		RequiredDefault: true,
	}

	v := govalidator.New(opts)
	e := v.ValidateJSON()
	//fmt.Println(user)

	err := map[string]interface{}{"validationError": e}

	return err, user
}

func productValidator(r *http.Request) (map[string]interface{}, Models.Product) {

	var product Models.Product

	rules := govalidator.MapData{
		"name":        []string{"required"},
		"category":    []string{"required", "numeric"},
		"description": []string{"required"},
		"state":       []string{"stateEnum"},
		//"laboratory":  []string{"required", "numeric"},
	}

	opts := govalidator.Options{
		Request:         r,
		Data:            &product,
		Rules:           rules,
		RequiredDefault: true,
	}

	v := govalidator.New(opts)
	e := v.ValidateJSON()
	//fmt.Println(user)

	err := map[string]interface{}{"validationError": e}

	return err, product
}

func transferValidator(r *http.Request) (map[string]interface{}, Models.Transfer) {

	var transfer Models.Transfer

	rules := govalidator.MapData{
		"product": []string{"required"},
		"user":    []string{"required"},
	}

	opts := govalidator.Options{
		Request:         r,
		Data:            &transfer,
		Rules:           rules,
		RequiredDefault: true,
	}

	v := govalidator.New(opts)
	e := v.ValidateJSON()
	//fmt.Println(user)

	err := map[string]interface{}{"validationError": e}

	return err, transfer
}
