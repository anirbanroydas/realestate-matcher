package main

type MatchedRequirement struct {
	Requirement
	MatchScore float32
}

func NewMatchedRequirement(r Requirement, score float32) MatchedRequirement {
	return MatchedRequirement{
		Requirement: r,
		MatchScore:  score,
	}
}

type PropMatchingAlgo interface {
	Match(PropListing, []ReqWithDistance, ReqMargins) []MatchedRequirement
}

type PropMatchAlgoV1 struct{}

func NewPropMatchingAlgo() PropMatchAlgoV1 {
	return PropMatchAlgoV1{}
}

func (a PropMatchAlgoV1) Match(p PropListing, requirements []ReqWithDistance, rMargins ReqMargins) []MatchedRequirement {
	matchedReqs := []MatchedRequirement{}
	scoring := make(chan bool)
	defer close(scoring)

	scores := a.createReqScores(requirements)

	// Run the 4 mathching tasks in goroutines to run them concurrently
	go a.distanceMatching(p.Latitude, p.Longitude, scores, scoring)
	go a.budgetMatching(p.Price, requirements, scores, rMargins, scoring)
	go a.bedroomsMatching(p.Bedrooms, requirements, scores, rMargins, scoring)
	go a.bathroomsMatching(p.Bathrooms, requirements, scores, rMargins, scoring)

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
		matchedReqs = append(matchedReqs, NewMatchedRequirement(requirements[scores[i].Index].Requirement, scores[i].Total))
	}
	return matchedReqs
}

func (a PropMatchAlgoV1) createReqScores(r []ReqWithDistance) []Score {
	scores := make([]Score, len(r))
	for i, _ := range r {
		scores[i] = NewScore(i, r[i].Distance)
	}
	return scores
}

func (a PropMatchAlgoV1) distanceMatching(lat, lon float32, scores []Score, scoring chan bool) {
	// base distance and maxDistance in miles
	baseDistance := float32(2)
	maxDistance := float32(10)

	for i, _ := range scores {
		scores[i].DistanceScore = GetDistanceScore(scores[i].Distance, baseDistance, maxDistance)
	}
	scoring <- true
}

func (a PropMatchAlgoV1) budgetMatching(price float32, r []ReqWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		scores[i].BudgetScore = GetBudgetScore(r[i].MinBudget, r[i].MaxBudget, price, rMargins.MinPrice, rMargins.MaxPrice)
	}
	scoring <- true
}

func (a PropMatchAlgoV1) bedroomsMatching(bedrooms uint16, r []ReqWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		scores[i].BedroomScore = GetBedroomScore(r[i].MinBedrooms, r[i].MaxBedrooms, bedrooms, rMargins.MinBeds, rMargins.MaxBeds)
	}
	scoring <- true
}

func (a PropMatchAlgoV1) bathroomsMatching(bathrooms uint16, r []ReqWithDistance, scores []Score, rMargins ReqMargins, scoring chan bool) {
	for i, _ := range scores {
		// since algor for bathrooms matching is similar to batrhooms matching, using the same GetBedroomScore function
		scores[i].BathroomScore = GetBedroomScore(r[i].MinBathrooms, r[i].MaxBathrooms, bathrooms, rMargins.MinBaths, rMargins.MaxBaths)
	}
	scoring <- true
}
