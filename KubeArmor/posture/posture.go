package posture

import (
	"context"
	"encoding/json"

	kg "github.com/kubearmor/KubeArmor/KubeArmor/log"
	tp "github.com/kubearmor/KubeArmor/KubeArmor/types"
	pb "github.com/kubearmor/KubeArmor/protobuf"
)

type ServiceServer struct {
	pb.PostureServiceServer
	UpdateDefaultPosture func(action string, namespace string, posture tp.DefaultPosture)
}

func (ps *ServiceServer) DefaultPosture(c context.Context, config *pb.Posture) (*pb.Result, error) {
	posture := tp.DefaultPosture{}
	res := new(pb.Result)

	err := json.Unmarshal(config.Data, &posture)
	if err == nil {
		ps.UpdateDefaultPosture(config.Action, config.Namespace, posture)
		res.Status = 1
	} else {
		kg.Warn("Invalid Posture")
		res.Status = 0
	}
	return res, nil
}
