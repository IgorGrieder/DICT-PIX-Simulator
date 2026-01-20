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

// OwnerType represents the type of account owner
type OwnerType string

// Reason represents the reason for an entry operation
type Reason string

// Account represents bank account information
type Account struct {
	Participant   string      `bson:"participant" json:"participant" validate:"required,len=8,numeric" example:"12345678"`
	Branch        string      `bson:"branch" json:"branch" validate:"required,len=4,numeric" example:"0001"`
	AccountNumber string      `bson:"accountNumber" json:"accountNumber" validate:"required" example:"123456789"`
	AccountType   AccountType `bson:"accountType" json:"accountType" validate:"required,oneof=CACC SVGS SLRY" example:"CACC"`
	OpeningDate   time.Time   `bson:"openingDate" json:"openingDate" validate:"required" example:"2024-01-15T00:00:00Z"`
}

// Owner represents the account owner information
type Owner struct {
	Type        OwnerType `bson:"type" json:"type" validate:"required,oneof=NATURAL_PERSON LEGAL_PERSON" example:"NATURAL_PERSON"`
	TaxIdNumber string    `bson:"taxIdNumber" json:"taxIdNumber" validate:"required" example:"12345678901"`
	Name        string    `bson:"name" json:"name" validate:"required" example:"John Doe"`
	TradeName   string    `bson:"tradeName,omitempty" json:"tradeName,omitempty" example:"Doe Enterprises"` // Only for LEGAL_PERSON
}

// UpdateAccount represents partial account updates (no required validations)
type UpdateAccount struct {
	Participant   string      `bson:"participant,omitempty" json:"participant,omitempty" validate:"omitempty,len=8,numeric" example:"12345678"`
	Branch        string      `bson:"branch,omitempty" json:"branch,omitempty" validate:"omitempty,len=4,numeric" example:"0001"`
	AccountNumber string      `bson:"accountNumber,omitempty" json:"accountNumber,omitempty" example:"123456789"`
	AccountType   AccountType `bson:"accountType,omitempty" json:"accountType,omitempty" validate:"omitempty,oneof=CACC SVGS SLRY" example:"CACC"`
	OpeningDate   *time.Time  `bson:"openingDate,omitempty" json:"openingDate,omitempty" example:"2024-01-15T00:00:00Z"`
}

// UpdateOwner represents partial owner updates (no required validations)
// Per DICT spec: Only name and trade name can be updated
type UpdateOwner struct {
	Name      string `bson:"name,omitempty" json:"name,omitempty" example:"John Doe"`
	TradeName string `bson:"tradeName,omitempty" json:"tradeName,omitempty" example:"Doe Enterprises"`
}

// Entry represents a DICT entry (Pix key registration)
type Entry struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Key              string             `bson:"key" json:"key"`
	KeyType          KeyType            `bson:"keyType" json:"keyType"`
	Account          Account            `bson:"account" json:"account"`
	Owner            Owner              `bson:"owner" json:"owner"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
	KeyOwnershipDate time.Time          `bson:"keyOwnershipDate" json:"keyOwnershipDate"`
}

// EntryResponse represents the API response for an entry
type EntryResponse struct {
	Key              string    `json:"key" example:"+5511999999999"`
	KeyType          KeyType   `json:"keyType" example:"PHONE"`
	Account          Account   `json:"account"`
	Owner            Owner     `json:"owner"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	KeyOwnershipDate time.Time `json:"keyOwnershipDate"`
}

