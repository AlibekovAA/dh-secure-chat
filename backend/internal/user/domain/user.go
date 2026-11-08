package domain

type ID string

type User struct {
	ID           ID
	Username     string
	PasswordHash string
}

type Summary struct {
	ID       ID
	Username string
}
