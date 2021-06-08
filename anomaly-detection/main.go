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
	BusName     string
	Timestamp   int64
	Lon         float64
	Lat         float64
	MinDistance int64
}

func getAnomaly(driver neo4j.Driver) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		session := driver.NewSession(neo4j.SessionConfig{
			AccessMode:   neo4j.AccessModeRead,
			DatabaseName: "neo4j",
		})
		defer unsafeClose(session)
		anomalyDetection(driver)

		cmd := "MATCH (a:Anomaly) RETURN a.name, a.timestamp, a.lon, a.lat, a.minDist"
		ad, err := neo4j.Collect(session.Run(cmd, nil))
		anomaly := []Anomaly{}

		for _, val := range ad {
			anomaly = append(anomaly, Anomaly{
				BusName:     val.Values[0].(string),
				Timestamp:   val.Values[1].(int64),
				Lon:         val.Values[2].(float64),
				Lat:         val.Values[3].(float64),
				MinDistance: val.Values[4].(int64),
			})
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

func anomalyDetection(driver neo4j.Driver) error {
	session := driver.NewSession(neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: "neo4j",
	})
	defer unsafeClose(session)

	for i := 0; i < 4; i++ {

		fmt.Println("---Anomaly Detection---")

		cmd := ""
		routeName := ""
		busName := ""

		if i == 0 {
			busName = "Bus 1"
			routeName = "Route A"
		} else if i == 1 {
			busName = "Bus 2"
			routeName = "Route B"
		} else if i == 2 {
			busName = "Bus 3"
			routeName = "Route C"
		} else if i == 3 {
			busName = "Bus 4"
			routeName = "Route C"
		}

		cmd = "MATCH (b:Bus) WHERE b.name = '" + busName + "' MATCH (:Route{name: '" + routeName + "'})-[r:LOCATED]-(l) " +
			"RETURN round(distance(point({longitude: b.lon, latitude: b.lat}),point({longitude: l.lon, latitude: l.lat}))) as dist"

		adValues, _ := neo4j.Collect(session.Run(cmd, nil))

		var arr []int
		for _, ad_val := range adValues {
			distance := int(ad_val.Values[0].(float64))
			arr = append(arr, distance)
		}

		fmt.Println(arr)
		bus, err := neo4j.Collect(session.Run("MATCH (b:Bus) WHERE b.name=$bus_name RETURN b.timestamp, b.lon, b.lat",
			map[string]interface{}{"bus_name": busName}))

		// that's anomaly
		if !contains(arr, 100) {
			for _, val := range bus {

				cmd := fmt.Sprint("MERGE (a:Anomaly {name: '", busName, "', timestamp:", val.Values[0].(int64), ", lon: ",
					val.Values[1].(float64), ", lat: ", val.Values[2].(float64), ", minDist: ", minDistance(arr), "})")

				ad, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
					return tx.Run(cmd, nil)
				})

				if err != nil {
					return err
				}
				fmt.Println(ad)
			}
		}

		if err != nil {
			return err
		}

		fmt.Println("---End Anomaly Detection---")

	}
	return nil
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a < e {
			return true
		}
	}
	return false
}

func minDistance(values []int) int {
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func main() {
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)

	anomalyDetection(driver)

	http.HandleFunc("/", getAnomaly(driver))
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
