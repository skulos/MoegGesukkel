package moeggrpc

import (
	"EkSukkel/moeggesukkel"
	"EkSukkel/persistence"
	"bufio"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger/fabric/common/flogging"
)

var log = flogging.MustGetLogger("GRPC")

type GrpcServer struct {
	moeggesukkel.UnimplementedMoegGeSukkelServer
}

const (
	BaseFilePath   string = "data/" // "/root/moeggesukkel/"
	defaultBufSize int    = 1024    //4096
)

func average[T int64 | string](inputData []T) T {

	log.Info("Starting Average function")

	// ScoreMap
	scoreMap := make(map[T]int)

	// Loop and count
	for _, v := range inputData {
		scoreMap[v] += 1
	}

	log.Info(scoreMap)

	var maxValue int = 0
	var mostInstances T

	for k, v := range scoreMap {
		if v > maxValue {
			mostInstances = k
			maxValue = v
		}
	}

	return mostInstances
}

func (gs *GrpcServer) Upload(stream moeggesukkel.MoegGeSukkel_UploadServer) error {

	// Get file name: UUID => until it's final
	fileRandomName := uuid.New().String()
	log.Info("Generating UUID as tempory file name: ", fileRandomName)

	// Reading all the files chunks and writing it to the file
	fileName := BaseFilePath + fileRandomName
	file, err := os.Open(fileName)

	if err != nil {
		log.Warning("ERROR = ", err)
		return err
	}

	// BufferedWriter
	buffWriter := bufio.NewWriter(file) //, 4096)
	var namesToAverage []string
	var timeToAverage []int64

	// loop oor stream
	// write to bufio.writer
	for {
		req, err := stream.Recv()

		if err != nil {
			break
		}

		// Append to names to be average => totally overkill
		namesToAverage = append(namesToAverage, req.GetFilename())
		timeToAverage = append(timeToAverage, req.GetTime())

		fileData := req.GetData()

		buffWriter.Write(fileData)
	}

	//  flush
	err = buffWriter.Flush()
	if err != nil {
		return err
	}

	// close file
	err = file.Close()
	if err != nil {
		return err
	}

	// average values
	averagedName := average(namesToAverage)
	averagedTime := average(timeToAverage)

	// insert
	path := BaseFilePath + averagedName
	ttl := time.Duration(averagedTime * int64(time.Second))
	token := persistence.UploadToCache(path, ttl)

	// rename
	err = os.Rename(fileName, path)
	if err != nil {
		return err
	}

	// return token
	log.Info("Returning token: ", token)
	result := moeggesukkel.UploadResponse{
		Token: token,
	}
	err = stream.SendAndClose(&result)

	return err
}

func (gs *GrpcServer) Download(req *moeggesukkel.DownloadRequest, stream moeggesukkel.MoegGeSukkel_DownloadServer) error {

	// Get the path of the token
	path := persistence.DownloadFromCache(req.GetToken())

	// File
	file, err := os.Open(path)
	fileName := file.Name()

	if err != nil {
		log.Warning("ERROR = ", err)
	}

	// BufferedReader
	dataArr := make([]byte, defaultBufSize)
	buffReader := bufio.NewReader(file)

	for {
		_, err := buffReader.Read(dataArr)
		if err == io.EOF {
			// there is no more data to read
			break
			// return err
		}

		response := moeggesukkel.DownloadResponse{
			Filename: fileName,
			Data:     dataArr,
		}

		err = stream.Send(&response)
		if err != nil {
			return err
		}
	}

	// close file
	err = file.Close()
	if err != nil {
		return err
	}

	return err
}
