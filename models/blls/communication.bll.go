package blls

import (
	"TaskManagementSystem_Api/models/dals"
	"TaskManagementSystem_Api/models/types"
)

type CommunicationBLL struct {
}

func (bll *CommunicationBLL) GetCommunications(id string) (c []*types.Communication_Get, err error) {
	c, err = (&dals.CommunicationDAL{}).GetCommunications(id)
	return
}

func (bll *CommunicationBLL) AddCommunication(c types.Communication_Post) (s map[string]string, err error) {
	s, err = (&dals.CommunicationDAL{}).AddCommunication(c)
	return
}
