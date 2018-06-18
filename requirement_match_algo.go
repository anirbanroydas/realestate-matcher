package main

import (
	"sort"
)

type Score struct {
	Index         int
	Distance      float32
	DistanceScore float32
	BudgetScore   float32
	BedroomScore  float32
	BathroomScore float32
	Total         float32
}

func NewScore(index int, distance float32) Score {
	return Score{
		Index:    index,
		Distance: distance,
	}
}

type MatchedProperty struct {
	Property
	MatchScore float32
}

func NewMatchedProperty(p Property, score float32) MatchedProperty {
	return MatchedProperty{
		Property:   p,
		MatchScore: score,
	}
}

type ReqMatchingAlgo interface {
	Match(PropRequirement, []PropWithDistance, ReqMargins) []MatchedProperty
}

type ReqMatchAlgoV1 struct{}

func NewReqMatchingAlgo() ReqMatchAlgoV1 {
	return ReqMatchAlgoV1{}
}

func (a ReqMatchAlgoV1) Match(p PropRequirement, properties []PropWithDistance, rMargins ReqMargins) []MatchedProperty {
	matchedProps := []MatchedProperty{}
	scoring := make(chan bool)
	defer close(scoring)

	scores := a.createPropScores(properties)

	// Run the 4 mathching tasks in goroutines to run them concurrently
	go a.distanceMatching(p.Latitude, p.Longitude, scores, scoring)
	go a.budgetMatching(p.MinBudget, p.MaxBudget, properties, scores, rMargins, scoring)
	go a.bedroomsMatching(p.MinBedrooms, p.MaxBedrooms, properties, scores, rMargins, scoring)
	go a.bathroomsMatching(p.MinBathrooms, p.MaxBathrooms, properties, scores, rMargins, scoring)

	// read from scoring channel, and wait and finish as soon as 4 of the goroutines finishes
	for i := 0; i < 4; i++ {
		<-scoring
	}

	// Score have been added, now final step, sort them
	SortScores(scores)

	for i, _ := range scores {
		if scores[i].Total < 40.0 {
			continue
		}
		matchedProps = append(matchedProps, NewMatchedProperty(properties[scores[i].Index].Property, scores[i].Total))
	}
	return matchedProps
}

func (a ReqMatchAlgoV1) createPropScores(p []PropWithDistance) []Score {
	scores := make([]Score, len(p))
	for i, _ := range p {
		scores[i] = NewScore(i, p[i].Distance)
	}
	return scores
}

func (a ReqMatchAlgoV1) distanceMatching(lat, lon float32, scores []Score, scoring chan bool) {
	// base distance and maxDistance in miles
	baseDistance := float32(2)
	maxDistance := float32(10)

	for i, _ := range scores {
		scores[i].DistanceScore = GetDistanceScore(scores[i].Distance, baseDistance, maxDistance)
	}
	scoring <- true
}

func (a ReqMatchAlgoV1) budgetMatching(minBudget, maxBudget float32, p []PropWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		scores[i].BudgetScore = GetBudgetScore(minBudget, maxBudget, p[i].Price, rMargins.MinPrice, rMargins.MaxPrice)
	}
	scoring <- true
}

func (a ReqMatchAlgoV1) bedroomsMatching(minBedrooms, maxBedrooms uint16, p []PropWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		scores[i].BedroomScore = GetBedroomScore(minBedrooms, maxBedrooms, p[i].Bedrooms, rMargins.MinBeds, rMargins.MaxBeds)
	}
	scoring <- true
}

func (a ReqMatchAlgoV1) bathroomsMatching(minBathrooms, maxBathrooms uint16, p []PropWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		// since algor for bathrooms matching is similar to batrhooms matching, using the same GetBedroomScore function
		scores[i].BathroomScore = GetBedroomScore(minBathrooms, maxBathrooms, p[i].Bathrooms, rMargins.MinBaths, rMargins.MaxBaths)
	}
	scoring <- true
}

func GetDistanceScore(distance, baseDistance, maxDistance float32) float32 {
	if distance <= baseDistance {
		return float32(30.0)
	}
	return ((maxDistance - distance) / (maxDistance - baseDistance)) * 30.0
}

