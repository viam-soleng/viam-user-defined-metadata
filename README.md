# Module user-defined-metadata 

A Viam sensor component that provides access to user-defined metadata for robots and robot parts through the Viam Fleet Management API. This module allows you to read and update custom metadata associated with your machines and their parts.

## Model viam-soleng:sensor:user-defined-metadata

This sensor component fetches and updates user-defined metadata from the Viam platform using the Fleet Management API. It provides both read and write access to metadata at the robot (machine) level and robot part level.

This module is implemented in Go using the [Viam Go SDK](https://github.com/viamrobotics/rdk).

### Features

- **Read Metadata**: Get user-defined metadata for both robots and robot parts via `Readings` (`GetReadings`)
- **Update Metadata**: Update user-defined metadata through `DoCommand`
- **Automatic Authentication**: Uses Viam API keys for secure access to the Fleet Management API

This resource automatically detects robot and part IDs from Viam environment variables

### Configuration

This component requires API credentials to access the Viam Fleet Management API. No additional configuration attributes are needed in the component configuration.

#### Required Environment Variables

The following environment variables must be set for the component to function.
If running from the registry, they will typically already be available in the environment.

| Variable | Description | Source |
|----------|-------------|---------|
| `VIAM_API_KEY` | Your Viam API key for Fleet Management API access | User-provided |
| `VIAM_API_KEY_ID` | Your Viam API key ID | User-provided |
| `VIAM_MACHINE_ID` | The ID of the machine/robot | Automatically set by Viam |
| `VIAM_MACHINE_PART_ID` | The ID of the robot part | Automatically set by Viam |

#### Example Configuration

```json
{
  "name": "metadata-sensor",
  "model": "viam-soleng:sensor:user-defined-metadata",
  "type": "sensor"
}
```

### API Usage

#### GetReadings

Returns the current user-defined metadata for both the robot and robot part.

**Response Format:**
```json
{
  "robot": {
    "owner": "john.doe",
    "maintenance_schedule": "weekly"
  },
  "part": {
    "calibration_date": "2024-01-15",
    "firmware_version": "1.2.3"
  }
}
```

#### DoCommand

Updates user-defined metadata for either the robot or robot part.

**Command Format** (`"scope"` is `"robot"` for machine-level metadata, `"part"` for part-level):
```json
{
  "command": "update",
  "scope": "robot",
  "metadata": {
    "key": "value"
  }
}
```

**Examples:**

Update robot-level metadata:
```json
{
  "command": "update",
  "scope": "robot",
  "metadata": {
    "location": "warehouse-2",
    "last_maintenance": "2024-01-20",
    "status": "operational"
  }
}
```

Update robot part-level metadata:
```json
{
  "command": "update",
  "scope": "part",
  "metadata": {
    "calibration_date": "2024-01-25",
    "sensor_range": "0-100°C",
    "accuracy": "±0.5°C"
  }
}
```

**Response Format:**

Success:
```json
{
  "success": true,
  "message": "Robot metadata updated successfully",
  "scope": "robot",
  "robot_id": "your-robot-id"
}
```

Error:
```json
{
  "success": false,
  "error": "Error message details",
  "scope": "robot",
  "command": "update"
}
```
