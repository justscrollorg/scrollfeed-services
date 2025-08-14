package main

type Meme struct {
	ID        string `bson:"_id,omitempty" json:"id"`
	Title     string `bson:"title" json:"title"`
	ImageURL  string `bson:"image_url" json:"image_url"`
	Source    string `bson:"source" json:"source"`
	Permalink string `bson:"permalink" json:"permalink"`
}
