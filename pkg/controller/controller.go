package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/crossedbot/common/golang/config"
	ccrypto "github.com/crossedbot/common/golang/crypto"
	caes "github.com/crossedbot/common/golang/crypto/aes"
	"github.com/crossedbot/simpleauth/pkg/database"
	"github.com/crossedbot/simplejwt"
	"github.com/crossedbot/simplejwt/algorithms"
	"github.com/crossedbot/simplejwt/jwk"
	middleware "github.com/crossedbot/simplemiddleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/crossedbot/simplem2m/pkg/models"
)

const (
	AccessTokenExpiration  = 1  // hours
	RefreshTokenExpiration = 24 // hours
)

var (
	ErrorClientIdRequired     = errors.New("Client ID is required")
	ErrorClientSecretRequired = errors.New("Client secret is required")
	ErrorClientNotFound       = errors.New("Client not found")
	ErrorBadCredentials       = errors.New("The client secret is incorrect")
	ErrorInvalidSecret        = errors.New("Invalid secret")
)

type Controller interface {
	Authenticate(models.ClientLogin) (models.AccessToken, error)
	Register(models.Client) (models.Client, error)
	GetJwks() (jwk.Jwks, error)
}

type controller struct {
	ctx           context.Context
	client        *mongo.Client
	encryptionKey []byte
	publicKey     []byte
	privateKey    []byte
	cert          jwk.Certificate
}

type Config struct {
	DatabaseAddr  string `toml:"database_addr"`
	EncryptionKey string `toml:"encryption_key"`
	PrivateKey    string `toml:"private_key"`
	Certificate   string `toml:"certificate"`
}

var control Controller
var controllerOnce sync.Once
var V1 = func() Controller {
	controllerOnce.Do(func() {
		var cfg Config
		if err := config.Load(&cfg); err != nil {
			panic(err)
		}
		ctx := context.Background()
		db, err := database.New(ctx, cfg.DatabaseAddr)
		if err != nil {
			panic(fmt.Errorf(
				"Controller: failed to connect to database at "+
					"address ('%s')",
				cfg.DatabaseAddr,
			))
		}
		encKey, err := ioutil.ReadFile(cfg.EncryptionKey)
		if err != nil {
			panic(fmt.Errorf(
				"Controller: encryption key not found ('%s')",
				cfg.EncryptionKey,
			))
		}
		privKey, err := ioutil.ReadFile(cfg.PrivateKey)
		if err != nil {
			panic(fmt.Errorf(
				"Controller: private key not found ('%s')",
				cfg.PrivateKey,
			))
		}
		cert := jwk.Certificate{}
		certFd, err := os.Open(cfg.Certificate)
		if err != nil {
			panic(fmt.Sprintf(
				"Controller: certificate not found ('%s')",
				cfg.Certificate,
			))
		}
		cert, err = jwk.NewCertificate(certFd)
		if err != nil {
			panic(fmt.Sprintf(
				"Controller: failed to parse certificate; %s",
				err,
			))
		}
		publicKey, err := cert.PublicKey()
		if err != nil {
			panic(fmt.Sprintf(
				"Controller: failed to parse certificate's "+
					"public key; %s",
				err,
			))
		}
		middleware.SetAuthPublicKey(publicKey)
		control = New(ctx, db, encKey, publicKey, privKey, cert)
	})
	return control
}

func New(
	ctx context.Context,
	client *mongo.Client,
	encryptionKey []byte,
	publicKey []byte,
	privateKey []byte,
	cert jwk.Certificate,
) Controller {
	return &controller{
		ctx,
		client,
		encryptionKey,
		publicKey,
		privateKey,
		cert,
	}
}

func (c *controller) Authenticate(login models.ClientLogin) (models.AccessToken, error) {
	clients := c.Clients()
	filter := bson.D{bson.E{Key: "client_id", Value: login.ClientId}}
	var foundClient models.Client
	err := clients.FindOne(c.ctx, filter).Decode(&foundClient)
	if err != nil {
		return models.AccessToken{}, ErrorClientNotFound
	}
	//Decode parameters
	secret, err := base64urlDecode(foundClient.Secret)
	if err != nil {
		return models.AccessToken{}, err
	}
	nonce, err := base64urlDecode(foundClient.Nonce)
	if err != nil {
		return models.AccessToken{}, err
	}
	salt, err := base64urlDecode(foundClient.Salt)
	if err != nil {
		return models.AccessToken{}, err
	}
	encParams, err := caes.NewEncryptionParamsWithValues(
		c.encryptionKey, salt, nonce)
	if err != nil {
		return models.AccessToken{}, err
	}
	// Authenticate the request
	err = VerifySecret(encParams, secret, []byte(login.ClientSecret))
	if err != nil {
		return models.AccessToken{}, ErrorBadCredentials
	}
	tkn, refreshTkn, err := GenerateTokens(
		foundClient, c.publicKey, c.privateKey)
	if err != nil {
		return models.AccessToken{}, err
	}
	err = c.UpdateTokens(tkn, refreshTkn, foundClient.ClientId)
	if err != nil {
		return models.AccessToken{}, err
	}
	return models.AccessToken{Token: tkn, RefreshToken: refreshTkn}, nil
}

