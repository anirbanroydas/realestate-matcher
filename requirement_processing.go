// Package main containces all method for performing allthe use cases of the application
// This package does not dpened on any other package except the domain. This package contains
// methods which can run indpendent of choice of db, web framework etc
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const (
	EarthRadius float32 = 3959.0
)

type PropRequirement struct {
	Latitude     float32
	Longitude    float32
	MinBudget    float32
	MaxBudget    float32
	MinBedrooms  uint16
	MaxBedrooms  uint16
	MinBathrooms uint16
	MaxBathrooms uint16
}

type PropWithDistance struct {
	Property
	Distance float32
}

type ReqMargins struct {
	MinLat   float32
	MaxLat   float32
	MinLon   float32
	MaxLon   float32
	MinPrice float32
	MaxPrice float32
	MinBeds  uint16
	MaxBeds  uint16
	MinBaths uint16
	MaxBaths uint16
}

func NewReqMargins(minLat, maxLat, minLon, maxLon, minPrice, maxPrice float32, minBeds, maxBeds, minBaths, maxBaths uint16) ReqMargins {
	return ReqMargins{
		MinLat:   minLat,
		MaxLat:   maxLat,
		MinLon:   minLon,
		MaxLon:   maxLon,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		MinBeds:  minBeds,
		MaxBeds:  maxBeds,
		MinBaths: minBaths,
		MaxBaths: maxBaths,
	}
}

type ReqProcessor struct {
	DB             *gorm.DB
	MatchAlgorithm ReqMatchingAlgo
}

func NewReqProcessor(db *gorm.DB, rAlgo ReqMatchingAlgo) ReqProcessor {
	return ReqProcessor{
		DB:             db,
		MatchAlgorithm: rAlgo,
	}
}

func (rP ReqProcessor) GetMatchingProps(p PropRequirement) ([]MatchedProperty, error) {
	var err error
	var candidateProps []PropWithDistance
	var matchingProps []MatchedProperty
	var rMargins ReqMargins

	// step 0:  validate the Property Requirement Request
	isValid := rP.validate(p)
	if !isValid {
		return matchingProps, errors.Wrap(err, "ReqProcessor couldn't validate")
	}

	// step 1: Add requirement to database
	err = rP.addToDB(p)
	if err != nil {
		return matchingProps, errors.Wrap(err, "ReqProcessor couldn't addToDB")
	}

	// step 2: Base Filtering - filter out a certain set of property listings first based on parameters which gives a set of possible candidate property listings
	candidateProps, rMargins, err = rP.getCandidateProps(p)
	if err != nil {
		return matchingProps, errors.Wrap(err, "ReqProcessor couldn't getCandidateProps")
	}

	// step 3: Run algorithm on candidate properties and get a result set of matching properties
	matchingProps = rP.MatchAlgorithm.Match(p, candidateProps, rMargins)
	return matchingProps, nil
}

func (rP ReqProcessor) validate(p PropRequirement) bool {
	if !validCoordinate(p.Latitude, p.Longitude) {
		log.Printf("bad coordinate - lat: %f or lon: %f", p.Latitude, p.Longitude)
		return false
	}
	if !validBudget(p.MinBudget, p.MaxBudget) {
		log.Printf("bad budget range min: %f - max: %f", p.MinBudget, p.MaxBudget)
		return false
	}
	if !validBedroomsRange(p.MinBedrooms, p.MaxBedrooms) {
		log.Printf("bad bedrooms range min: %d - max: %d", p.MinBedrooms, p.MaxBedrooms)
		return false
	}
	if !validBathroomsRange(p.MinBathrooms, p.MaxBathrooms) {
		log.Printf("bad bathrooms range min: %d - max: %d", p.MinBathrooms, p.MaxBathrooms)
		return false
	}
	return true
}

func (rP ReqProcessor) addToDB(p PropRequirement) error {
	req := NewRequirement(p.Latitude, p.Longitude, p.MinBudget, p.MaxBudget, p.MinBedrooms, p.MaxBedrooms, p.MinBathrooms, p.MaxBathrooms)

	err := rP.DB.Debug().Create(req).Error
	if err != nil {
		log.Printf("ReqProcessor unable to insert requirement: (req: %v, err: %v)", req, err)
		return errors.Wrap(err, fmt.Sprintf("ReqProcessor couldn't insert requirement: %v", req))
	}
	return nil
}

