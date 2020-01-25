package main

type isbnAPI interface {
	Get(isbn string) error
	Save(path string) error
	Load(path string) error
}
