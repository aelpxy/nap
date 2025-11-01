package models

import "time"

type VPC struct {
	Name        string    `json:"name"`
	NetworkID   string    `json:"network_id"`
	NetworkName string    `json:"network_name"`
	Subnet      string    `json:"subnet"`
	CreatedAt   time.Time `json:"created_at"`
	Databases   []string  `json:"databases"`
	Apps        []string  `json:"apps"`
}

type VPCList struct {
	VPCs []VPC `json:"vpcs"`
}
