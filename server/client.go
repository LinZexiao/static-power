package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"static-power/api"
)

var client *http.Client = &http.Client{}
var host string = "127.0.0.1:8090"

func SetHost(h string) {
	host = h
}

func baseUrl(rel string) string {
	return "http://" + host + "/api/v0/" + rel
}

func GetMiners() ([]api.Miner, error) {
	resp, err := client.Get(baseUrl("miner"))
	if err != nil {
		return nil, fmt.Errorf("get /miner err: %w", err)
	}
	log.Println(resp.Status)
	defer resp.Body.Close()
	var miners []api.Miner
	// json marshal body into target
	err = json.NewDecoder(resp.Body).Decode(&miners)
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return miners, nil
}

func UpdatePowerInfo(power *api.PowerInfo) error {
	data, err := json.Marshal(power)
	if err != nil {
		return fmt.Errorf("marshal power info error: %w", err)
	}
	r := bytes.NewReader(data)
	resp, err := client.Post(baseUrl("power"), "application/json", r)
	if err != nil {
		return fmt.Errorf("post /power err: %w", err)
	}
	log.Println(resp.Status)
	defer resp.Body.Close()
	return nil
}

func UpdateAgentInfo(agent *api.AgentInfo) error {
	data, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("marshal agent info error: %w", err)
	}
	r := bytes.NewReader(data)
	resp, err := client.Post(baseUrl("agent"), "application/json", r)
	if err != nil {
		return fmt.Errorf("post /agent err: %w", err)
	}
	log.Println(resp.Status)
	defer resp.Body.Close()
	return nil
}

func UpdatePeerInfo(peer *api.PeerInfo) error {
	data, err := json.Marshal(peer)
	if err != nil {
		return fmt.Errorf("marshal peer info error: %w", err)
	}
	r := bytes.NewReader(data)
	resp, err := client.Post(baseUrl("peer"), "application/json", r)
	if err != nil {
		return fmt.Errorf("post /peer err: %w", err)
	}
	log.Println(resp.Status)
	defer resp.Body.Close()
	return nil
}
