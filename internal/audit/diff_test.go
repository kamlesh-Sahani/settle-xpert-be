package audit

import (
	"testing"

	auditmodels "lead-crm/backend/internal/audit/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type sampleTeamMember struct {
	ID            string `json:"id"`
	TeamID        string `json:"teamId"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	Status        string `json:"status"`
	Phone         string `json:"phone"`
	PANCard       string `json:"panCard"`
	DateOfJoining string `json:"dateOfJoining"`
}

func TestComputeDiff_Create(t *testing.T) {
	member := sampleTeamMember{
		ID:            "670000000000000000000001",
		TeamID:        "TEAM-00001",
		Name:          "John Doe",
		Email:         "john@example.com",
		Password:      "secret123",
		Status:        "ACTIVE",
		Phone:         "9876543210",
		PANCard:       "ABCDE1234F",
		DateOfJoining: "2026-10-01",
	}

	diffs, count := ComputeDiff(nil, member, auditmodels.ActionCreate)
	// ID and Password should be ignored, leaving 7 fields
	if count != 7 {
		t.Fatalf("expected 7 diffs (excluding id and password), got %d: %+v", count, diffs)
	}

	for _, d := range diffs {
		if d.OlderValue != "NA" {
			t.Errorf("expected OlderValue to be 'NA', got %s", d.OlderValue)
		}
		if d.Field == "Password" {
			t.Errorf("password should be ignored in diff")
		}
	}
}

func TestComputeDiff_PartialUpdate(t *testing.T) {
	oldMember := sampleTeamMember{
		ID:            "670000000000000000000001",
		TeamID:        "TEAM-00001",
		Name:          "John Doe",
		Email:         "john@example.com",
		Status:        "ACTIVE",
		Phone:         "9876543210",
		PANCard:       "ABCDE1234F",
		DateOfJoining: "2026-10-01",
	}

	// Update only status and phone
	updateFields := map[string]any{
		"status":     "SUSPENDED",
		"phone":      "9123456780",
		"updated_at": "2026-10-11",
	}

	diffs, count := ComputeDiff(oldMember, updateFields, auditmodels.ActionUpdate)
	if count != 2 {
		t.Fatalf("expected exactly 2 diffs, got %d: %+v", count, diffs)
	}

	foundStatus := false
	foundPhone := false
	for _, d := range diffs {
		if d.Field == "Status" {
			foundStatus = true
			if d.OlderValue != "ACTIVE" || d.ChangedValue != "SUSPENDED" {
				t.Errorf("Status diff mismatch: older=%s, changed=%s", d.OlderValue, d.ChangedValue)
			}
		}
		if d.Field == "Phone" {
			foundPhone = true
			if d.OlderValue != "9876543210" || d.ChangedValue != "9123456780" {
				t.Errorf("Phone diff mismatch: older=%s, changed=%s", d.OlderValue, d.ChangedValue)
			}
		}
	}

	if !foundStatus || !foundPhone {
		t.Errorf("missing expected diff fields: status=%v, phone=%v", foundStatus, foundPhone)
	}
}

func TestComputeDiff_Delete(t *testing.T) {
	oldMember := map[string]any{
		"id":      "123",
		"name":    "John Doe",
		"team_id": "TEAM-00001",
	}

	diffs, count := ComputeDiff(oldMember, nil, auditmodels.ActionDelete)
	if count != 2 {
		t.Fatalf("expected 2 diffs on delete, got %d", count)
	}

	for _, d := range diffs {
		if d.ChangedValue != "Deleted" {
			t.Errorf("expected ChangedValue to be 'Deleted', got %s", d.ChangedValue)
		}
	}
}

func TestComputeDiff_HumanizeField(t *testing.T) {
	cases := map[string]string{
		"team_id":         "Team ID",
		"date_of_joining": "Date Of Joining",
		"pan_card":        "PAN Card",
		"aadhar_number":   "Aadhar Number",
		"dob":             "DOB",
		"father_name":     "Father Name",
	}

	for input, expected := range cases {
		output := HumanizeField(input)
		if output != expected {
			t.Errorf("HumanizeField(%q) = %q; expected %q", input, output, expected)
		}
	}
}

func TestComputeDiff_PrimitiveD(t *testing.T) {
	oldD := primitive.D{
		{Key: "name", Value: "John"},
		{Key: "phone", Value: "111"},
	}
	newD := primitive.D{
		{Key: "name", Value: "John Updated"},
		{Key: "phone", Value: "222"},
	}

	diffs, count := ComputeDiff(oldD, newD, auditmodels.ActionUpdate)
	if count != 2 {
		t.Fatalf("expected 2 diffs with primitive.D, got %d: %+v", count, diffs)
	}
}
