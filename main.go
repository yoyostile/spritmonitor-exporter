package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron"
)

var addr 		 = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var username = os.Getenv("SPRITMONITOR_USERNAME")
var password = os.Getenv("SPRITMONITOR_PASSWORD")
var every 	 = flag.String("every", "6h", "Update time")
var myClient = &http.Client{Timeout: 10 * time.Second}
var baseUrl  = "https://api.spritmonitor.de/v1/"

var (
	promVehicleConsumption = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_consumption",
			Help: "Vehicle Consumption",
		},
		[]string{"id", "make", "model"},
	)
	promVehicleTripSum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_trip_sum",
			Help: "Vehicle Trip Sum",
		},
		[]string{"id", "make", "model"},
	)
	promVehicleFuelSum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fuel_sum",
			Help: "Vehicle Fuel Sum ",
		},
		[]string{"id", "make", "model"},
	)
	promFuelingOdometer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fueling_odometer",
			Help: "Vehicle Fueling Odometer",
		},
		[]string{"id", "make", "model", "date", "fuelsortid"},
	)

	promFuelingTrip = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fueling_trip",
			Help: "Vehicle Fueling Trip",
		},
		[]string{"id", "make", "model", "date", "fuelsortid"},
	)

	promFuelingQuantity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fueling_quantity",
			Help: "Vehicle Fueling Quantity",
		},
		[]string{"id", "make", "model", "date", "fuelsortid"},
	)

	promFuelingCost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fueling_cost",
			Help: "Vehicle Fueling Cost",
		},
		[]string{"id", "make", "model", "date", "fuelsortid"},
	)

	promFuelingConsumption = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vehicle_fueling_consumption",
			Help: "Vehicle Fueling Consumption",
		},
		[]string{"id", "make", "model", "date", "fuelsortid"},
	)
)

type Vehicle struct {
	ID                int         `json:"id"`
	Make              string      `json:"make"`
	Model             string      `json:"model"`
	Consumption       string      `json:"consumption"`
	Consumptionunit   string      `json:"consumptionunit"`
	Tripsum           string      `json:"tripsum"`
	Tripunit          string      `json:"tripunit"`
	Quantitysum       string      `json:"quantitysum"`
	Maintank          int         `json:"maintank"`
	Maintanktype      int         `json:"maintanktype"`
	Sign              string      `json:"sign"`
	PictureTs         int         `json:"picture_ts"`
	Bcconsumptionunit interface{} `json:"bcconsumptionunit"`
	Country           string      `json:"country"`
	RankingInfo       struct {
		Min       string `json:"min"`
		Avg       string `json:"avg"`
		Max       string `json:"max"`
		Unit      string `json:"unit"`
		Total     int    `json:"total"`
		Rank      int    `json:"rank"`
		Histogram []struct {
			Consumption     string `json:"consumption"`
			Count           int    `json:"count"`
			ContainsVehicle int    `json:"contains_vehicle"`
		} `json:"histogram"`
	} `json:"rankingInfo"`
}

type Fueling struct {
	ID                int         `json:"id"`
	Type              string      `json:"type"`
	Date              string      `json:"date"`
	Odometer          string      `json:"odometer"`
	Trip              string      `json:"trip"`
	Fuelsortid        int         `json:"fuelsortid"`
	Quantity          string      `json:"quantity"`
	Quantityunitid    int         `json:"quantityunitid"`
	QuantityConverted string      `json:"quantity_converted"`
	Cost              string      `json:"cost"`
	Currencyid        int         `json:"currencyid"`
	CostConverted     string      `json:"cost_converted"`
	Note              string      `json:"note"`
	Attributes        string      `json:"attributes"`
	Streets           string      `json:"streets"`
	Consumption       string      `json:"consumption"`
	BcSpeed           interface{} `json:"bc_speed"`
	BcQuantity        interface{} `json:"bc_quantity"`
	BcConsumption     interface{} `json:"bc_consumption"`
	Position          interface{} `json:"position"`
	Stationname       interface{} `json:"stationname"`
	Tankid            int         `json:"tankid"`
	Country           interface{} `json:"country"`
	Location          string      `json:"location"`
}

