package models

import (
	"time"
)

type Baito struct {
	BaitoId          string    `bson:"baitoid,omitempty" json:"baitoid"`
	EntityType       string    `bson:"entityType" json:"entityType"`
	EntityID         string    `bson:"entityId" json:"entityId"`
	Title            string    `bson:"title" json:"title"`
	Description      string    `bson:"description" json:"description"`
	Category         string    `bson:"category" json:"category"`
	SubCategory      string    `bson:"subcategory" json:"subcategory"`
	Location         string    `bson:"location" json:"location"`
	Wage             string    `bson:"wage" json:"wage"`
	Phone            string    `bson:"phone" json:"phone"`
	Requirements     string    `bson:"requirements" json:"requirements"`
	BannerURL        string    `bson:"banner,omitempty" json:"banner,omitempty"`
	Images           []string  `bson:"images" json:"images"`
	WorkHours        string    `bson:"workHours" json:"workHours"`
	Benefits         string    `bson:"benefits,omitempty" json:"benefits,omitempty"`
	Email            string    `bson:"email,omitempty" json:"email,omitempty"`
	Tags             []string  `bson:"tags,omitempty" json:"tags,omitempty"`
	LastDateToApply  time.Time `bson:"lastdate" json:"lastdate"`
	CreatedAt        time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	OwnerID          string    `bson:"ownerId" json:"ownerId"`
	ApplicationCount int       `bson:"applicationcount" json:"applicationcount"`
}

type BaitoApplication struct {
	BaitoAppId  string    `bson:"baitoapplid,omitempty" json:"baitoapplid"`
	BaitoID     string    `bson:"baitoid" json:"baitoid"`
	UserID      string    `bson:"userid" json:"userid"`
	Username    string    `bson:"username" json:"username"`
	Pitch       string    `bson:"pitch" json:"pitch"`
	SubmittedAt time.Time `bson:"submittedAt" json:"submittedAt"`
}

type BaitoWorker struct {
	UserID       string   `json:"userId" bson:"userId"`
	BaitoUserID  string   `json:"baitoUserId" bson:"baitoUserId"`
	Name         string   `json:"name" bson:"name"`
	Age          int      `json:"age" bson:"age"`
	Phone        string   `json:"phone" bson:"phone"`
	Location     string   `json:"location" bson:"location"`
	Preferred    []string `json:"preferredRoles" bson:"preferredRoles"`
	Bio          string   `json:"bio" bson:"bio"`
	Avatar       string   `json:"avatar" bson:"avatar"`
	Email        string   `json:"email,omitempty" bson:"email,omitempty"`
	Experience   string   `json:"experience,omitempty" bson:"experience,omitempty"`
	Skills       string   `json:"skills,omitempty" bson:"skills,omitempty"`
	Availability string   `json:"availability,omitempty" bson:"availability,omitempty"`
	ExpectedWage string   `json:"expectedWage,omitempty" bson:"expectedWage,omitempty"`
	Languages    string   `json:"languages,omitempty" bson:"languages,omitempty"`
	Documents    []string `json:"documents,omitempty" bson:"documents,omitempty"`
	CreatedAt    int64    `json:"createdAt" bson:"createdAt"`
	UpdatedAt    int64    `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}