func (c *controller) Register(client models.Client) (models.Client, error) {
	encParams, err := caes.NewEncryptionParams(c.encryptionKey)
	if err != nil {
		return models.Client{}, err
	}
	now, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	client.CreatedAt = now
	client.UpdatedAt = now
	client.Id = primitive.NewObjectID()
	client.ClientId = client.Id.Hex()
	client.Nonce = base64urlEncode(encParams.Nonce(nil))
	client.Salt = base64urlEncode(encParams.Salt(nil))
	secret, err := ccrypto.GenerateRandomString(32)
	if err != nil {
		return models.Client{}, err
	}
	client.Secret = base64urlEncode(encParams.Encrypt([]byte(secret)))
	_, err = c.Clients().InsertOne(c.ctx, client)
	if err != nil {
		return models.Client{}, err
	}
	// Set secret to plain text for viewing
	client.Secret = secret
	return client, nil
}

func (c *controller) GetJwks() (jwk.Jwks, error) {
	webKey, err := c.cert.ToJwk()
	return jwk.Jwks{Keys: []jwk.Jwk{webKey}}, err
}

func (c *controller) SetAuthCert(cert io.Reader) error {
	newCert, err := jwk.NewCertificate(cert)
	if err != nil {
		return err
	}
	publicKey, err := newCert.PublicKey()
	if err != nil {
		return err
	}
	middleware.SetAuthPublicKey(publicKey)
	c.cert = newCert
	return nil
}

func (c *controller) UpdateTokens(token, refreshToken, clientId string) error {
	clients := c.Clients()
	now, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	update := primitive.D{
		bson.E{Key: "token", Value: token},
		bson.E{Key: "refresh_token", Value: refreshToken},
		bson.E{Key: "updated_at", Value: now},
	}
	upsert := true
	_, err := clients.UpdateOne(
		c.ctx,
		bson.M{"client_id": clientId},
		bson.D{bson.E{Key: "$set", Value: update}},
		&options.UpdateOptions{Upsert: &upsert},
	)
	return err
}

func (c *controller) Clients() *mongo.Collection {
	return c.client.Database("auth").Collection("clients")
}

func VerifySecret(params caes.EncryptionParams, cipher, secret []byte) error {
	actual, err := params.Decrypt(cipher)
	if err != nil {
		return err
	}
	if bytes.Compare(actual, secret) != 0 {
		return ErrorInvalidSecret
	}
	return nil
}

func GenerateTokens(client models.Client, publicKey, privateKey []byte) (string, string, error) {
	claims := simplejwt.CustomClaims{
		"client_id":            client.ClientId,
		"alias":                client.Alias,
		middleware.ClaimUserId: client.ClientId,
		"exp": time.Now().Local().Add(
			time.Hour * time.Duration(AccessTokenExpiration),
		).Unix(),
	}
	jwt := simplejwt.New(claims, algorithms.AlgorithmRS256)
	jwt.Header["kid"] = jwk.EncodeToString(ccrypto.KeyId(publicKey))
	tkn, err := jwt.Sign(privateKey)
	if err != nil {
		return "", "", err
	}
	refreshClaims := simplejwt.CustomClaims{
		middleware.ClaimUserId: client.ClientId,
		"exp": time.Now().Local().Add(
			time.Hour * time.Duration(RefreshTokenExpiration),
		).Unix(),
	}
	refreshTkn, err := simplejwt.New(refreshClaims, algorithms.AlgorithmRS256).
		Sign(privateKey)
	if err != nil {
		return "", "", err
	}
	return tkn, refreshTkn, nil
}

func base64urlEncode(v []byte) string {
	return base64.URLEncoding.EncodeToString(v)
}

func base64urlDecode(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}
