package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	// "net/http"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

//Neo4jConfiguration holds the configuration for connecting to the DB
type Neo4jConfiguration struct {
	URL      string
	Username string
	Password string
	Database string
}

type GPS struct {
	DeviceId  string  `json:"deviceId"`
	Timestamp int     `json:"timestamp"`
	Lon       float32 `json:"lon"`
	Lat       float32 `json:"lat"`
}

type Stop struct {
	Name string
	Lon  float32
	Lat  float32
}

type Route struct {
	Name string
}

type Bus struct {
	Name string
	// current location
	Timestamp int
	Lon       float32
	Lat       float32
}

type Garage struct {
	Name string
	Lon  float32
	Lat  float32
}

func addStopInTxFunc(driver neo4j.Driver, stop Stop) error {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run("MERGE (s:Stop {name:$name, lon: $lon, lat: $lat})", map[string]interface{}{"name": stop.Name, "lon": stop.Lon, "lat": stop.Lat})
		if err != nil {
			return nil, err
		}

		return result.Consume()
	})

	return err
}

func addStopsAsRoute(driver neo4j.Driver, route Route) (int, error) {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	tx := ""
	if route.Name == "Route A" {
		tx = "MATCH (s:Stop) WHERE s.name='Visnelik' OR s.name='MEB'  OR s.name='OGU IIBF' RETURN s.name AS name"
	} else if route.Name == "Route B" {
		tx = "MATCH (s:Stop) WHERE s.name='Ogretmen Evi' OR s.name='MEB'  OR s.name='OGU IIBF' RETURN s.name AS name"
	} else if route.Name == "Route C" {
		tx = "MATCH (s:Stop) WHERE s.name='OGU IIBF' OR s.name='Buyukdere' OR s.name='Goztepe' OR s.name='Odunpazari' RETURN s.name AS name"
	}

	stops, err := neo4j.Collect(session.Run(tx, nil))
	fmt.Println(stops)

	if err != nil {
		return 0, err
	}

	stopCount := 0
	for _, stop := range stops {
		_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
			return tx.Run("MATCH (s:Stop {name: $stop_name}) "+
				"MERGE (r:Route {name: $route_name}) "+
				"MERGE (s)-[:HAS]->(r)", map[string]interface{}{"stop_name": stop.Values[0], "route_name": route.Name})
		})
		if err != nil {
			return 0, err
		}

		stopCount++
	}

	return stopCount, nil
}

func addRoutesAsBus(driver neo4j.Driver, bus Bus) (int, error) {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	tx := ""
	if bus.Name == "Bus 1" {
		tx = "MATCH (r:Route) WHERE r.name='Route A'  RETURN r.name AS name"
	} else if bus.Name == "Bus 2" {
		tx = "MATCH (r:Route) WHERE r.name='Route B'  RETURN r.name AS name"
	} else if bus.Name == "Bus 3" || bus.Name == "Bus 4" {
		tx = "MATCH (r:Route) WHERE r.name='Route C'  RETURN r.name AS name"
	}

	routes, err := neo4j.Collect(session.Run(tx, nil))
	fmt.Println(routes)

	if err != nil {
		return 0, err
	}

	routeCount := 0
	for _, route := range routes {
		_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
			return tx.Run("MATCH (r:Route {name: $route_name}) "+
				"MERGE (b:Bus {name: $bus_name, timestamp: $bus_ts, lon: $bus_lon, lat: $bus_lat }) "+
				"MERGE (r)-[:HAS]->(b)", map[string]interface{}{"route_name": route.Values[0], "bus_name": bus.Name, "bus_ts": bus.Timestamp, "bus_lon": bus.Lon, "bus_lat": bus.Lat})
		})
		if err != nil {
			return 0, err
		}

		routeCount++
	}

	return routeCount, nil
}

func addLocationToBus(driver neo4j.Driver, busName string, timestamp int, lon float32, lat float32) (int, error) {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	fmt.Println("----------------------------")
	fmt.Println("----Add Location for Bus----")
	fmt.Println("----------------------------")

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		return tx.Run("MATCH (b:Bus {name: $bus_name}) "+
			"SET b.timestamp = $bus_ts, b.lon = $bus_lon, b.lat = $bus_lat",
			map[string]interface{}{"bus_name": busName, "bus_ts": timestamp, "bus_lon": lon, "bus_lat": lat})
	})

	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		return tx.Run("MATCH (b:Bus {name: $bus_name}) "+
			"MERGE (l:Location {timestamp: $bus_ts, lon: $bus_lon, lat: $bus_lat }) "+
			"MERGE (b)-[:LOCATED]->(l)", map[string]interface{}{"bus_name": busName, "bus_ts": timestamp, "bus_lon": lon, "bus_lat": lat})
	})

	if err != nil {
		return 0, err
	}

	return 1, nil
}

