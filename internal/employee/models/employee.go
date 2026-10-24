package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	EmployeeStatusActive    = "ACTIVE"
	EmployeeStatusSuspended = "SUSPENDED"
	EmployeeStatusDeleted   = "DELETED"
)

type Address struct {
	Line1   string `bson:"line1,omitempty" json:"line1,omitempty"`
	Country string `bson:"country,omitempty" json:"country,omitempty"`
	State   string `bson:"state,omitempty" json:"state,omitempty"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
	Zip     string `bson:"zip,omitempty" json:"zip,omitempty"`
}

type Employee struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	EmpID         string              `bson:"emp_id,omitempty" json:"empId,omitempty"`
	RoleID        *primitive.ObjectID `bson:"role_id,omitempty" json:"roleId,omitempty"`
	Name          string              `bson:"name" json:"name"`
	Email         string              `bson:"email" json:"email"`
	Password      string              `bson:"password" json:"-"`
	Phone         string              `bson:"phone,omitempty" json:"phone,omitempty"`
	Status        string              `bson:"status" json:"status"`
	Logo          string              `bson:"logo,omitempty" json:"logo,omitempty"`
	DOB           string              `bson:"dob,omitempty" json:"dob,omitempty"`
	Gender        string              `bson:"gender,omitempty" json:"gender,omitempty"`
	FatherName    string              `bson:"father_name,omitempty" json:"fatherName,omitempty"`
	MotherName    string              `bson:"mother_name,omitempty" json:"motherName,omitempty"`
	PANCard       string              `bson:"pan_card,omitempty" json:"panCard,omitempty"`
	DateOfJoining string              `bson:"date_of_joining,omitempty" json:"dateOfJoining,omitempty"`
	AadharNumber  string              `bson:"aadhar_number,omitempty" json:"aadharNumber,omitempty"`
	AadharFront   string              `bson:"aadhar_front,omitempty" json:"aadharFront,omitempty"`
	AadharBack    string              `bson:"aadhar_back,omitempty" json:"aadharBack,omitempty"`
	Address       Address             `bson:"address,omitempty" json:"address,omitempty"`
	CreatedAt     time.Time           `bson:"created_at" json:"createdAt"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updatedAt"`
	DeletedAt     *time.Time          `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}
