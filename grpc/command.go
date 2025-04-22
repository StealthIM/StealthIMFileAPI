// Debug GRPC Command Runner
// Do Not Use Online!

package grpc

import (
	"StealthIMFileAPI/gateway"
	"context"
	"fmt"
	"strings"

	pb_gtw "StealthIMFileAPI/StealthIM.DBGateway"
	pb "StealthIMFileAPI/StealthIM.FileAPI"

	"google.golang.org/protobuf/encoding/protojson"
)

func (s *server) Command(ctx context.Context, in *pb.CommandRequest) (*pb.CommandResponse, error) {
	args := strings.Split(in.Command, " ")
	switch args[0] {
	case "sql":
		switch args[1] {
		case "1":
			res, err := gateway.ExecSQL(&pb_gtw.SqlRequest{Commit: true, Sql: "INSERT INTO user_info (uid, create_time, vip, email, phone_number) VALUES (11114, '2023-10-05 14:48:00', 1, 'example@example.com', '123-456-7890');", Db: pb_gtw.SqlDatabases_Users})
			if err != nil {
				return &pb.CommandResponse{Result: (err.Error())}, nil
			}
			r, _ := protojson.Marshal(res)
			fmt.Printf("SQL Result: %v\n", string(r))
			return &pb.CommandResponse{Result: ""}, nil
		case "2":
			res, err := gateway.ExecSQL(&pb_gtw.SqlRequest{Commit: false, Sql: "SELECT * FROM user_info;", Db: pb_gtw.SqlDatabases_Users})
			if err != nil {
				return &pb.CommandResponse{Result: (err.Error())}, nil
			}
			r, _ := protojson.Marshal(res)
			fmt.Printf("SQL Result: %v\n", string(r))
			return &pb.CommandResponse{Result: ""}, nil
		}
	}
	return &pb.CommandResponse{Result: "Null Command"}, nil
}
