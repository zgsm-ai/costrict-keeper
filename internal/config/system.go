package config

import (
	"costrict-keeper/internal/env"
	"costrict-keeper/internal/logger"
	"costrict-keeper/internal/models"
	"costrict-keeper/internal/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

/**
 * Load remote services configuration from URL
 * @param {string} url - URL of the remote configuration file
 * @returns {(*RemoteServicesConfig, error)} Returns configuration struct and error if any
 * @description
 * - Makes HTTP GET request to specified URL
 * - Validates HTTP response status code
 * - Reads response body and parses JSON
 * - Returns unmarshaled configuration structure
 * @throws
 * - HTTP request errors
 * - HTTP status code errors
 * - Response body reading errors
 * - JSON unmarshaling errors
 * @example
 * config, err := FetchRemoteSystemSpecification()
 * if err != nil {
 *     logger.Fatal(err)
 * }
 */
func FetchRemoteSystemSpecification() error {
	cfg := utils.UpgradeConfig{}
	cfg.PackageName = "system"
	cfg.TargetPath = filepath.Join(env.CostrictDir, "share", "system-spec.json")
	cfg.BaseUrl = fmt.Sprintf("%s/costrict", GetBaseURL())
	cfg.Correct()

	pkg, upgraded, err := utils.UpgradePackage(cfg, nil)
	if err != nil {
		logger.Errorf("fetch config failed: %v", err)
		return err
	}
	if !upgraded {
		logger.Infof("The '%s' version is up to date\n", cfg.PackageName)
	} else {
		logger.Infof("The '%s' is upgraded to version %s\n", cfg.PackageName, utils.PrintVersion(pkg.VersionId))
	}
	return nil
}

func LoadLocalSystemSpecification() (*models.SystemSpecification, error) {
	fname := filepath.Join(env.CostrictDir, "share", "system-spec.json")

	bytes, err := os.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("load 'system-spec.json' failed: %v", err)
	}
	var spec models.SystemSpecification
	if err := json.Unmarshal(bytes, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal 'system-spec.json' failed: %v", err)
	}
	return &spec, nil
}

var system *models.SystemSpecification

func LoadLocalSpec() error {
	if system != nil {
		return nil
	}
	var err error
	system, err = LoadLocalSystemSpecification()
	if err != nil {
		logger.Errorf("Load failed: %v", err)
		return err
	}
	return nil
}

func LoadSpec() error {
	if system != nil {
		return nil
	}
	if err := FetchRemoteSystemSpecification(); err != nil {
		logger.Errorf("Fetch failed: %v", err)
		return err
	}
	var err error
	system, err = LoadLocalSystemSpecification()
	if err != nil {
		logger.Errorf("Load failed: %v", err)
		return err
	}
	return nil
}

func Spec() *models.SystemSpecification {
	if system == nil {
		log.Fatalln("Must run config.LoadSpec/config.LoadLocalSpec first")
		return nil
	}
	return system
}