func init() {
	prometheus.MustRegister(promVehicleConsumption)
	prometheus.MustRegister(promVehicleTripSum)
	prometheus.MustRegister(promVehicleFuelSum)
	prometheus.MustRegister(promFuelingOdometer)
	prometheus.MustRegister(promFuelingTrip)
	prometheus.MustRegister(promFuelingQuantity)
	prometheus.MustRegister(promFuelingCost)
	prometheus.MustRegister(promFuelingConsumption)
}

func getApi(path string, target interface{}) error {
	r, err := http.NewRequest("GET", baseUrl + path, nil)
	if err != nil {
		log.Fatal(err)
	}
	r.SetBasicAuth(username, password)
	r.Header.Set("Application-Id", "81699ea0a8cf1e252cbbf5e582f3aad3")
	r.Header.Set("User-Agent", "Spritmonitor.de Android App (28) Samsung Galaxy S10")
	r.Header.Set("API-Language", "en")
	resp, err := myClient.Do(r)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
  if err := json.Unmarshal([]byte(body), &target); err != nil {
    log.Fatal(err)
  }

  return err
}

func getVehicles(target interface{}) error {
	return getApi("vehicles.json", &target)
}

func getFuelings(vehicleId int, target interface{}) error {
	return getApi(fmt.Sprintf("vehicle/%d/fuelings.json?limit=1000", vehicleId), &target)
}

func collectSample() {
	log.Println("Collecting sample...")
  var vehicles []Vehicle
	getVehicles(&vehicles)

	for _, vehicle := range vehicles {
		consumption, _ := strconv.ParseFloat(vehicle.Consumption, 64)
		promVehicleConsumption.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model}).Set(consumption)

		tripSum, _ := strconv.ParseFloat(vehicle.Tripsum, 64)
		promVehicleTripSum.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model}).Set(tripSum)

		fuelSum, _ := strconv.ParseFloat(vehicle.Quantitysum, 64)
		promVehicleFuelSum.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model}).Set(fuelSum)

		var fuelings []Fueling
		getFuelings(vehicle.ID, &fuelings)

		for _, fueling := range fuelings {
			odometer, _ := strconv.ParseFloat(fueling.Odometer, 64)
			promFuelingOdometer.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model, "date": fueling.Date, "fuelsortid": fmt.Sprintf("%d", fueling.Fuelsortid)}).Set(odometer)

			trip, _ := strconv.ParseFloat(fueling.Trip, 64)
			promFuelingTrip.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model, "date": fueling.Date, "fuelsortid": fmt.Sprintf("%d", fueling.Fuelsortid)}).Set(trip)

			quantity, _ := strconv.ParseFloat(fueling.Quantity, 64)
			promFuelingQuantity.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model, "date": fueling.Date, "fuelsortid": fmt.Sprintf("%d", fueling.Fuelsortid)}).Set(quantity)

			cost, _ := strconv.ParseFloat(fueling.Cost, 64)
			promFuelingCost.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model, "date": fueling.Date, "fuelsortid": fmt.Sprintf("%d", fueling.Fuelsortid)}).Set(cost)

			consumption, _ := strconv.ParseFloat(fueling.Consumption, 64)
			promFuelingConsumption.With(prometheus.Labels{"id": fmt.Sprintf("%d", vehicle.ID), "make": vehicle.Make, "model": vehicle.Model, "date": fueling.Date, "fuelsortid": fmt.Sprintf("%d", fueling.Fuelsortid)}).Set(consumption)
		}
	}
}

func main() {
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())

	collectSample()
	c := cron.New()
	c.AddFunc(fmt.Sprintf("@every %s", *every), collectSample)
	c.Start()

	log.Printf("Listening on %s!", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
