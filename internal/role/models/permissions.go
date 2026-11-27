package models

type PermissionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
}

type PermissionGroup struct {
	Module      string           `json:"module"`
	Permissions []PermissionItem `json:"permissions"`
}

var AvailablePermissions = []PermissionGroup{
	{
		Module: "Dashboard",
		Permissions: []PermissionItem{
			{
				ID:          "dashboard:read",
				Title:       "View Dashboard",
				Description: "Can view executive dashboard, pipeline metrics, and business overview",
				Category:    "Dashboard",
			},
		},
	},
	{
		Module: "Leads",
		Permissions: []PermissionItem{
			{
				ID:          "lead:read",
				Title:       "View Leads",
				Description: "Can browse, search, and view inbound lead records",
				Category:    "Leads",
			},
			{
				ID:          "lead:create",
				Title:       "Create Lead",
				Description: "Can create new leads manually or via capture forms",
				Category:    "Leads",
			},
			{
				ID:          "lead:update",
				Title:       "Update Lead",
				Description: "Can update lead notes, qualification status, and contact details",
				Category:    "Leads",
			},
			{
				ID:          "lead:assign",
				Title:       "Assign Lead",
				Description: "Can assign or reassign leads to sales executives",
				Category:    "Leads",
			},
			{
				ID:          "lead:convert",
				Title:       "Convert Lead",
				Description: "Can qualify and convert a lead into a registered customer profile",
				Category:    "Leads",
			},
			{
				ID:          "lead:delete",
				Title:       "Delete Lead",
				Description: "Can remove or archive disqualified leads",
				Category:    "Leads",
			},
		},
	},
	{
		Module: "Agreements",
		Permissions: []PermissionItem{
			{
				ID:          "agreement:read",
				Title:       "View Agreement & PDF",
				Description: "Can access the Agreement page and generate agreement PDFs",
				Category:    "Agreements",
			},
			{
				ID:          "agreement:create",
				Title:       "Create Agreement",
				Description: "Can create and save agreements for leads",
				Category:    "Agreements",
			},
			{
				ID:          "agreement:template_update",
				Title:       "Update Agreement Template",
				Description: "Can update the default agreement body and footer templates",
				Category:    "Agreements",
			},
		},
	},
	{
		Module: "Customers",
		Permissions: []PermissionItem{
			{
				ID:          "customer:read",
				Title:       "View Customer",
				Description: "Can view customer profiles, debt portfolios, and history",
				Category:    "Customers",
			},
			{
				ID:          "customer:create",
				Title:       "Create Customer",
				Description: "Can create new customer profiles directly",
				Category:    "Customers",
			},
			{
				ID:          "customer:update",
				Title:       "Update Customer",
				Description: "Can modify customer personal and employment information",
				Category:    "Customers",
			},
			{
				ID:          "customer:delete",
				Title:       "Delete Customer",
				Description: "Can archive customer profiles",
				Category:    "Customers",
			},
		},
	},
	{
		Module: "Employees",
		Permissions: []PermissionItem{
			{
				ID:          "employee:read",
				Title:       "View Employees",
				Description: "Can view employee directory and staff profiles",
				Category:    "Employees",
			},
			{
				ID:          "employee:create",
				Title:       "Create Employee",
				Description: "Can onboard new staff members with assigned roles",
				Category:    "Employees",
			},
			{
				ID:          "employee:update",
				Title:       "Update Employee",
				Description: "Can edit employee roles, phone, or status",
				Category:    "Employees",
			},
			{
				ID:          "employee:audit",
				Title:       "View Employee Audit",
				Description: "Can view audit history and changes for employees",
				Category:    "Employees",
			},
		},
	},
	{
		Module: "Departments",
		Permissions: []PermissionItem{
			{
				ID:          "department:read",
				Title:       "View Departments",
				Description: "Can view department directory and details",
				Category:    "Departments",
			},
			{
				ID:          "department:create",
				Title:       "Create Department",
				Description: "Can create new company departments",
				Category:    "Departments",
			},
			{
				ID:          "department:update",
				Title:       "Update Department",
				Description: "Can edit department names, codes, or status",
				Category:    "Departments",
			},
			{
				ID:          "department:delete",
				Title:       "Delete Department",
				Description: "Can delete unused company departments",
				Category:    "Departments",
			},
		},
	},
	{
		Module: "Role Management",
		Permissions: []PermissionItem{
			{
				ID:          "role:read",
				Title:       "View Roles",
				Description: "Can view defined RBAC roles and their granted permissions",
				Category:    "Roles",
			},
			{
				ID:          "role:create",
				Title:       "Create Role",
				Description: "Can configure custom roles with selective permissions",
				Category:    "Roles",
			},
			{
				ID:          "role:update",
				Title:       "Update Role",
				Description: "Can edit role names and adjust permission sets",
				Category:    "Roles",
			},
			{
				ID:          "role:delete",
				Title:       "Delete Role",
				Description: "Can delete unused custom roles",
				Category:    "Roles",
			},
		},
	},
	{
		Module: "Meta Integration",
		Permissions: []PermissionItem{
			{
				ID:          "meta:read",
				Title:       "View Meta Integration",
				Description: "Can view Meta Lead Ads settings, credentials status, and integration logs",
				Category:    "Meta",
			},
			{
				ID:          "meta:update",
				Title:       "Manage Meta Integration",
				Description: "Can configure Meta API credentials, test connections, and update webhook settings",
				Category:    "Meta",
			},
		},
	},
	{
		Module: "Profile",
		Permissions: []PermissionItem{
			{
				ID:          "profile:update_name",
				Title:       "Update Name",
				Description: "Can change their own display name",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_phone",
				Title:       "Update Phone Number",
				Description: "Can update their own contact phone number",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_photo",
				Title:       "Update Profile Photo",
				Description: "Can upload or change their profile avatar image",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_dob",
				Title:       "Update Date of Birth",
				Description: "Can update their date of birth",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_gender",
				Title:       "Update Gender",
				Description: "Can update their gender",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_father_name",
				Title:       "Update Father's Name",
				Description: "Can update their father's name",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_mother_name",
				Title:       "Update Mother's Name",
				Description: "Can update their mother's name",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_date_of_joining",
				Title:       "Update Date of Joining",
				Description: "Can update their date of joining",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_pan_card",
				Title:       "Update PAN Card",
				Description: "Can update their PAN card number",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_aadhar_number",
				Title:       "Update Aadhar Number",
				Description: "Can update their Aadhar number",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_aadhar_front",
				Title:       "Update Aadhar Card (Front)",
				Description: "Can upload or update the front side of their Aadhar card",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_aadhar_back",
				Title:       "Update Aadhar Card (Back)",
				Description: "Can upload or update the back side of their Aadhar card",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_address",
				Title:       "Update Address",
				Description: "Can update their residential address details",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_password",
				Title:       "Update Password",
				Description: "Can update their own account password",
				Category:    "Profile",
			},
			{
				ID:          "profile:update_theme",
				Title:       "Update Theme",
				Description: "Can customize the application theme and appearance",
				Category:    "Profile",
			},
		},
	},
}

// GetAllPermissionIDs returns a slice of all granular permission strings in the system
func GetAllPermissionIDs() []string {
	var ids []string
	for _, grp := range AvailablePermissions {
		for _, perm := range grp.Permissions {
			ids = append(ids, perm.ID)
		}
	}
	return ids
}
