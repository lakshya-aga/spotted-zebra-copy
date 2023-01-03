// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0

package db

import ()

type Corrpair struct {
	Date string  `json:"date"`
	X0   string  `json:"x0"`
	X1   string  `json:"x1"`
	Corr float64 `json:"corr"`
}

type Historicaldatum struct {
	Date       string  `json:"date"`
	Ticker     string  `json:"ticker"`
	K          float64 `json:"k"`
	T          float64 `json:"t"`
	Ivol       float64 `json:"ivol"`
	Underlying string  `json:"underlying"`
}

type Modelparameter struct {
	Date   string  `json:"date"`
	Ticker string  `json:"ticker"`
	Sigma  float64 `json:"sigma"`
	Alpha  float64 `json:"alpha"`
	Beta   float64 `json:"beta"`
	Kappa  float64 `json:"kappa"`
	Rho    float64 `json:"rho"`
}

type Statistic struct {
	Date   string  `json:"date"`
	Ticker string  `json:"ticker"`
	Index  int32   `json:"index"`
	Mean   float64 `json:"mean"`
	Fixing float64 `json:"fixing"`
}

type User struct {
	EmailAddress string `json:"email_address"`
	Prefix       string `json:"prefix"`
	Token        string `json:"token"`
	GeneratedAt  string `json:"generated_at"`
	ExpiredAt    string `json:"expired_at"`
}
