package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jConfiguration struct {
	URL      string
	Username string
	Password string
	Database string
}

type Trkseg struct {
	XMLName xml.Name `xml:"trkseg"`
	Trkseg  []Trkpt  `xml:"trkpt"`
}

type Trkpt struct {
	XMLName xml.Name `xml:"trkpt"`
	Lat     float32  `xml:"lat,attr"`
	Lon     float32  `xml:"lon,attr"`
}

// Xml dosyasından okunan konum verileri Neo4j'ye kaydedilmektedir.
func addLocationToRoute(driver neo4j.Driver, routeName string, lon float32, lat float32) (int, error) {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	fmt.Println("\n------------------------------")
	fmt.Println("----Add Location for Route----")
	fmt.Println("------------------------------\n")

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		return tx.Run("MATCH (r:Route {name: $route_name}) "+
			"MERGE (l:Location {lon: $route_lon, lat: $route_lat }) "+
			"MERGE (r)-[:LOCATED]->(l)", map[string]interface{}{"route_name": routeName, "route_lon": lon, "route_lat": lat})
	})

	if err != nil {
		return 0, err
	}

	return 1, nil
}

func main() {
	configuration := parseConfiguration()
	driver, err := configuration.newDriver()
	if err != nil {
		log.Fatal(err)
	}
	defer unsafeClose(driver)

	parseXML("./xml/ibbf-odunp.xml", driver)
	parseXML("./xml/ibbf-ogevi.xml", driver)
	parseXML("./xml/ibbf-visnelik.xml", driver)
}

// Gelen dosya adına göre rota bilgilerinin olduğu xml dosyası okunup Neo4j'ye kaydedilmektedir.
func parseXML(file string, driver neo4j.Driver) {

	xmlFile, err := os.Open(file)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Successfully Opened" + file)
	}

	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)
	var trkseg Trkseg
	xml.Unmarshal(byteValue, &trkseg)

	routeName := ""
	if file == "./xml/ibbf-visnelik.xml" {
		routeName = "Route A"
	} else if file == "./xml/ibbf-ogevi.xml" {
		routeName = "Route B"
	} else if file == "./xml/ibbf-odunp.xml" {
		routeName = "Route C"
	}

	for i := 0; i < len(trkseg.Trkseg); i++ {
		addLocationToRoute(driver, routeName, trkseg.Trkseg[i].Lon, trkseg.Trkseg[i].Lat)
	}
}

//neo4j bağlantıları ve ilgili konfigürasyon işlemleri yapılır.
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