func GetBudgetScore(minBudget, maxBudget, price, minPrice, maxPrice float32) float32 {
	// case 1: when both minBudget and maxBudther is given
	if minBudget > 0 && maxBudget > 0 {
		return budgetScoreUtil(minBudget, maxBudget, price, minPrice, maxPrice)
	}

	var min10Budget, max10Budget float32
	// case 2: only minBudget given
	if minBudget > 0 {
		min10Budget = MaxF((minBudget - (0.10 * minBudget)), 1.0)
		max10Budget = MaxF((minBudget + (0.10 * minBudget)), 1.1)
	} else {
		// case 3: only maxBudget given
		min10Budget = MaxF((maxBudget - (0.10 * maxBudget)), 1.0)
		max10Budget = MaxF((maxBudget + (0.10 * maxBudget)), 1.1)
	}
	return budgetScoreUtil(min10Budget, max10Budget, price, minPrice, maxPrice)
}

func budgetScoreUtil(minBudget, maxBudget, price, minPrice, maxPrice float32) float32 {
	var weightage float32

	if price >= minBudget && price <= maxBudget {
		// price falls within budget range, 30 marks
		weightage = 1
	} else if price < minBudget {
		// price falls in the minPrice - minBudget range
		weightage = (price - minPrice) / (minBudget - minPrice)
	} else {
		// price falls in the maxBudget - maxPrice range
		weightage = (maxPrice - price) / (maxPrice - maxBudget)
	}
	return weightage * 30
}

func GetBedroomScore(minBedrooms, maxBedrooms, bedrooms, minBeds, maxBeds uint16) float32 {
	// case 1: when both minBedrooms and maxBedrooms is given
	if minBedrooms > 0 && maxBedrooms > 0 {
		return bedroomScoreUtil(minBedrooms, maxBedrooms, bedrooms, minBeds, maxBeds)
	}

	// case 2: only minBedrooms given
	if minBedrooms > 0 {
		return bedroomScoreUtil(minBedrooms, minBedrooms, bedrooms, minBeds, maxBeds)
	} else {
		// case 3: only maxBedrooms given
		return bedroomScoreUtil(maxBedrooms, maxBedrooms, bedrooms, minBeds, maxBeds)
	}
}

func bedroomScoreUtil(minBedrooms, maxBedrooms, bedrooms, minBeds, maxBeds uint16) float32 {
	var weightage float32

	if bedrooms >= minBedrooms && bedrooms <= maxBedrooms {
		// price falls within bedrooms range, 30 marks
		weightage = 1
	} else if bedrooms < minBedrooms {
		// price falls in the minBeds - minBedrooms range
		a := bedrooms - minBedrooms
		b := Max((minBedrooms - minBeds), 1)
		weightage = float32(a) / float32(b)
	} else {
		// price falls in the maxBedrooms - maxBeds range
		c := maxBeds - bedrooms
		d := maxBeds - maxBedrooms
		weightage = float32(c) / float32(d)
	}
	return weightage * 20
}

func SortScores(s []Score) {
	// first add the scores and save in Total attribute
	for i, _ := range s {
		s[i].Total = getTotalScore(s[i])
	}

	// now sort beased on sorting less function
	sort.Slice(s, func(i, j int) bool {
		first := s[i]
		second := s[j]

		if first.Total < second.Total {
			return false // to return result in descending order
		} else if first.Total > second.Total {
			return true
		}
		// in case score are equal, sort based on distance value
		if first.Distance < second.Distance {
			return true // for ascending order
		} else if first.Distance > second.Distance {
			return false
		}
		// in case score are equal and distance both equal, sort based on budget score
		if first.BudgetScore < second.BudgetScore {
			return false // for descending order
		} else if first.BudgetScore > second.BudgetScore {
			return true
		}
		// in case score, distance, budgetScore are equal, sort based on bedrooms score
		if first.BedroomScore < second.BedroomScore {
			return false // for descending order
		} else if first.BedroomScore > second.BedroomScore {
			return true
		}
		// in case all above are equal, sort based on bathrooms score
		if first.BathroomScore < second.BathroomScore {
			return false // for descending order
		} else if first.BathroomScore > second.BathroomScore {
			return true
		}
		return true
	})
}

func getTotalScore(s Score) float32 {
	return s.DistanceScore + s.BudgetScore + s.BedroomScore + s.BathroomScore
}
