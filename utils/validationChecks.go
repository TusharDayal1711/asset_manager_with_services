package utils

import (
	"asset/models"
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"time"
)

func IsAssetTypeValid(assetType string) bool {
	return assetType == "laptop" || assetType == "mouse" || assetType == "monitor" || assetType == "hard_disk" ||
		assetType == "pen_drive" ||
		assetType == "mobile" ||
		assetType == "sim" ||
		assetType == "accessory"
}

func IsOwnershipValid(ownership string) bool {
	return ownership == "remotestate" || ownership == "client"
}

func AssetValidityCheck(reqModel models.AddAssetWithConfigReq) error {
	if strings.TrimSpace(reqModel.Brand) == "" {
		return errors.New("brand is required")
	}

	if strings.TrimSpace(reqModel.Model) == "" {
		return errors.New("model is required")
	}

	if strings.TrimSpace(reqModel.SerialNo) == "" {
		return errors.New("serial number is required")
	}

	if reqModel.PurchaseDate.After(time.Now()) {
		return errors.New("purchase date cannot be in the future")
	}

	if !IsAssetTypeValid(reqModel.Type) {
		return fmt.Errorf("invalid asset type")
	}

	if !IsOwnershipValid(reqModel.OwnedBy) {
		return fmt.Errorf("invalid owner ship")
	}
	return nil
}
