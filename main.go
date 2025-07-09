package main

import (
	"StealthIMFileAPI/config"
	"StealthIMFileAPI/gateway"
	"StealthIMFileAPI/grpc"
	"StealthIMFileAPI/msap"
	"StealthIMFileAPI/storage"
	"log"
)

func main() {
	cfg := config.ReadConf()
	log.Printf("Start server [%v]\n", config.Version)
	log.Printf("Block Size: %dKB\n", cfg.Storage.BlockSize)
	log.Printf("+ GRPC\n")
	log.Printf("    Host: %s\n", cfg.FileAPI.Host)
	log.Printf("    Port: %d\n", cfg.FileAPI.Port)
	log.Printf("+ DBGateway\n")
	log.Printf("    Host: %s\n", cfg.DBGateway.Host)
	log.Printf("    Port: %d\n", cfg.DBGateway.Port)
	log.Printf("+ FileStorage\n")
	for _, storage := range cfg.FileStorage {
		log.Printf("  + ID: %d\n", storage.ID)
		log.Printf("    Host: %s\n", storage.Host)
		log.Printf("    Port: %d\n", storage.Port)
	}
	if cfg.Callback.Host != "" {
		log.Printf("+ Callback MSAP\n")
		log.Printf("    Host: %s\n", cfg.Callback.Host)
		log.Printf("    Port: %d\n", cfg.Callback.Port)
		go msap.InitConns()
	}
	go gateway.InitConns()
	go storage.Start()
	grpc.Start(cfg)
}