// CreateEntryRequest represents the request body for creating an entry
type CreateEntryRequest struct {
	Key       string  `json:"key" validate:"required" example:"+5511999999999"`
	KeyType   KeyType `json:"keyType" validate:"required,oneof=CPF CNPJ EMAIL PHONE EVP" example:"PHONE"`
	Account   Account `json:"account" validate:"required"`
	Owner     Owner   `json:"owner" validate:"required"`
	Reason    Reason  `json:"reason" validate:"required,oneof=USER_REQUESTED RECONCILIATION" example:"USER_REQUESTED"`
	RequestId string  `json:"requestId" validate:"required,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateEntryRequest represents the request body for updating an entry
// Per DICT spec: Only account info, name, and trade name can be updated
// EVP keys cannot be updated
type UpdateEntryRequest struct {
	Key     string         `json:"key" validate:"required" example:"+5511999999999"`
	Account *UpdateAccount `json:"account,omitempty" validate:"omitempty"`
	Owner   *UpdateOwner   `json:"owner,omitempty" validate:"omitempty"`
	Reason  Reason         `json:"reason" validate:"required,oneof=USER_REQUESTED BRANCH_TRANSFER RECONCILIATION RFB_VALIDATION" example:"USER_REQUESTED"`
}

// DeleteEntryRequest represents the request body for deleting an entry
// Per DICT spec: POST /entries/{Key}/delete with request body
type DeleteEntryRequest struct {
	Key         string `json:"key" validate:"required" example:"+5511999999999"`
	Participant string `json:"participant" validate:"required,len=8,numeric" example:"12345678"`
	Reason      Reason `json:"reason" validate:"required,oneof=USER_REQUESTED ACCOUNT_CLOSURE RECONCILIATION FRAUD RFB_VALIDATION" example:"USER_REQUESTED"`
}

// DeleteEntryResponse represents the response for deleting an entry
type DeleteEntryResponse struct {
	Message string `json:"message" example:"Entry deleted successfully"`
	Key     string `json:"key" example:"+5511999999999"`
}

// EntryRepository handles database operations for entries
type EntryRepository struct {
	collection *mongo.Collection
}

// NewEntryRepository creates a new entry repository
func NewEntryRepository(db *db.Mongo) *EntryRepository {
	return &EntryRepository{
		collection: db.Collection("entries"),
	}
}

// EnsureIndexes creates necessary indexes for the entries collection
func (r *EntryRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "owner.taxIdNumber", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// Create creates a new entry in the database
func (r *EntryRepository) Create(ctx context.Context, req *CreateEntryRequest) (*Entry, error) {
	now := time.Now()
	entry := &Entry{
		Key:              req.Key,
		KeyType:          req.KeyType,
		Account:          req.Account,
		Owner:            req.Owner,
		CreatedAt:        now,
		UpdatedAt:        now,
		KeyOwnershipDate: now, // For new entries, ownership date equals creation date
	}

	result, err := r.collection.InsertOne(ctx, entry)
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

// FindByKey finds an entry by its key
func (r *EntryRepository) FindByKey(ctx context.Context, key string) (*Entry, error) {
	var entry Entry
	err := r.collection.FindOne(ctx, bson.M{"key": key}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

// DeleteByKeyAndParticipant deletes an entry by its key and participant, and returns the deleted entry
// This combined operation ensures atomicity and reduces DB calls
func (r *EntryRepository) DeleteByKeyAndParticipant(ctx context.Context, key string, participant string) (*Entry, error) {
	var entry Entry
	filter := bson.M{
		"key":                 key,
		"account.participant": participant,
	}

	err := r.collection.FindOneAndDelete(ctx, filter).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

// UpdateByKey updates an entry by its key
// Only updates the fields that are provided in the request
func (r *EntryRepository) UpdateByKey(ctx context.Context, key string, req *UpdateEntryRequest) (*Entry, error) {
	update := bson.M{
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	setFields := update["$set"].(bson.M)

	if req.Account != nil {
		setFields["account"] = req.Account
	}

	if req.Owner != nil {
		// Only update name and trade name, not taxIdNumber per DICT spec
		if req.Owner.Name != "" {
			setFields["owner.name"] = req.Owner.Name
		}
		if req.Owner.TradeName != "" {
			setFields["owner.tradeName"] = req.Owner.TradeName
		}
	}

	var entry Entry
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"key": key}, update, opts).Decode(&entry)
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
		Key:              e.Key,
		KeyType:          e.KeyType,
		Account:          e.Account,
		Owner:            e.Owner,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
		KeyOwnershipDate: e.KeyOwnershipDate,
	}
}
