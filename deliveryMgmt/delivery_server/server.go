// ***************************************
// Created by Alka Sinha on 9th Nov 2019
// ***************************************

package main

import (
	"GO/GeoLocation/deliveryMgmt/deliverypb"
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
}

//model for mongoDB table/collection
var collection *mongo.Collection

//struct model for Location
type Locasion struct {
	Longitude float32
	Lattitude float32
}

//model for Delivery Boy
type DeliveryBoyInfo struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `bson:"name"`
	Rating   string             `bson:"rating"`
	Location Locasion           `bson:"location"`
}

//interface implementation
//interface to get the data from the client as a request and save it into mongo db and return the data with created ID asresponse
func (*server) CreateDeliveryBoy(ctx context.Context, req *deliverypb.CreateDeliveryBoyRequest) (*deliverypb.CreateDeliveryBoyResponse, error) {
	fmt.Println("Create Delivery Boy request")
	dboy := req.GetEmp()

	data := DeliveryBoyInfo{
		Name:   dboy.GetName(),
		Rating: dboy.GetRating(),
		Location: Locasion{Longitude: float32(dboy.GetLocation().GetLatitude()),
			Lattitude: float32(dboy.GetLocation().GetLongitude())},
	}

	//fmt.Println(data)
	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}

	return &deliverypb.CreateDeliveryBoyResponse{
		Emp: &deliverypb.DeliveryBoy{
			Id:     oid.Hex(),
			Name:   dboy.GetName(),
			Rating: dboy.GetRating(),
			Location: &deliverypb.Location{Latitude: dboy.GetLocation().GetLatitude(),
				Longitude: dboy.GetLocation().GetLongitude()},
		},
	}, nil

}

//rpc interface implementation
//this will response all the delivery boy details from the database so that we can complare the shortest distance from the restaurant to the delivery boy
func (*server) GetAllDeliveryBoys(req *deliverypb.GetAllDeliveryBoysRequest, stream deliverypb.DeliveryService_GetAllDeliveryBoysServer) error {
	fmt.Println("list All DeliveryBoy details Request")
	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown Internal Error received %v", err),
		)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &DeliveryBoyInfo{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from Mongo DB %v", err),
			)

		}
		stream.Send(&deliverypb.GetAllDeliveryBoysResponse{DeliveryBoy: dataToDeliverypb(data)})
	}
	if err = cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown cursur Internal Error received %v", err),
		)
	}
	return nil
}

//helper function the map received the data with the struct model
func dataToDeliverypb(data *DeliveryBoyInfo) *deliverypb.DeliveryBoy {
	return &deliverypb.DeliveryBoy{
		Id:     data.ID.Hex(),
		Name:   data.Name,
		Rating: data.Rating,
		Location: &deliverypb.Location{Latitude: data.Location.Lattitude,
			Longitude: data.Location.Longitude,
		},
	}
}

//interface implementation
//interface to receive the request with the order information from the client and the server will response with assigned delivery boy details on the baasis of shortest distance
func (*server) ReceiveOreder(ctx context.Context, req *deliverypb.ReceiveOrederRequest) (*deliverypb.ReceiveOrederResponse, error) {
	fmt.Println("ReceiveOreder request")
	var minDistance float64
	minDistance = 99999999.99999
	orederDetails := req.GetOrderDetails()
	//orderId := orederDetails.GetOrderid()
	restaurantLat := float64(orederDetails.GetRestaurantLocation().GetLatitude())
	restaurantLongi := float64(orederDetails.GetRestaurantLocation().GetLongitude())

	//deliveryLat := float64(orederDetails.GetDeliveryLocation().GetLatitude())
	//deliveryLongi := float64(orederDetails.GetDeliveryLocation().GetLongitude())

	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown Internal Error received %v", err),
		)
	}
	defer cur.Close(context.Background())

	deliverBoyDetails := &DeliveryBoyInfo{}
	for cur.Next(context.Background()) {
		data := &DeliveryBoyInfo{}
		err := cur.Decode(data)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from Mongo DB %v", err),
			)
		}
		deliverBoyLat := float64(data.Location.Lattitude)
		deliverBoyLongi := float64(data.Location.Longitude)
		distance := distance(restaurantLat, restaurantLongi, deliverBoyLat, deliverBoyLongi, "K")
		if minDistance > distance {
			minDistance = distance
			fmt.Println(minDistance)
			deliverBoyDetails = data
		}
	}
	if err = cur.Err(); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown cursur Internal Error received %v", err),
		)
	}
	return &deliverypb.ReceiveOrederResponse{DeliveryBoy: dataToDeliverypb(deliverBoyDetails)}, nil
}

//helper function to find the distance between two latitude and longitude
func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64, unit ...string) float64 {
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	if len(unit) > 0 {
		if unit[0] == "K" {
			dist = dist * 1.609344
		} else if unit[0] == "N" {
			dist = dist * 0.8684
		}
	}

	return dist
}

//server main function
func main() {
	//if we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Delivery Server Started")
	//creation of mongo db client and connection with the database and creating and deliveryboy table under mydb
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Delevery Boy Management service started.....")
	collection = client.Database("mydb").Collection("deliveryboy")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		log.Fatalf("failed to listen  %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)

	deliverypb.RegisterDeliveryServiceServer(s, &server{})
	reflection.Register(s)

	//binding the port to grpc server
	go func() {
		fmt.Println("starting server.....")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve %v ", err)
		}
	}()
	//channel to give the interruption signal to stop the server,listener and DB connection gracefully
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch
	fmt.Println("stopping the server...")
	s.Stop()
	fmt.Println("closing the listener...")
	lis.Close()
	fmt.Println("Closing MongoDB Connection")
	client.Disconnect(context.TODO())
	fmt.Println("End of Program")

}
