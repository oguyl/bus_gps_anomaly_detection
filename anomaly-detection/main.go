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

type Anomaly struct {
	BusName   string
	Timestamp int64
	Lon       float64
	Lat       float64
}

func busAnomalyDetection(driver neo4j.Driver, busName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		session := driver.NewSession(neo4j.SessionConfig{
			AccessMode:   neo4j.AccessModeRead,
			DatabaseName: "neo4j",
		})
		defer unsafeClose(session)

		bus, err := neo4j.Collect(session.Run("MATCH (b:Bus) WHERE b.name=$bus_name RETURN b.timestamp, b.lon, b.lat", map[string]interface{}{"bus_name": busName}))

		routeName := ""
		tx := ""
		if busName == "Bus 1" {
			routeName = "Route A"
		} else if busName == "Bus 2" {
			routeName = "Route B"
		} else if busName == "Bus 3" || busName == "Bus 4" {
			routeName = "Route C"
		}

		tx = "MATCH (b:Bus) WHERE b.name = '" + busName + "' MATCH (:Route{name: '" + routeName + "'})-[r:LOCATED]-(l) " +
			"RETURN round(distance(point({longitude: b.lon, latitude: b.lat}),point({longitude: l.lon, latitude: l.lat}))) as dist"

		anomaly := []Anomaly{}
		adValues, _ := neo4j.Collect(session.Run(tx, nil))

		var arr []int
		for _, ad_val := range adValues {
			// fmt.Printf("\nThe type of ad_val.Values[0] is : %T", ad_val.Values[0])
			// fmt.Printf("\nThe type of ad_val is : %T", ad_val)
			distance := int(ad_val.Values[0].(float64))
			arr = append(arr, distance)
		}

		fmt.Println(arr)

		// for _, val := range bus {
		// 	fmt.Println(val.Values[0].(int64))
		// 	fmt.Println(val.Values[1].(float64))
		// 	fmt.Println(val.Values[2].(float64))
		// }

		if !contains(arr, 100) {
			for _, val := range bus {
				n := Anomaly{BusName: busName, Timestamp: val.Values[0].(int64), Lon: val.Values[1].(float64), Lat: val.Values[2].(float64)}
				anomaly = append(anomaly, n)
			}
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		busJSON, err := json.Marshal(anomaly)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(busJSON)
	}
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a < e {
			return true
		}
	}
	return false
}

func main() {
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)

	http.HandleFunc("/", busAnomalyDetection(driver, "Bus 1"))
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
