package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

//Neo4jConfiguration holds the configuration for connecting to the DB
type Neo4jConfiguration struct {
	URL      string
	Username string
	Password string
	Database string
}

// type Bus struct {
// 	Name string
// 	// current location
// 	Timestamp int
// 	Lon       string
// 	Lat       string
// }

func busAnomalyDetection(driver neo4j.Driver, busName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		session := driver.NewSession(neo4j.SessionConfig{
			AccessMode:   neo4j.AccessModeRead,
			DatabaseName: "neo4j",
		})
		defer unsafeClose(session)

		routeName := ""
		if busName == "Bus 1" {
			routeName = "Route A"
		} else if busName == "Bus 2" {
			routeName = "Route B"
		} else if busName == "Bus 3" || busName == "Bus 4" {
			routeName = "Route C"
		}

		var anomaly []string

		// bus, err := neo4j.Collect(session.Run("MATCH (b:Bus) WHERE b.name=$bus_name RETURN b.lon, b.lat", map[string]interface{}{"bus_name": busName}))

		// ad, err := neo4j.Collect(session.Run("MATCH (b:Bus) WHERE b.name=$bus_name"+
		// 	"MATCH (:Route{name:$route_name})-[r:LOCATED]-(l)"+
		// 	"WITH point({longitude: b.lon, latitude: b.lat}) as p1, point({longitude: l.lon, latitude: l.lat}) as p2"+
		// 	"RETURN round(distance(p1,p2)) as dist", map[string]interface{}{"bus_name": busName, "route_name": routeName}))

		fmt.Println(routeName)

		adValues, err := neo4j.Collect(session.Run(
			"MATCH (b:Bus) WHERE b.name='Bus 1'"+
				"MATCH (:Route{name:'Route A'})-[r:LOCATED]-(l)"+
				"WITH point({longitude: b.lon, latitude: b.lat}) as p1, point({longitude: l.lon, latitude: l.lat}) as p2"+
				"RETURN round(distance(p1,p2)) as dist",
			map[string]interface{}{}))

		fmt.Println(adValues)

		// for _, ad_val := range ad {
		// anomaly = append(anomaly, ad_val.Values[0].(string))
		// if ad_val.Values[0] < 100 {
		// 	for _, bus_val := range bus {

		// 		anomaly = append(anomaly, busName)
		// 		anomaly = append(anomaly, bus_val.Values[0].(string))
		// 		anomaly = append(anomaly, bus_val.Values[1].(string))
		// 		anomaly = append(anomaly, ad_val.Values[0].(string))
		// 	}
		// }
		// }

		// bus, err := neo4j.Collect(session.Run("MATCH (b:Bus) WHERE b.name=$bus_name RETURN b.lon, b.lat", map[string]interface{}{"bus_name": busName}))
		// //Bus 1 current location - (Route a all location) < 100
		// if err != nil {
		// 	fmt.Println(err)
		// }
		// for _, val := range bus {
		// 	fmt.Println("0: ", val.Values[0])
		// 	fmt.Println("1: ", val.Values[1])
		// 	list = append(list, val.Values[0].(string))
		// 	list = append(list, val.Values[1].(string))
		// }

		// routeAllLocation, err := neo4j.Collect(session.Run("MATCH (:Route{name:$route_name})-[r:LOCATED]-(b) RETURN b.lon, b.lat", map[string]interface{}{"route_name": routeName}))

		// for _, val := range routeAllLocation {

		// 	fmt.Println("route 0: ", val.Values[0])
		// 	// fmt.Println("route 1: ", val.Values[1])

		// }

		// fmt.Println("list 0: ", list[0])
		// fmt.Println("list 1: ", list[1])

		busJSON, err := json.Marshal(anomaly)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(busJSON)
	}
}

func main() {
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)

	log.Println("busAnomalyDetection")
	http.HandleFunc("/", busAnomalyDetection(driver, "Bus 1"))
	log.Println("-------------------------------------")
	http.ListenAndServe(":8080", nil)

}

//newDrive is a method for Neo4jConfiguration to return a connection to the DB
func (nc *Neo4jConfiguration) newDriver() (neo4j.Driver, error) {
	return neo4j.NewDriver(nc.URL, neo4j.BasicAuth(nc.Username, nc.Password, ""))
}

func parseConfiguration() *Neo4jConfiguration {
	return &Neo4jConfiguration{
		URL:      "neo4j://neo4j:7687",
		Username: "neo4j",
		Password: "streams",
	}
}

func unsafeClose(closeable io.Closer) {
	if err := closeable.Close(); err != nil {
		log.Fatal(fmt.Errorf("could not close resource: %w", err))
	}
}
