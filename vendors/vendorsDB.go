package vendors

import (
	"context"
	"errors"
	"naevis/config"
	"naevis/infra"
	"naevis/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	vendorCollection = config.Collections.VendorCollection
	hiringCollection = config.Collections.HiringCollection
)

// RegisterVendor creates a new vendor profile
func RegisterVendor(ctx context.Context, app *infra.Deps, userID, name, category, description string) (*models.Vendor, error) {
	vendor := &models.Vendor{
		VendorID:    primitive.NewObjectID().Hex(),
		UserID:      userID,
		Name:        name,
		Category:    category,
		Description: description,
		Rating:      0,
		RatingCount: 0,
		Verified:    false,
		Available:   true,
		CreatedAt:   time.Now(),
	}

	err := app.DB.InsertOne(ctx, vendorCollection, vendor)
	if err != nil {
		return nil, err
	}

	return vendor, nil
}

// GetVendorByID retrieves a vendor by vendor ID
func GetVendorByID(ctx context.Context, app *infra.Deps, vendorID string) (*models.Vendor, error) {
	var vendor models.Vendor
	err := app.DB.FindOne(ctx, vendorCollection, bson.M{"vendorid": vendorID}, &vendor)
	if err != nil {
		return nil, errors.New("vendor not found")
	}

	return &vendor, nil
}

// GetVendorByUserID retrieves vendor info for a specific user
func GetVendorByUserID(ctx context.Context, app *infra.Deps, userID string) (*models.Vendor, error) {
	var vendor models.Vendor
	err := app.DB.FindOne(ctx, vendorCollection, bson.M{"userid": userID}, &vendor)
	if err != nil {
		return nil, nil // User is not a vendor
	}

	return &vendor, nil
}

// GetVendorsByCategory retrieves all vendors in a specific category
func GetVendorsByCategory(ctx context.Context, app *infra.Deps, category string) ([]models.Vendor, error) {
	var vendors []models.Vendor
	err := app.DB.FindMany(ctx, vendorCollection, bson.M{"category": category, "available": true}, &vendors)
	if err != nil {
		return nil, err
	}

	return vendors, nil
}

// GetAllVendors retrieves all available vendors
func GetAllVendors(ctx context.Context, app *infra.Deps) ([]models.Vendor, error) {
	var vendors []models.Vendor
	err := app.DB.FindMany(ctx, vendorCollection, bson.M{"available": true}, &vendors)
	if err != nil {
		return nil, err
	}

	return vendors, nil
}

// UpdateVendor updates vendor information
func UpdateVendor(ctx context.Context, app *infra.Deps, vendorID string, updates bson.M) error {
	updates["updated_at"] = time.Now()
	return app.DB.Update(ctx, vendorCollection, bson.M{"vendorid": vendorID}, bson.M{"$set": updates})
}

// DeleteVendor removes a vendor (soft delete by setting available to false)
func DeleteVendor(ctx context.Context, app *infra.Deps, vendorID string) error {
	return UpdateVendor(ctx, app, vendorID, bson.M{"available": false})
}

// HireVendor creates a vendor hiring record for an event
func HireVendor(ctx context.Context, app *infra.Deps, eventID, vendorID, vendorName, vendorCategory, hiredBy string) (*models.VendorHiring, error) {
	// Check if vendor is already hired for this event
	var existing models.VendorHiring
	err := app.DB.FindOne(ctx, hiringCollection, bson.M{
		"eventid":  eventID,
		"vendorid": vendorID,
		"status":   bson.M{"$ne": "rejected"},
	}, &existing)
	if err == nil {
		return nil, errors.New("vendor already hired for this event")
	}

	hiring := &models.VendorHiring{
		HiringID:       primitive.NewObjectID().Hex(),
		EventID:        eventID,
		VendorID:       vendorID,
		VendorName:     vendorName,
		VendorCategory: vendorCategory,
		HiredBy:        hiredBy,
		Status:         "hired",
		HiredAt:        time.Now(),
		CreatedAt:      time.Now(),
	}

	err = app.DB.InsertOne(ctx, hiringCollection, hiring)
	if err != nil {
		return nil, err
	}

	return hiring, nil
}

// GetEventVendors retrieves all vendors hired for an event
func GetEventVendors(ctx context.Context, app *infra.Deps, eventID string) ([]models.VendorHiring, error) {
	var hirings []models.VendorHiring
	err := app.DB.FindMany(ctx, hiringCollection, bson.M{"eventid": eventID, "status": bson.M{"$ne": "rejected"}}, &hirings)
	if err != nil {
		return nil, err
	}

	return hirings, nil
}

// RemoveVendorFromEvent removes a vendor from an event
func RemoveVendorFromEvent(ctx context.Context, app *infra.Deps, eventID, vendorID string) error {
	return app.DB.Update(ctx, hiringCollection, bson.M{
		"eventid":  eventID,
		"vendorid": vendorID,
	}, bson.M{"$set": bson.M{"status": "rejected", "updated_at": time.Now()}})
}

// UpdateVendorStatus updates the status of a vendor hiring
func UpdateVendorStatus(ctx context.Context, app *infra.Deps, hiringID, status string) error {
	return app.DB.Update(ctx, hiringCollection, bson.M{"hiringid": hiringID}, bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}})
}

// GetVendorsByEvent retrieves detailed vendor info for an event
func GetVendorsByEvent(ctx context.Context, app *infra.Deps, eventID string) ([]models.VendorResponse, error) {
	hirings, err := GetEventVendors(ctx, app, eventID)
	if err != nil {
		return nil, err
	}

	var responses []models.VendorResponse
	for _, hiring := range hirings {
		vendor, err := GetVendorByID(ctx, app, hiring.VendorID)
		if err != nil {
			continue
		}
		responses = append(responses, models.VendorResponse{
			VendorID:     vendor.VendorID,
			Name:         vendor.Name,
			Category:     vendor.Category,
			Description:  vendor.Description,
			Email:        vendor.Email,
			Phone:        vendor.Phone,
			Location:     vendor.Location,
			Rating:       vendor.Rating,
			RatingCount:  vendor.RatingCount,
			ProfileImage: vendor.ProfileImage,
			Portfolio:    vendor.Portfolio,
			Verified:     vendor.Verified,
		})
	}

	return responses, nil
}