// createTransaction is a helper function which takes TransactionRequest object and returns pointer instance of domain.Transaction
func (rP ReqProcessor) getCandidateProps(p PropRequirement) ([]PropWithDistance, ReqMargins, error) {
	properties := []PropWithDistance{}
	distanceRange := float32(10) // distance threshold in miles
	rMargins := rP.getReqMargins(p, distanceRange)

	queryString := rP.getQueryString()

	err := rP.DB.Debug().
		Raw(queryString, p.Latitude, p.Latitude, p.Longitude, EarthRadius,
			rMargins.MinLat, rMargins.MaxLat, rMargins.MinLon, rMargins.MaxLon,
			distanceRange, rMargins.MinPrice, rMargins.MaxPrice, rMargins.MinBeds,
			rMargins.MaxBeds, rMargins.MinBaths, rMargins.MaxBaths).
		Scan(&properties).Error
	if err != nil {
		log.Printf("ReqProcessor couldn't getCandidateProps for: (requirement: %v, err: %v)", p, err)
		return properties, rMargins, errors.Wrap(err, "ReqProcessor couldn't getCandidateProps")
	}

	return properties, rMargins, nil
}

func (rP ReqProcessor) getQueryString() string {
	selectBaseClause := "SELECT property_id, latitude, longitude, "
	selectDistanceClause := "acos(sin(rs(latitude))*sin(rs(?)) + cos(rd(latitude))*cos(rs(?))*cos(rs(?) - rs(longitude)) ) * ? as distance, "
	selectRestClause := "price, bedrooms, bathrooms "

	fromClause := "FROM properties "

	latCondition := "latitude BETWEEN ? AND ? "
	lonCondition := "AND longitude BETWEEN ? AND ? "
	distCondition := "AND distance <= ? "
	priceCondition := "AND price BETWEEN ? AND ? "
	bedsCondtion := "AND bedrooms BETWEEN ? AND ? "
	bathsCondition := "AND bathrooms BETWEEN ? AND ?"

	return selectBaseClause + selectDistanceClause + selectRestClause +
		fromClause + "Where " + latCondition + lonCondition + distCondition +
		priceCondition + bedsCondtion + bathsCondition
}

func (rP ReqProcessor) getReqMargins(p PropRequirement, distanceRange float32) ReqMargins {
	minLat, maxLat := GetMinMaxLat(p.Latitude, distanceRange)
	minLon, maxLon := GetMinMaxLon(p.Latitude, p.Longitude, distanceRange)
	minPrice, maxPrice := rP.getMinMaxPrice(p.MinBudget, p.MaxBudget)
	minBeds, maxBeds := rP.getMinMaxBedrooms(p.MinBedrooms, p.MaxBedrooms)
	minBaths, maxBaths := rP.getMinMaxBathrooms(p.MinBathrooms, p.MaxBathrooms)

	return NewReqMargins(minLat, maxLat, minLon, maxLon, minPrice, maxPrice, minBeds, maxBeds, minBaths, maxBaths)
}

func (rP ReqProcessor) getMinMaxPrice(minBudget, maxBudget float32) (float32, float32) {
	if minBudget > 0 && maxBudget > 0 {
		// if both minBudet and maxBudget given
		return MaxF((minBudget - (0.25 * minBudget)), 1.0), MaxF((maxBudget + (0.25 * maxBudget)), 1.25)
	}
	if minBudget > 0 {
		// if only minBudget given
		return MaxF((minBudget - (0.25 * minBudget)), 1.0), MaxF((minBudget + (0.25 * minBudget)), 1.25)
	}
	// if only maxBudget given
	return MaxF((maxBudget - (0.25 * maxBudget)), 1.0), MaxF((maxBudget + (0.25 * maxBudget)), 1.25)
}

func (rP ReqProcessor) getMinMaxBedrooms(minBeds, maxBeds uint16) (uint16, uint16) {
	if minBeds > 0 && maxBeds > 0 {
		// if both maxBeds and maxBeds given
		return Max(minBeds-2, 1), Max(maxBeds+2, 3)
	}
	if minBeds > 0 {
		// if only minBeds given
		return Max(minBeds-2, 1), Max(minBeds+2, 3)
	}
	// if only maxBeds given
	return Max(maxBeds-2, 1), Max(maxBeds+2, 3)
}

func (rP ReqProcessor) getMinMaxBathrooms(minBaths, maxBaths uint16) (uint16, uint16) {
	// using hte smae MinMaxBedrooms functiion as it has the same functionality
	return rP.getMinMaxBedrooms(minBaths, maxBaths)
}

//////////////////////////////////////////////////////////
////			Common Exported Funtions 			/////
//////////////////////////////////////////////////////////

func GetMinMaxLat(lat, distanceRange float32) (float32, float32) {
	degree := RadToDeg(float64(EarthRadius / distanceRange))
	return lat - float32(degree), lat + float32(degree)
}

func GetMinMaxLon(lat, lon, distanceRange float32) (float32, float32) {
	degree := RadToDeg(math.Asin(float64(EarthRadius/distanceRange)) / math.Cos(DegToRad(float64(lat))))
	return lon - float32(degree), lon + float32(degree)
}

func MaxF(x, y float32) float32 {
	if x >= y {
		return x
	}
	return y
}

func Max(x, y uint16) uint16 {
	if x >= y {
		return x
	}
	return y
}

func DegToRad(d float64) float64 {
	return d * math.Pi / 180
}

func RadToDeg(r float64) float64 {
	return (r * 180) / math.Pi
}
