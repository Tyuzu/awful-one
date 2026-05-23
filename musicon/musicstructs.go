package musicon

import "time"

// --------------------------- Structs ---------------------------

type Album struct {
	ReleaseDate string   `json:"releaseDate" bson:"releaseDate"`
	Description string   `json:"description" bson:"description"`
	Published   bool     `json:"published" bson:"published"`
	Title       string   `json:"title" bson:"title"`
	ArtistID    string   `json:"artistid" bson:"artistid"`
	AlbumID     string   `json:"albumid" bson:"albumid"`
	Songs       []string `json:"songs" bson:"songs"`
	CoverURL    string   `json:"coverUrl,omitempty" bson:"coverUrl,omitempty"`
}

type Playlist struct {
	Name          string    `json:"name" bson:"name"`
	Description   string    `json:"description" bson:"description"`
	UserID        string    `json:"userid" bson:"userid"`
	PlaylistID    string    `json:"playlistid" bson:"playlistid"`
	Songs         []string  `json:"songs" bson:"songs"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
	Duration      int       `json:"duration" bson:"duration"`
	IsCompilation bool      `json:"isCompilation" bson:"isCompilation"`
	Copyrights    string    `json:"copyrights" bson:"copyrights"`
	CoverURL      string    `bson:"coverUrl,omitempty"`
}

type Song struct {
	SongID      string    `json:"songid" bson:"songid,omitempty"`
	ArtistID    string    `json:"artistid" bson:"artistid,omitempty"`
	Title       string    `json:"title" bson:"title"`
	Genre       string    `json:"genre" bson:"genre"`
	Duration    string    `json:"duration" bson:"duration"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	AudioURL    string    `json:"audioUrl,omitempty" bson:"audioUrl,omitempty"`
	Published   bool      `json:"published" bson:"published"`
	Plays       int       `json:"plays,omitempty" bson:"plays,omitempty"`
	UploadedAt  time.Time `json:"uploadedAt" bson:"uploadedAt"`
	Poster      string    `bson:"poster,omitempty" json:"poster,omitempty"`
	Language    string    `json:"language" bson:"language"`
	AudioExtn   string    `json:"audioextn" bson:"audioextn"`
	PosterExtn  string    `json:"posterextn" bson:"posterextn"`
}
