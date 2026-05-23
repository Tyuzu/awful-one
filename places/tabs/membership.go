package places

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Membership struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	PlaceID     primitive.ObjectID `bson:"placeId,omitempty" json:"placeId"`
	Name        string             `bson:"name" json:"name"`
	Price       float64            `bson:"price" json:"price"`
	Description string             `bson:"description" json:"description"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

var membershipColl *mongo.Collection // set this from your DB init

// GET /place/:placeId/membership/:id
func GetMembership(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := primitive.ObjectIDFromHex(ps.ByName("id"))
	if err != nil {
		http.Error(w, "Invalid membership ID", http.StatusBadRequest)
		return
	}

	var membership Membership
	err = membershipColl.FindOne(r.Context(), bson.M{"_id": id}).Decode(&membership)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Membership not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch membership", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(membership)
}

// POST /place/:placeId/membership
func PostMembership(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	placeID, err := primitive.ObjectIDFromHex(ps.ByName("placeId"))
	if err != nil {
		http.Error(w, "Invalid place ID", http.StatusBadRequest)
		return
	}

	var membership Membership
	if err := json.NewDecoder(r.Body).Decode(&membership); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	membership.PlaceID = placeID
	membership.CreatedAt = time.Now()

	res, err := membershipColl.InsertOne(r.Context(), membership)
	if err != nil {
		http.Error(w, "Failed to create membership", http.StatusInternalServerError)
		return
	}

	membership.ID = res.InsertedID.(primitive.ObjectID)
	json.NewEncoder(w).Encode(membership)
}

// PUT /place/:placeId/membership/:id
func PutMembership(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := primitive.ObjectIDFromHex(ps.ByName("id"))
	if err != nil {
		http.Error(w, "Invalid membership ID", http.StatusBadRequest)
		return
	}

	var update Membership
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err = membershipColl.UpdateOne(
		r.Context(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"name":        update.Name,
			"price":       update.Price,
			"description": update.Description,
		}},
	)
	if err != nil {
		http.Error(w, "Failed to update membership", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /place/:placeId/membership/:id
func DeleteMembership(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := primitive.ObjectIDFromHex(ps.ByName("id"))
	if err != nil {
		http.Error(w, "Invalid membership ID", http.StatusBadRequest)
		return
	}

	_, err = membershipColl.DeleteOne(r.Context(), bson.M{"_id": id})
	if err != nil {
		http.Error(w, "Failed to delete membership", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /place/:placeId/membership/:id/join
func PostJoinMembership(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// This could insert into a `membership_users` collection
	http.Error(w, "Join membership not implemented", http.StatusNotImplemented)
}

// GET /place/:placeId/memberships
func GetMemberships(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	placeID, err := primitive.ObjectIDFromHex(ps.ByName("placeId"))
	if err != nil {
		http.Error(w, "Invalid place ID", http.StatusBadRequest)
		return
	}

	cur, err := membershipColl.Find(r.Context(), bson.M{"placeId": placeID})
	if err != nil {
		http.Error(w, "Failed to fetch memberships", http.StatusInternalServerError)
		return
	}
	defer cur.Close(context.Background())

	var memberships []Membership
	if err := cur.All(r.Context(), &memberships); err != nil {
		http.Error(w, "Failed to parse memberships", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(memberships)
}
