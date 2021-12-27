package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	SymmetricBlockSize = 256
	SymmetricKeyLength = int(SymmetricBlockSize / 8)
)

var (
	ErrorInvalidKeyLength = fmt.Errorf(
		"invalid key length; key length must be %dB",
		SymmetricKeyLength,
	)
)

type ClientLogin struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type Client struct {
	Id        primitive.ObjectID `bson:"_id" json:"-"`
	ClientId  string             `bson:"client_id" json:"client_id"`
	Secret    string             `bson:"client_secret" json:"client_secret"`
	Nonce     string             `bson:"nonce" json:"-"`
	Salt      string             `bson:"salt" json:"-"`
	Alias     string             `bson:"alias" json:"alias"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	Options   map[string]string  `bson:"options" json:"options"`
}

type AccessToken struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}
