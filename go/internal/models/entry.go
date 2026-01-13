package models

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dict-simulator/go/internal/db"
)

// KeyType represents the type of Pix key
type KeyType string

const (
	KeyTypeCPF   KeyType = "CPF"
	KeyTypeCNPJ  KeyType = "CNPJ"
	KeyTypeEMAIL KeyType = "EMAIL"
	KeyTypePHONE KeyType = "PHONE"
	KeyTypeEVP   KeyType = "EVP"
)

// AccountType represents the type of bank account
type AccountType string

const (
	AccountTypeCACC AccountType = "CACC" // Current Account
	AccountTypeSVGS AccountType = "SVGS" // Savings Account
	AccountTypeSLRY AccountType = "SLRY" // Salary Account
)

// OwnerType represents the type of account owner
type OwnerType string

const (
	OwnerTypeNaturalPerson OwnerType = "NATURAL_PERSON"
	OwnerTypeLegalPerson   OwnerType = "LEGAL_PERSON"
)

// Account represents bank account information
type Account struct {
	Participant   string      `bson:"participant" json:"participant" validate:"required,len=8,numeric"`
	Branch        string      `bson:"branch" json:"branch" validate:"required,len=4,numeric"`
	AccountNumber string      `bson:"accountNumber" json:"accountNumber" validate:"required"`
	AccountType   AccountType `bson:"accountType" json:"accountType" validate:"required,oneof=CACC SVGS SLRY"`
}

// Owner represents the account owner information
type Owner struct {
	Type        OwnerType `bson:"type" json:"type" validate:"required,oneof=NATURAL_PERSON LEGAL_PERSON"`
	TaxIdNumber string    `bson:"taxIdNumber" json:"taxIdNumber" validate:"required"`
	Name        string    `bson:"name" json:"name" validate:"required"`
}

// Entry represents a DICT entry (Pix key registration)
type Entry struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Key       string             `bson:"key" json:"key"`
	KeyType   KeyType            `bson:"keyType" json:"keyType"`
	Account   Account            `bson:"account" json:"account"`
	Owner     Owner              `bson:"owner" json:"owner"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt,omitempty"`
}

// EntryResponse represents the API response for an entry
type EntryResponse struct {
	Key       string    `json:"key"`
	KeyType   KeyType   `json:"keyType"`
	Account   Account   `json:"account"`
	Owner     Owner     `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// CreateEntryRequest represents the request body for creating an entry
type CreateEntryRequest struct {
	Key     string  `json:"key" validate:"required"`
	KeyType KeyType `json:"keyType" validate:"required,oneof=CPF CNPJ EMAIL PHONE EVP"`
	Account Account `json:"account" validate:"required,dive"`
	Owner   Owner   `json:"owner" validate:"required,dive"`
}

// DeleteEntryResponse represents the response for deleting an entry
type DeleteEntryResponse struct {
	Message string `json:"message"`
	Key     string `json:"key"`
}

func EntriesCollection() *mongo.Collection {
	return db.Collection("entries")
}

// EnsureEntryIndexes creates necessary indexes for the entries collection
func EnsureEntryIndexes(ctx context.Context) error {
	collection := EntriesCollection()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "owner.taxIdNumber", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// CreateEntry creates a new entry in the database
func CreateEntry(ctx context.Context, req *CreateEntryRequest) (*Entry, error) {
	now := time.Now()
	entry := &Entry{
		Key:       req.Key,
		KeyType:   req.KeyType,
		Account:   req.Account,
		Owner:     req.Owner,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := EntriesCollection().InsertOne(ctx, entry)
	if err != nil {
		return nil, err
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, errors.New("failed to get inserted ID")
	}
	entry.ID = oid

	return entry, nil
}

// FindEntryByKey finds an entry by its key
func FindEntryByKey(ctx context.Context, key string) (*Entry, error) {
	var entry Entry
	err := EntriesCollection().FindOne(ctx, bson.M{"key": key}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

// DeleteEntryByKey deletes an entry by its key and returns the deleted entry
func DeleteEntryByKey(ctx context.Context, key string) (*Entry, error) {
	var entry Entry
	err := EntriesCollection().FindOneAndDelete(ctx, bson.M{"key": key}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

// ToResponse converts Entry to EntryResponse
func (e *Entry) ToResponse() EntryResponse {
	return EntryResponse{
		Key:       e.Key,
		KeyType:   e.KeyType,
		Account:   e.Account,
		Owner:     e.Owner,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
