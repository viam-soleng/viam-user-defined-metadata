// Package models implements the viam-soleng:sensor:user-defined-metadata sensor.
//
// It provides read and write access to user-defined metadata for the machine
// (robot) and machine part it runs on, via the Viam Fleet Management (App) API.
package models

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.viam.com/rdk/app"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// Model is the resource model triple for this sensor.
var Model = resource.NewModel("viam-soleng", "sensor", "user-defined-metadata")

func init() {
	resource.RegisterComponent(sensor.API, Model,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: newUserDefinedMetadata,
		},
	)
}

// Config holds the (currently empty) component configuration. No attributes are
// required: the component reads its identity and credentials from environment
// variables. See Validate.
type Config struct{}

// Validate reports required and optional dependencies. This component has none.
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	return nil, nil, nil
}

type userDefinedMetadata struct {
	resource.Named
	resource.AlwaysRebuild

	logger logging.Logger

	mu         sync.Mutex
	viamClient *app.ViamClient
}

func newUserDefinedMetadata(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (sensor.Sensor, error) {
	return &userDefinedMetadata{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}, nil
}

// getClientLocked lazily creates and caches an authenticated ViamClient.
// Credentials are read from the VIAM_API_KEY and VIAM_API_KEY_ID environment
// variables by the SDK helper. The caller must hold s.mu so that the client's
// creation and use are serialized with Close (see Readings/DoCommand).
func (s *userDefinedMetadata) getClientLocked(ctx context.Context) (*app.ViamClient, error) {
	if s.viamClient != nil {
		return s.viamClient, nil
	}

	client, err := app.CreateViamClientFromEnvVars(ctx, nil, s.logger)
	if err != nil {
		return nil, err
	}
	s.viamClient = client
	return client, nil
}

// machineIDs returns the robot and robot-part IDs from the environment.
func machineIDs() (robotID, robotPartID string, err error) {
	robotID = os.Getenv("VIAM_MACHINE_ID")
	robotPartID = os.Getenv("VIAM_MACHINE_PART_ID")
	if robotID == "" {
		return "", "", fmt.Errorf("VIAM_MACHINE_ID environment variable is required")
	}
	if robotPartID == "" {
		return "", "", fmt.Errorf("VIAM_MACHINE_PART_ID environment variable is required")
	}
	return robotID, robotPartID, nil
}

// Readings returns robot and robot-part user-defined metadata under the "robot"
// and "part" keys. On error it logs and returns empty metadata for both.
func (s *userDefinedMetadata) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	robotID, robotPartID, err := machineIDs()
	if err != nil {
		s.logger.Errorf("Error fetching metadata: %v", err)
		return map[string]interface{}{"robot": map[string]interface{}{}, "part": map[string]interface{}{}}, nil
	}

	// Hold the lock for the duration of the request so the client cannot be
	// closed out from under the in-flight gRPC calls by a concurrent Close.
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.getClientLocked(ctx)
	if err != nil {
		s.logger.Errorf("Error fetching metadata: %v", err)
		return map[string]interface{}{"robot": map[string]interface{}{}, "part": map[string]interface{}{}}, nil
	}
	appClient := client.AppClient()

	robotMetadata, err := appClient.GetRobotMetadata(ctx, robotID)
	if err != nil {
		s.logger.Errorf("Error fetching metadata: %v", err)
		return map[string]interface{}{"robot": map[string]interface{}{}, "part": map[string]interface{}{}}, nil
	}

	partMetadata, err := appClient.GetRobotPartMetadata(ctx, robotPartID)
	if err != nil {
		s.logger.Errorf("Error fetching metadata: %v", err)
		return map[string]interface{}{"robot": map[string]interface{}{}, "part": map[string]interface{}{}}, nil
	}

	return map[string]interface{}{
		"robot": robotMetadata,
		"part":  partMetadata,
	}, nil
}

// DoCommand updates user-defined metadata for the robot or robot part.
//
// Expected command:
//
//	{
//	  "command":  "update",
//	  "scope":    "part" | "robot",
//	  "metadata": { ... }
//	}
func (s *userDefinedMetadata) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	result, err := s.doUpdate(ctx, cmd)
	if err != nil {
		errMsg := fmt.Sprintf("Error updating metadata: %v", err)
		s.logger.Error(errMsg)
		// Echo back the caller's raw scope/command values, falling back to
		// "unknown" only when the key is absent.
		return map[string]interface{}{
			"success": false,
			"error":   errMsg,
			"scope":   valueOrUnknown(cmd["scope"]),
			"command": valueOrUnknown(cmd["command"]),
		}, nil
	}
	return result, nil
}

// valueOrUnknown returns v, or "unknown" when v is absent (nil).
func valueOrUnknown(v interface{}) interface{} {
	if v == nil {
		return "unknown"
	}
	return v
}

func (s *userDefinedMetadata) doUpdate(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	command, _ := cmd["command"].(string)
	scope, _ := cmd["scope"].(string)
	metadata, metadataOK := cmd["metadata"].(map[string]interface{})

	if command != "update" {
		return nil, fmt.Errorf("unsupported command: %v. Only 'update' is supported", cmd["command"])
	}
	if scope != "part" && scope != "robot" {
		return nil, fmt.Errorf("invalid scope: %v. Must be 'part' or 'robot'", cmd["scope"])
	}
	if !metadataOK {
		return nil, fmt.Errorf("metadata must be an object")
	}

	robotID, robotPartID, err := machineIDs()
	if err != nil {
		return nil, err
	}

	// Hold the lock for the duration of the request so the client cannot be
	// closed out from under the in-flight gRPC calls by a concurrent Close.
	s.mu.Lock()
	defer s.mu.Unlock()

	client, err := s.getClientLocked(ctx)
	if err != nil {
		return nil, err
	}
	appClient := client.AppClient()

	switch scope {
	case "robot":
		if err := appClient.UpdateRobotMetadata(ctx, robotID, metadata); err != nil {
			return nil, err
		}
		s.logger.Infof("Successfully updated robot metadata for robot %s", robotID)
		return map[string]interface{}{
			"success":  true,
			"message":  "Robot metadata updated successfully",
			"scope":    "robot",
			"robot_id": robotID,
		}, nil
	default: // "part"
		if err := appClient.UpdateRobotPartMetadata(ctx, robotPartID, metadata); err != nil {
			return nil, err
		}
		s.logger.Infof("Successfully updated robot part metadata for part %s", robotPartID)
		return map[string]interface{}{
			"success":       true,
			"message":       "Robot part metadata updated successfully",
			"scope":         "part",
			"robot_part_id": robotPartID,
		}, nil
	}
}

// Close releases the cached ViamClient connection, if any.
func (s *userDefinedMetadata) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.viamClient != nil {
		err := s.viamClient.Close()
		s.viamClient = nil
		return err
	}
	return nil
}
