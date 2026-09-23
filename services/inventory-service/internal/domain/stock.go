package domain

import (
	"github.com/logicore-wms/logicore-wms/libs/platform"
	"time"
)

type Stock struct {
	ID          string     `json:"id"`
	ProductID   string     `json:"productId"`
	WarehouseID string     `json:"warehouseId"`
	LocationID  *string    `json:"locationId"`
	Quantity    int64      `json:"quantity"`
	Reserved    int64      `json:"reservedQuantity"`
	Batch       string     `json:"batchNumber"`
	Expiration  *time.Time `json:"expirationDate"`
	Weight      float64    `json:"unitWeight"`
	Volume      float64    `json:"unitVolume"`
	StorageType string     `json:"storageType"`
}

func Available(quantity, reserved int64) int64 { return quantity - reserved }
func CheckQuantity(quantity, available int64) error {
	if quantity <= 0 {
		return &platform.Fault{Code: "VALIDATION_ERROR", Message: "Quantity must be positive", Status: 400}
	}
	if quantity > available {
		return platform.Fail("INSUFFICIENT_STOCK", "Not enough available stock")
	}
	return nil
}

type Line struct {
	ProductID string `json:"productId"`
	Quantity  int64  `json:"quantity"`
}
