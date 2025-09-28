package model

type Entry struct {
	ID              int64
	TS              string
	Category        string
	Text            string
	Project         string
	Tags            string
	DurationMinutes int
}
