// Package gateways constains all the interfaces, gateways etc to communicate between other service, db
// and act as a bridge between the infra and usecases
package main

import (
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// EstabilishConnection takes the database config object loaded from YAML file and
// creates a connection with mysql server. It also takes care of connection pool.
func NewDBClient() (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	pass := os.Getenv("MYSQL_PASS")
	dbName := os.Getenv("MYSQL_DATABASE")

	dbURL := user + ":" + pass + "@/tcp(" + host + ":" + port + ")/" + dbName + "?charset=utf8&parseTime=True&loc=Local"

	db, err = gorm.Open("mysql", dbURL)
	if err != nil {
		log.Println("Error :: Could not estabilish connection to mysql server")
		return db, err
	}
	defer db.Close()

	// db.DB().SetMaxIdleConns(0)
	// db.DB().SetMaxOpenConns(20)
	db.LogMode(true)

	return db, nil
}
