// Package main containces all method for performing allthe use cases of the application
// This package does not dpened on any other package except the domain. This package contains
// methods which can run indpendent of choice of db, web framework etc
package main

import (
	"log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

// PropListing is kind of a DTO which is used by CheckFraudulency method of
// TransactionFraudProcessor to process the transaction
type PropListing struct {
	Latitude  float32
	Longitude float32
	Price     float32
	Bedrooms  uint16
	Bathrooms uint16
}

type ReqWithDistance struct {
	Requirement
	Distance float32
}

// TransactionFraudProcessor is a usecase interactor which has methods which checks if a
// transactino if fraudulent or not using other usecase interactors or interfaces.
// It uses a FraudProcessor to process the fraud, a TransactionValidator to validate the transaction
// and a TransactionRepo to store and retreive history transactions
type PropProcessor struct {
	DB             *gorm.DB
	MatchAlgorithm PropMatchingAlgo
}

func NewPropProcessor(db *gorm.DB, pAlgo PropMatchingAlgo) PropProcessor {
	return PropProcessor{
		DB:             db,
		MatchAlgorithm: pAlgo,
	}
}

// CheckFraudulency use_case takes a TransactionRequest object as input and creates a domain level
// Transaction  object and sends it to a FraudProcess which process it from there on, asynchronously.
// It returns an error if there is a problem in any of the above processes.
func (plP PropProcessor) GetMatchingReqs(p PropListing) ([]MatchedRequirement, error) {
	var err error
	var candidateReqs []ReqWithDistance
	var matchingReqs []MatchedRequirement
	var rMargins ReqMargins

	// step 0:  validate the Property Requirement Request
	isValid := plP.validate(p)
	if !isValid {
		return matchingReqs, errors.Wrap(err, "PropProcessor couldn't validate")
	}

	// step 1: Add property listing to database
	err = plP.addToDB(p)
	if err != nil {
		return matchingReqs, errors.Wrap(err, "PropProcessor couldn't addToDB")
	}

	// step 2: Base Filtering - filter out a certain set of requirements first based on parameters which gives a set of possible candidate requirements
	candidateReqs, rMargins, err = plP.getCandidateReqs(p)
	if err != nil {
		return matchingReqs, errors.Wrap(err, "PropProcessor couldn't getCandidateReqs")
	}

	// step 3: Run algorithm on candidate requirements and get a result set of matching requirement
	matchingReqs = plP.MatchAlgorithm.Match(p, candidateReqs, rMargins)
	return matchingReqs, nil
}

func (plP PropProcessor) validate(p PropListing) bool {
	if !validCoordinate(p.Latitude, p.Longitude) {
		log.Printf("bad coordinate - lat: %f or lon: %f", p.Latitude, p.Longitude)
		return false
	}
	if !validPrice(p.Price) {
		log.Printf("bad price val: %f", p.Price)
		return false
	}
	if !validBedrooms(p.Bedrooms) {
		log.Printf("bad bedrooms val: %d", p.Bedrooms)
		return false
	}
	if !validBathrooms(p.Bathrooms) {
		log.Printf("bad bathrooms val: %d", p.Bathrooms)
		return false
	}
	return true
}

func (plP PropProcessor) addToDB(p PropListing) error {
	newProperty := NewProperty(p.Latitude, p.Longitude, p.Price, p.Bedrooms, p.Bathrooms)

	err := plP.DB.Debug().Create(newProperty).Error
	if err != nil {
		log.Printf("PropProcessor unable to insert property: (prop: %v, err: %v)", newProperty, err)
		return errors.Wrap(err, "PropProcessor couldn't insert property")
	}
	return nil
}

// createTransaction is a helper function which takes TransactionRequest object and returns pointer instance of domain.Transaction
func (plP PropProcessor) getCandidateReqs(p PropListing) ([]ReqWithDistance, ReqMargins, error) {
	requirements := []ReqWithDistance{}
	distanceRange := float32(10) // distance threshold in miles
	rMargins := plP.getReqMargins(p, distanceRange)

	queryString := plP.getQueryString()

	err := plP.DB.Debug().
		Raw(queryString, p.Latitude, p.Latitude, p.Longitude, EarthRadius,
			rMargins.MinLat, rMargins.MaxLat, rMargins.MinLon, rMargins.MaxLon,
			distanceRange, p.Price, rMargins.MinPrice, rMargins.MaxPrice, rMargins.MinPrice, rMargins.MaxPrice,
			p.Bedrooms, rMargins.MinBeds, rMargins.MaxBeds, rMargins.MinBeds, rMargins.MaxBeds,
			p.Bathrooms, rMargins.MinBaths, rMargins.MaxBaths, rMargins.MinBaths, rMargins.MaxBaths).
		Scan(&requirements).Error
	if err != nil {
		log.Printf("PropProcessor couldn't getCandidateReqs for: (property: %v, err: %v)", p, err)
		return requirements, rMargins, errors.Wrap(err, "PropProcessor couldn't getCandidateReqs")
	}

	return requirements, rMargins, nil
}

func (plP PropProcessor) getQueryString() string {
	selectBaseClause := "SELECT requirement_id, latitude, longitude, "
	selectDistanceClause := "acos(sin(radians(latitude))*sin(radians(?)) + cos(radiand(latitude))*cos(radians(?))*cos(radians(?) - radians(longitude)) ) * ? as distance, "
	selectRestClause := "min_budget, max_budget, min_bedrooms, max_bedrooms, min_bathrooms, max_bathrooms "

	fromClause := "FROM requirements "

	latCondition := "latitude BETWEEN ? AND ? "
	lonCondition := "AND longitude BETWEEN ? AND ? "
	distCondition := "AND distance <= ? "
	priceCondition := "AND ((? BETWEEN min_budget AND max_budget) OR (min_budget BETWEEN ? AND ?) OR (max_budget BETWEEN ? AND ?) "
	bedsCondtion := "AND ((? BETWEEN min_bedrooms AND max_bedrooms) OR (min_bedrooms BETWEEN ? AND ?) OR (max_bedrooms BETWEEN ? AND ?) "
	bathsCondition := "AND ((? BETWEEN min_bathrooms AND max_bathrooms) OR (min_bathrooms BETWEEN ? AND ?) OR (max_bathrooms BETWEEN ? AND ?)"

	return selectBaseClause + selectDistanceClause + selectRestClause +
		fromClause + "Where " + latCondition + lonCondition + distCondition +
		priceCondition + bedsCondtion + bathsCondition
}

func (plP PropProcessor) getReqMargins(p PropListing, distanceRange float32) ReqMargins {
	minLat, maxLat := GetMinMaxLat(p.Latitude, distanceRange)
	minLon, maxLon := GetMinMaxLon(p.Latitude, p.Longitude, distanceRange)
	minPrice, maxPrice := plP.getMinMaxPrice(p.Price)
	minBeds, maxBeds := plP.getMinMaxBedrooms(p.Bedrooms)
	minBaths, maxBaths := plP.getMinMaxBathrooms(p.Bathrooms)

	return NewReqMargins(minLat, maxLat, minLon, maxLon, minPrice, maxPrice, minBeds, maxBeds, minBaths, maxBaths)
}

func (plP PropProcessor) getMinMaxPrice(price float32) (float32, float32) {
	return MaxF((price - (0.25 * price)), 1.0), MaxF((price + (0.25 * price)), 1.25)
}

func (plP PropProcessor) getMinMaxBedrooms(bedrooms uint16) (uint16, uint16) {
	return Max(bedrooms-2, 1), Max(bedrooms+2, 3)
}

func (plP PropProcessor) getMinMaxBathrooms(bathrooms uint16) (uint16, uint16) {
	return Max(bathrooms-2, 1), Max(bathrooms+2, 3)
}
