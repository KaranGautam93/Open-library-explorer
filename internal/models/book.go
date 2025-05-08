package models

type Book struct {
	ISBN          string   `json:"isbn" bson:"isbn"`
	Title         string   `json:"title" bson:"title"`
	Author        string   `json:"author" bson:"author"`
	Publisher     string   `json:"publisher" bson:"publisher"`
	Tags          []string `json:"tags" bson:"tags"`
	Subject       string   `json:"subject" bson:"subject"`
	PublishedYear int      `json:"published_year" bson:"published_year"`
}

const (
	BookEntity = "book"
)
