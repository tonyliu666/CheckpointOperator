package utility

import (
	"context"
	"fmt"
	generated "grpc-test/generated"
	"io"
	"os"
	internal "grpc-test/internal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// TempDir = "./tmp"
	TempDir = "/home/Tony/MyOperstorProjects/practice/grpc/tar-file/tmp"
)

type AgentServer struct {
	generated.UnimplementedAgentServer
}

// first test the create OCI image on the localhost
func (s *AgentServer) CreateCheckpointImage(ctx context.Context, request *generated.CreateCheckpointImageRequest) (*generated.CreateCheckpointImageResponse, error) {
	fmt.Println("Got CreateCheckpointImage request")
	fmt.Println("archive",request.CheckpointArchiveLocation,"podname", request.PodName,"checkpointname", request.CheckpointName)
	// err := internal.CreateOCIImage(request.CheckpointArchiveLocation, request.PodName, request.CheckpointName)
	// imageId,checkpointName,err := internal.CreateCheckpointImage(ctx,request.CheckpointArchiveLocation, request.PodName, request.CheckpointName)
	_,_,err := internal.CreateCheckpointImage(ctx,request.CheckpointArchiveLocation, request.PodName, request.CheckpointName)
	if err != nil {
		return nil, err
	}
	return &generated.CreateCheckpointImageResponse{}, nil
}

func (s *AgentServer) TransferCheckpoint(ctx context.Context, request *generated.TransferCheckpointRequest) (*generated.TransferCheckpointResponse, error) {
	conn, err := grpc.Dial(request.TransferTo, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err, "did not connect")
		return nil, err
	}
	c := generated.NewAgentClient(conn)
	client, err := c.AcceptCheckpoint(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Println("opening checkpoint archive")
	checkpointArchive, err := os.Open(fmt.Sprintf("%s/%s", TempDir, request.PodName))
	if err != nil {
		return nil, err
	}
	defer checkpointArchive.Close()

	bytesSent := 0
	buffer := make([]byte, 4000)
	first := true
	for {
		bytesRead, err := checkpointArchive.Read(buffer)
		if err != nil {
			if err == io.EOF {
				fmt.Println("ignoring EOF")
				break
			}
			return nil, err
		}

		bytesSent += bytesRead

		if bytesRead < len(buffer) {
			// only send slice here to avoid sending superfluous data
			err = client.Send(&generated.AcceptCheckpointRequest{CheckpointOCIArchive: buffer[:bytesRead]})
			if err != nil {
				fmt.Println("error while sending last slice ", err)
				return nil, err
			}
			break
		}

		acceptRequest := &generated.AcceptCheckpointRequest{CheckpointOCIArchive: buffer}
		if first {
			acceptRequest.CheckpointImageName = request.PodName
			first = false
		}
		err = client.Send(acceptRequest)
		if err != nil {
			fmt.Println("error while sending ", err)
			return nil, err
		}
	}
	fmt.Println("sent bytes", bytesSent)

	fmt.Println("executing close send")
	response, err := client.CloseAndRecv()
	if err != nil {
		fmt.Println("CloseSend error ", err)
		return nil, err
	}
	fmt.Println(response)

	return &generated.TransferCheckpointResponse{}, nil
}
