package main

import "log"

func main() {
	// step 1: read configs

	// step 2: add dependecies (Dependency Injections)
	reqProcessor, propProcessor := dependencgInjections()

	// Simulate using the above usecase processors to do something
	log.Printf("Got %v and %v", reqProcessor, propProcessor)

	// step 3: add routes and attach controllers to web app and start server
}

// dependencgInjections is like a dependency injector which initiates all different
// infrastructre objects and instances and adds its to the App instance which can
// be passed anywhere down the dependency tree
func dependencgInjections() (ReqProcessor, PropProcessor) {
	// get a single DB connection/pool
	db, err := NewDBClient()
	if err != nil {
		panic("Unable to get a DB connection")
	}

	rAlgo := NewReqMatchingAlgo()
	pAlgo := NewPropMatchingAlgo()

	reqProcessor := NewReqProcessor(db, rAlgo)
	propProcessor := NewPropProcessor(db, pAlgo)

	// Here we use r and p to perform the usecasaes
	// API handler/cotrollers will have access to r and p to perform the usecases
	return reqProcessor, propProcessor
}
