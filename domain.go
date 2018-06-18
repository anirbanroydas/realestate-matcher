// Package domain consists of all the core domain objects of the business/application.
//
// Domain:)
// 2. Transaction - This places an order for a customer (which makes a customer receive the order),
// this contain transcation infomration like order, payment details etc
//
// So by using the **Domain Experts** knowledge and seeing the **Ubiquitous Language**,
// we can derive at some domain objects
//
// This package basically creates all the domain object and expose some interfaces
// to use them but doesnt depend on anything outside the domain layer
package main

import (
	"time"
)

// Transaction is an aggregrate which specifies what is the Order on which a transaction is taking place
// alongs with information like payment details
type Property struct {
	PropertyID uint64  `gorm:"primary_key"`
	Latitude   float32 `gorm:"index:idx_properties_latitude_longitude"`
	Longitude  float32 `gorm:"index:idx_properties_latitude_longitude"`
	Price      float32
	Bedrooms   uint16
	Bathrooms  uint16
	AddedDate  time.Time
}

func NewProperty(lat, lon, price float32, bedrooms, bathrooms uint16) *Property {
	p := Property{
		Latitude:  lat,
		Longitude: lon,
		Price:     price,
		Bedrooms:  bedrooms,
		Bathrooms: bathrooms,
		AddedDate: time.Now().UTC(),
	}
	return &p
}

type Requirement struct {
	RequirementID uint64  `gorm:"primary_key"`
	Latitude      float32 `gorm:"index:idx_requirements_latitude_longitude"`
	Longitude     float32 `gorm:"index:idx_requirements_latitude_longitude"`
	MinBudget     float32
	MaxBudget     float32
	MinBedrooms   uint16
	MaxBedrooms   uint16
	MinBathrooms  uint16
	MaxBathrooms  uint16
	AddedDate     time.Time
}

func NewRequirement(lat, lon, minBudget, maxBudget float32, minBedrooms, maxBedrooms, minBathrooms, maxBathrooms uint16) *Requirement {
	r := Requirement{
		Latitude:     lat,
		Longitude:    lon,
		MinBudget:    minBudget,
		MaxBudget:    maxBudget,
		MinBedrooms:  minBedrooms,
		MaxBedrooms:  maxBedrooms,
		MinBathrooms: minBathrooms,
		MaxBathrooms: maxBathrooms,
		AddedDate:    time.Now().UTC(),
	}
	return &r
}

func validCoordinate(lat, long float32) bool {
	if lat < -90 || lat > 90 || long < -180 || long > 180 {
		return false
	}
	return true
}

func validBudget(minBudget, maxBudget float32) bool {
	if minBudget < 0 || maxBudget < 0 || minBudget > maxBudget {
		return false
	}
	return true
}

func validBedroomsRange(minRooms, maxRooms uint16) bool {
	return validIntRange(minRooms, maxRooms)
}

func validBathroomsRange(minRooms, maxRooms uint16) bool {
	return validIntRange(minRooms, maxRooms)
}

func validIntRange(x, y uint16) bool {
	if x < 0 || y < 0 || x > y {
		return false
	}
	return true
}

func validPrice(price float32) bool {
	return price > 0
}

func validBedrooms(bedRooms uint16) bool {
	return bedRooms > 0
}

func validBathrooms(bathRooms uint16) bool {
	return bathRooms > 0
}
