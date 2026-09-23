package domain

import "math"

type Candidate struct {
	ID            string  `json:"id"`
	WarehouseID   string  `json:"warehouseId"`
	MaxWeight     float64 `json:"maxWeight"`
	MaxVolume     float64 `json:"maxVolume"`
	CurrentWeight float64 `json:"currentWeight"`
	CurrentVolume float64 `json:"currentVolume"`
	Distance      float64 `json:"distance"`
	StorageType   string  `json:"storageType"`
	ZoneType      string  `json:"zoneType"`
	Status        string  `json:"status"`
}

func Score(c Candidate, warehouse, storageType string, weight, volume float64) (float64, bool) {
	if c.WarehouseID != warehouse || c.ZoneType != "STORAGE" || c.StorageType != storageType || c.Status == "BLOCKED" || c.Status == "MAINTENANCE" || c.MaxWeight <= 0 || c.MaxVolume <= 0 || weight <= 0 || volume <= 0 {
		return 0, false
	}
	if c.CurrentWeight+weight > c.MaxWeight+1e-9 || c.CurrentVolume+volume > c.MaxVolume+1e-9 {
		return 0, false
	}
	occupancy := math.Max(c.CurrentWeight/c.MaxWeight, c.CurrentVolume/c.MaxVolume)
	remaining := math.Min((c.MaxWeight-c.CurrentWeight-weight)/c.MaxWeight, (c.MaxVolume-c.CurrentVolume-volume)/c.MaxVolume)
	return 40 + 30*remaining + 30/(1+c.Distance/10) - 20*occupancy, true
}
