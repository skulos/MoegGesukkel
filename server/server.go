package main

import (
	"EkSukkel/moeggesukkel"
	"EkSukkel/moeggrpc"
	"EkSukkel/persistence"
	"net"
	"os"
	"os/exec"

	// "rand"

	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"
)

var log = flogging.MustGetLogger("MAIN")

func dataFolder() {

	_, err := os.Stat(moeggrpc.BaseFilePath)

	if os.IsNotExist(err) {
		log.Warning("Folder, ", moeggrpc.BaseFilePath, ", does not exist.")
		log.Info("Creating folder ", moeggrpc.BaseFilePath)

		exec.Command("mkdir", moeggrpc.BaseFilePath)
	}
}

func main() {
	log.Info("Starting up server")

	log.Info("Startup protocols")
	dataFolder()

	persistence.Init()

	log.Info("New gRPC server")
	server := grpc.NewServer()

	log.Info("Registering server")
	moeggesukkel.RegisterMoegGeSukkelServer(server, &moeggrpc.GrpcServer{})

	log.Info("Allocating port :8080")
	lis, err := net.Listen("tcp", ":8080")

	if err != nil {
		log.Error("Failed to start a network listerner")
	}

	log.Info("Staring Moeggesukkel gRPC server")
	if err := server.Serve(lis); err != nil {
		log.Error("Failed to start gRPC server")
	}

}
