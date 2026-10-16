package main

type Organization struct {
	Name       string `json:"name"`
	INN        string `json:"inn"`
	Director   string `json:"director"`
	Accountant string `json:"accountant"`
	Logo       string `json:"logo"`
	CreatedAt  string `json:"created_at"`
	Registered bool   `json:"registered"`
}

type ServerConfig struct {
	Version       string      `json:"version"`
	Initialized   bool        `json:"initialized"`
	Organization  interface{} `json:"organization"`
	CreatedAt     string      `json:"created_at"`
	MaxConcurrent int         `json:"max_concurrent"`
	Blacklist     interface{} `json:"blacklist"`
}

type User struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TabNumber    string `json:"tab_number"`
	Confirmed    bool   `json:"confirmed"`
	Active       bool   `json:"active"`
	PasswordHash string `json:"password_hash"`
	LastLogin    string `json:"last_login"`
}
