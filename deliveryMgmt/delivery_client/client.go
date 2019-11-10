// ***************************************
// Created by Alka Sinha on 9th Nov 2019
// ***************************************

package main

import (
	"GO/GeoLocation/deliveryMgmt/deliverypb"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/kelvins/geocoder"
	"google.golang.org/grpc"
)

//helper function to get the latitude and longitude from address
//used  geocoder library to achieve this
func getLocationfromAddress(adr geocoder.Address) (geocoder.Location, error) {
	location, err := geocoder.Geocoding(adr)
	if err != nil {
		fmt.Println("Could not get the location: ", err)
		return location, err
	}
	// fmt.Println("Latitude: ", location.Latitude)
	// fmt.Println("Longitude: ", location.Longitude)
	return location, nil
}

//function to interact with server with the request of inserting the delivery boy details into DB
func doCreateDeliveryBoy(c deliverypb.DeliveryServiceClient) {
	DBoyAddress := geocoder.Address{
		Street:  "Sarjapur Singnal ",
		Number:  0,
		City:    "Bengaluru",
		State:   "Karnataka",
		Country: "India",
	}

	location, err := getLocationfromAddress(DBoyAddress)
	if err != nil {
		log.Fatalf("error while getting the Latitude and longitude: %v", err)
	}
	lat := float32(location.Latitude)
	longi := float32(location.Longitude)

	dboy := &deliverypb.DeliveryBoy{
		Name:   "Vidya",
		Rating: "good",
		Location: &deliverypb.Location{Latitude: lat,
			Longitude: longi,
		},
	}

	createDboyRes, err := c.CreateDeliveryBoy(context.Background(), &deliverypb.CreateDeliveryBoyRequest{Emp: dboy})
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	fmt.Printf("Delivery Boy details has been pused into db: %v", createDboyRes)
}

//client function to list all the data from the db
func doListAllDeliveryBoys(c deliverypb.DeliveryServiceClient) {
	stream, err := c.GetAllDeliveryBoys(context.Background(), &deliverypb.GetAllDeliveryBoysRequest{})
	if err != nil {
		log.Fatalf("error while calling listBlog RPC %v\n", err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("something wrong happend %v\n", err)
		}
		// location := geocoder.Location{
		// 	Latitude:  float64(res.GetDeliveryBoy().GetLocation().GetLatitude()),
		// 	Longitude: float64(res.GetDeliveryBoy().GetLocation().GetLatitude()),
		// }
		// address, err := geocoder.GeocodingReverse(location)
		// if err != nil {
		// 	fmt.Println("could not convert into address ", err)
		// }
		fmt.Println(res.GetDeliveryBoy())
	}
}

//client function to assign delivery boy to the order
func doAssignDeliveryBoy(c deliverypb.DeliveryServiceClient) {
	RestaurantAddress := geocoder.Address{
		Street:  "Bellandur",
		Number:  0,
		City:    "Bengaluru",
		State:   "Karnataka",
		Country: "India",
	}
	DeliveryAddress := geocoder.Address{
		Street:  "Kaikondrahalli",
		Number:  0,
		City:    "Bengaluru",
		State:   "Karnataka",
		Country: "India",
	}

	restaurantLocation, err := getLocationfromAddress(RestaurantAddress)
	if err != nil {
		log.Fatalf("error while getting the Latitude and longitude: %v", err)
	}
	deliveryLocation, err := getLocationfromAddress(DeliveryAddress)
	if err != nil {
		log.Fatalf("error while getting the Latitude and longitude: %v", err)
	}

	orderDetails := &deliverypb.OrderDetails{
		Orderid: "a123",
		RestaurantLocation: &deliverypb.Location{
			Latitude:  float32(restaurantLocation.Latitude),
			Longitude: float32(restaurantLocation.Longitude),
		},
		DeliveryLocation: &deliverypb.Location{
			Latitude:  float32(deliveryLocation.Latitude),
			Longitude: float32(deliveryLocation.Longitude),
		},
	}

	res, err := c.ReceiveOreder(context.Background(), &deliverypb.ReceiveOrederRequest{OrderDetails: orderDetails})
	if err != nil {
		log.Fatalf("issues while calling the ReceiveOrder: %v ", err)
	}
	fmt.Println(strings.Repeat("-", 45))
	fmt.Println("Below is the delivery boy information assigned for the order : ", orderDetails.Orderid)
	fmt.Printf("%-30s %10s\n", "DeliveryBoyID", res.GetDeliveryBoy().GetId())
	fmt.Printf("%-30s %10s\n", "Name", res.GetDeliveryBoy().GetName())
	fmt.Printf("%-30s %10s\n", "Rating", res.GetDeliveryBoy().GetRating())

	//getting the current address of the delvery boy

	location := geocoder.Location{
		Latitude:  float64(res.GetDeliveryBoy().GetLocation().GetLatitude()),
		Longitude: float64(res.GetDeliveryBoy().GetLocation().GetLongitude()),
	}
	//code to get the address back t=from the latitude and longitude
	address, err := geocoder.GeocodingReverse(location)
	if err != nil {
		fmt.Println("Could not get the addresses: ", err)
	}
	// Usually, the first address returned from the API
	// is more detailed, so let's work with it
	addr := address[0]
	// Print the formatted address from the API
	fmt.Printf("%-30s %10s\n", "Address", addr.FormatAddress())
	fmt.Println("DeliveryAddress ", DeliveryAddress.Street, " ", DeliveryAddress.Number, " ", DeliveryAddress.City, " ", DeliveryAddress.State, " ", DeliveryAddress.Country)
	// Print the type of the address

}

func main() {

	//Google API to get the location details and addresses
	geocoder.ApiKey = "AIzaSyA4tOpyQl2H22qnzgUYOO5ewKLiK1xJ1yw"
	// See all Address fields in the documentation

	fmt.Println("DeliveryBoyMgmt client")
	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect client %v", err)
		return
	}
	//closing the client gracefully
	defer cc.Close()

	c := deliverypb.NewDeliveryServiceClient(cc)
	fmt.Println("delivery boy creation")
	doCreateDeliveryBoy(c)
	doListAllDeliveryBoys(c)
	doAssignDeliveryBoy(c)
}