func addBusAsGarage(driver neo4j.Driver, garage Garage) (int, error) {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	tx := ""
	if garage.Name == "Garage 1" {
		tx = "MATCH (b:Bus) WHERE b.name='Bus 1' OR b.name='Bus 2' OR b.name='Bus 3' OR b.name='Bus 4' RETURN b.name AS name"
	}

	buses, err := neo4j.Collect(session.Run(tx, nil))
	fmt.Println(buses)

	if err != nil {
		return 0, err
	}

	busCount := 0
	for _, bus := range buses {
		_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
			return tx.Run("MATCH (b:Bus {name: $bus_name}) "+
				"MERGE (g:Garage {name: $garage_name, lon: $garage_lon, lat: $garage_lat }) "+
				"MERGE (b)-[:HAS]->(g)", map[string]interface{}{"bus_name": bus.Values[0], "garage_name": garage.Name, "garage_lon": garage.Lon, "garage_lat": garage.Lat})
		})

		if err != nil {
			return 0, err
		}

		busCount++
	}

	return busCount, nil
}

func initialize(driver neo4j.Driver) {

	// add stop node
	stop1 := Stop{"OGU IIBF", 39.752998, 30.486864}
	stop2 := Stop{"Buyukdere", 39.752998, 30.486864}
	stop3 := Stop{"Goztepe", 39.752998, 30.486864}
	stop4 := Stop{"MEB", 39.756881, 30.493327}
	stop5 := Stop{"Visnelik", 39.764613, 30.499905}
	stop6 := Stop{"Ogretmen Evi", 39.764992, 30.511165}
	stop7 := Stop{"Odunpazari", 39.764992, 30.511165}

	addStopInTxFunc(driver, stop1)
	addStopInTxFunc(driver, stop2)
	addStopInTxFunc(driver, stop3)
	addStopInTxFunc(driver, stop4)
	addStopInTxFunc(driver, stop5)
	addStopInTxFunc(driver, stop6)
	addStopInTxFunc(driver, stop7)

	// add route node
	routeA := Route{"Route A"}
	routeB := Route{"Route B"}
	routeC := Route{"Route C"}

	rA, errA := addStopsAsRoute(driver, routeA)
	fmt.Println(rA)
	rB, errB := addStopsAsRoute(driver, routeB)
	fmt.Println(rB)
	rC, errC := addStopsAsRoute(driver, routeC)
	fmt.Println(rC)
	if errA != nil || errB != nil || errC != nil {
		return
	}

	// add bus node
	bus1 := Bus{"Bus 1", 1633683332, 30.502349, 39.767360}
	bus2 := Bus{"Bus 2", 1633683332, 39.751049, 30.477519}
	bus3 := Bus{"Bus 3", 1633683332, 39.751049, 30.477519}
	bus4 := Bus{"Bus 4", 1633683332, 39.751049, 30.477519}

	b1, err1 := addRoutesAsBus(driver, bus1)
	fmt.Println(b1)
	b2, err2 := addRoutesAsBus(driver, bus2)
	fmt.Println(b2)
	b3, err3 := addRoutesAsBus(driver, bus3)
	fmt.Println(b3)
	b4, err4 := addRoutesAsBus(driver, bus4)
	fmt.Println(b4)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return
	}

	// add garage node
	garage1 := Garage{"Garage 1", 39.751049, 30.477519}
	g1, err5 := addBusAsGarage(driver, garage1)
	fmt.Println(g1)
	if err5 != nil {
		return
	}
}

func main() {
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)
	initialize(driver)

	// kafka consumer
	log.Println("NewConsumer")
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka:19092",
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		panic(err)
	}

	topic := "gps_data"
	consumer.SubscribeTopics([]string{topic, "^aRegex.*[Tt]opic"}, nil)

	for {
		msg, err := consumer.ReadMessage(-1)

		if err == nil {
			data := GPS{}
			json.Unmarshal([]byte(msg.Value), &data)
			fmt.Printf("DeviceId: %s\n", data.DeviceId)
			fmt.Printf("Timestamp: %d\n", data.Timestamp)
			fmt.Printf("Lon: %g\n", data.Lon)
			fmt.Printf("Lat: %g\n", data.Lat)

			addLocationToBus(driver, data.DeviceId, data.Timestamp, data.Lon, data.Lat)

		} else {
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	consumer.Close()
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
