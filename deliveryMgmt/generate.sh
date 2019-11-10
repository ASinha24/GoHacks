#!/bin/bash

protoc deliveryMgmt/deliverypb/delivery.proto --go_out=plugins=grpc:.
