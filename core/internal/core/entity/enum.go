package entity

// TenantStatus represents the status of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "ACTIVE"
	TenantStatusSuspended TenantStatus = "SUSPENDED"
	TenantStatusArchived  TenantStatus = "ARCHIVED"
)

// IsValid checks if the tenant status is valid.
func (t TenantStatus) IsValid() bool {
	switch t {
	case TenantStatusActive, TenantStatusSuspended, TenantStatusArchived:
		return true
	}
	return false
}

// WorkspaceType represents the type of workspace.
type WorkspaceType string

const (
	WorkspaceTypeSystem WorkspaceType = "SYSTEM"
	WorkspaceTypeClient WorkspaceType = "CLIENT"
)

// IsValid checks if the workspace type is valid.
func (w WorkspaceType) IsValid() bool {
	switch w {
	case WorkspaceTypeSystem, WorkspaceTypeClient:
		return true
	}
	return false
}

// WorkspaceStatus represents the status of a workspace.
type WorkspaceStatus string

const (
	WorkspaceStatusActive    WorkspaceStatus = "ACTIVE"
	WorkspaceStatusSuspended WorkspaceStatus = "SUSPENDED"
	WorkspaceStatusArchived  WorkspaceStatus = "ARCHIVED"
)

// IsValid checks if the workspace status is valid.
func (w WorkspaceStatus) IsValid() bool {
	switch w {
	case WorkspaceStatusActive, WorkspaceStatusSuspended, WorkspaceStatusArchived:
		return true
	}
	return false
}

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusInvited   UserStatus = "INVITED"
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// IsValid checks if the user status is valid.
func (u UserStatus) IsValid() bool {
	switch u {
	case UserStatusInvited, UserStatusActive, UserStatusSuspended:
		return true
	}
	return false
}

// WorkspaceRole represents a user's role within a workspace.
type WorkspaceRole string

const (
	WorkspaceRoleOwner    WorkspaceRole = "OWNER"
	WorkspaceRoleAdmin    WorkspaceRole = "ADMIN"
	WorkspaceRoleEditor   WorkspaceRole = "EDITOR"
	WorkspaceRoleOperator WorkspaceRole = "OPERATOR"
	WorkspaceRoleViewer   WorkspaceRole = "VIEWER"
)

// IsValid checks if the workspace role is valid.
func (w WorkspaceRole) IsValid() bool {
	switch w {
	case WorkspaceRoleOwner, WorkspaceRoleAdmin, WorkspaceRoleEditor, WorkspaceRoleOperator, WorkspaceRoleViewer:
		return true
	}
	return false
}

// Weight returns the numeric weight of the role for permission comparisons.
// Higher weight = more permissions.
func (w WorkspaceRole) Weight() int {
	switch w {
	case WorkspaceRoleOwner:
		return 50
	case WorkspaceRoleAdmin:
		return 40
	case WorkspaceRoleEditor:
		return 30
	case WorkspaceRoleOperator:
		return 20
	case WorkspaceRoleViewer:
		return 10
	default:
		return 0
	}
}

// HasPermission checks if this role has at least the required role's permissions.
func (w WorkspaceRole) HasPermission(required WorkspaceRole) bool {
	return w.Weight() >= required.Weight()
}

// SystemRole represents a user's role at the platform level.
type SystemRole string

const (
	SystemRoleSuperAdmin    SystemRole = "SUPERADMIN"
	SystemRolePlatformAdmin SystemRole = "PLATFORM_ADMIN"
)

// IsValid checks if the system role is valid.
func (s SystemRole) IsValid() bool {
	switch s {
	case SystemRoleSuperAdmin, SystemRolePlatformAdmin:
		return true
	}
	return false
}

// Weight returns the numeric weight of the role for permission comparisons.
// Higher weight = more permissions.
func (s SystemRole) Weight() int {
	switch s {
	case SystemRoleSuperAdmin:
		return 100
	case SystemRolePlatformAdmin:
		return 90
	default:
		return 0
	}
}

// HasPermission checks if this role has at least the required role's permissions.
func (s SystemRole) HasPermission(required SystemRole) bool {
	return s.Weight() >= required.Weight()
}

// TenantRole represents a user's role within a specific tenant.
type TenantRole string

const (
	TenantRoleOwner TenantRole = "TENANT_OWNER"
	TenantRoleAdmin TenantRole = "TENANT_ADMIN"
)

// IsValid checks if the tenant role is valid.
func (t TenantRole) IsValid() bool {
	switch t {
	case TenantRoleOwner, TenantRoleAdmin:
		return true
	}
	return false
}

// Weight returns the numeric weight of the role for permission comparisons.
// Higher weight = more permissions.
func (t TenantRole) Weight() int {
	switch t {
	case TenantRoleOwner:
		return 60
	case TenantRoleAdmin:
		return 55
	default:
		return 0
	}
}

// HasPermission checks if this role has at least the required role's permissions.
func (t TenantRole) HasPermission(required TenantRole) bool {
	return t.Weight() >= required.Weight()
}

// MembershipStatus represents the status of a workspace membership.
type MembershipStatus string

const (
	MembershipStatusPending MembershipStatus = "PENDING"
	MembershipStatusActive  MembershipStatus = "ACTIVE"
)

// IsValid checks if the membership status is valid.
func (m MembershipStatus) IsValid() bool {
	switch m {
	case MembershipStatusPending, MembershipStatusActive:
		return true
	}
	return false
}

// InjectableDataType represents the data type of an injectable variable.
type InjectableDataType string

const (
	InjectableDataTypeText     InjectableDataType = "TEXT"
	InjectableDataTypeNumber   InjectableDataType = "NUMBER"
	InjectableDataTypeDate     InjectableDataType = "DATE"
	InjectableDataTypeCurrency InjectableDataType = "CURRENCY"
	InjectableDataTypeBoolean  InjectableDataType = "BOOLEAN"
	InjectableDataTypeImage    InjectableDataType = "IMAGE"
	InjectableDataTypeTable    InjectableDataType = "TABLE"
	InjectableDataTypeList     InjectableDataType = "LIST"
)

// IsValid checks if the injectable data type is valid.
func (i InjectableDataType) IsValid() bool {
	switch i {
	case InjectableDataTypeText, InjectableDataTypeNumber, InjectableDataTypeDate,
		InjectableDataTypeCurrency, InjectableDataTypeBoolean, InjectableDataTypeImage,
		InjectableDataTypeTable, InjectableDataTypeList:
		return true
	}
	return false
}

// InjectableSourceType indicates whether an injectable's value is calculated
// internally by the system or provided from an external source.
type InjectableSourceType string

const (
	InjectableSourceTypeInternal InjectableSourceType = "INTERNAL"
	InjectableSourceTypeExternal InjectableSourceType = "EXTERNAL"
)

// IsValid checks if the injectable source type is valid.
func (i InjectableSourceType) IsValid() bool {
	switch i {
	case InjectableSourceTypeInternal, InjectableSourceTypeExternal:
		return true
	}
	return false
}

// VersionStatus represents the lifecycle status of a template version.
type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "DRAFT"
	VersionStatusScheduled VersionStatus = "SCHEDULED"
	VersionStatusPublished VersionStatus = "PUBLISHED"
	VersionStatusArchived  VersionStatus = "ARCHIVED"
)

// IsValid checks if the version status is valid.
func (v VersionStatus) IsValid() bool {
	switch v {
	case VersionStatusDraft, VersionStatusScheduled, VersionStatusPublished, VersionStatusArchived:
		return true
	}
	return false
}

// String returns the string representation of the version status.
func (v VersionStatus) String() string {
	return string(v)
}

// CanTransitionTo checks if transition to target status is allowed.
func (v VersionStatus) CanTransitionTo(target VersionStatus) bool {
	switch v {
	case VersionStatusDraft:
		return target == VersionStatusScheduled || target == VersionStatusPublished
	case VersionStatusScheduled:
		return target == VersionStatusDraft || target == VersionStatusPublished
	case VersionStatusPublished:
		return target == VersionStatusArchived
	case VersionStatusArchived:
		return false
	}
	return false
}

// RecipientStatus represents the signing status of a document recipient.
type RecipientStatus string

const (
	// RecipientStatusPending - recipient not yet notified.
	RecipientStatusPending RecipientStatus = "PENDING"
	// RecipientStatusSent - email notification sent to recipient.
	RecipientStatusSent RecipientStatus = "SENT"
	// RecipientStatusDelivered - recipient has viewed/opened the document.
	RecipientStatusDelivered RecipientStatus = "DELIVERED"
	// RecipientStatusSigned - recipient has completed signing.
	RecipientStatusSigned RecipientStatus = "SIGNED"
	// RecipientStatusDeclined - recipient has rejected/declined signing.
	RecipientStatusDeclined RecipientStatus = "DECLINED"

	// Legacy status values for backward compatibility with existing DB data.
	// RecipientStatusWaiting is mapped to RecipientStatusPending.
	RecipientStatusWaiting RecipientStatus = "WAITING"
	// RecipientStatusRejected is mapped to RecipientStatusDeclined.
	RecipientStatusRejected RecipientStatus = "REJECTED"
)

// IsValid checks if the recipient status is valid.
func (r RecipientStatus) IsValid() bool {
	switch r {
	case RecipientStatusPending, RecipientStatusSent, RecipientStatusDelivered,
		RecipientStatusSigned, RecipientStatusDeclined,
		RecipientStatusWaiting, RecipientStatusRejected:
		return true
	}
	return false
}

// Normalize converts legacy status values to their current equivalents.
func (r RecipientStatus) Normalize() RecipientStatus {
	switch r {
	case RecipientStatusWaiting:
		return RecipientStatusPending
	case RecipientStatusRejected:
		return RecipientStatusDeclined
	default:
		return r
	}
}

// String returns the string representation of the recipient status.
func (r RecipientStatus) String() string {
	return string(r)
}

// ProcessType represents how a process identifier is interpreted.
type ProcessType string

const (
	ProcessTypeID            ProcessType = "ID"
	ProcessTypeCanonicalName ProcessType = "CANONICAL_NAME"
)

// DefaultProcess is the base system process.
const DefaultProcess = "DEFAULT"

// DefaultProcessType is the default process type for the base process.
const DefaultProcessType = ProcessTypeCanonicalName

// IsValid checks if the process type is valid.
func (p ProcessType) IsValid() bool {
	switch p {
	case ProcessTypeID, ProcessTypeCanonicalName:
		return true
	}
	return false
}

// String returns the string representation of the process type.
func (p ProcessType) String() string {
	return string(p)
}

// OperationType defines the type of operation on documents.
type OperationType string

const (
	// OperationCreate creates a new document.
	OperationCreate OperationType = "CREATE"
	// OperationRenew renews an existing document.
	OperationRenew OperationType = "RENEW"
	// OperationAmend amends/modifies an existing document.
	OperationAmend OperationType = "AMEND"
	// OperationCancel cancels a document.
	OperationCancel OperationType = "CANCEL"
	// OperationPreview generates a preview only (no signing).
	OperationPreview OperationType = "PREVIEW"
)

// IsValid checks if the operation type is valid.
func (o OperationType) IsValid() bool {
	switch o {
	case OperationCreate, OperationRenew, OperationAmend, OperationCancel, OperationPreview:
		return true
	}
	return false
}

// String returns the string representation of the operation type.
func (o OperationType) String() string {
	return string(o)
}

// DocumentStatus represents the signing workflow status of a document.
type DocumentStatus string

const (
	// DocumentStatusDraft - document created, not yet sent for signing.
	DocumentStatusDraft DocumentStatus = "DRAFT"
	// DocumentStatusAwaitingInput - document has interactive fields awaiting recipient input before signing.
	DocumentStatusAwaitingInput DocumentStatus = "AWAITING_INPUT"
	// DocumentStatusPendingProvider - PDF ready, waiting for worker to upload to provider.
	DocumentStatusPendingProvider DocumentStatus = "PENDING_PROVIDER"
	// DocumentStatusPending - document sent to signing provider, awaiting action.
	DocumentStatusPending DocumentStatus = "PENDING"
	// DocumentStatusInProgress - at least one recipient has interacted with the document.
	DocumentStatusInProgress DocumentStatus = "IN_PROGRESS"
	// DocumentStatusCompleted - all recipients have completed signing.
	DocumentStatusCompleted DocumentStatus = "COMPLETED"
	// DocumentStatusDeclined - a recipient has rejected/declined signing.
	DocumentStatusDeclined DocumentStatus = "DECLINED"
	// DocumentStatusVoided - document was cancelled by the user.
	DocumentStatusVoided DocumentStatus = "VOIDED"
	// DocumentStatusExpired - document signing period has expired.
	DocumentStatusExpired DocumentStatus = "EXPIRED"
	// DocumentStatusError - an error occurred with the signing provider.
	DocumentStatusError DocumentStatus = "ERROR"
)

// IsValid checks if the document status is valid.
func (d DocumentStatus) IsValid() bool {
	switch d {
	case DocumentStatusDraft, DocumentStatusAwaitingInput, DocumentStatusPendingProvider,
		DocumentStatusPending, DocumentStatusInProgress, DocumentStatusCompleted,
		DocumentStatusDeclined, DocumentStatusVoided, DocumentStatusExpired, DocumentStatusError:
		return true
	}
	return false
}

// String returns the string representation of the document status.
func (d DocumentStatus) String() string {
	return string(d)
}

// IsTerminal returns true if the status represents a final state.
func (d DocumentStatus) IsTerminal() bool {
	switch d {
	case DocumentStatusCompleted, DocumentStatusDeclined, DocumentStatusVoided, DocumentStatusExpired:
		return true
	}
	return false
}

// validStatusTransitions defines allowed document status transitions.
var validStatusTransitions = map[DocumentStatus]map[DocumentStatus]bool{
	DocumentStatusDraft: {
		DocumentStatusAwaitingInput:   true, // Document has interactive fields requiring recipient input
		DocumentStatusPendingProvider: true, // Worker flow: PDF saved, waiting for upload
		DocumentStatusPending:         true, // Direct flow: immediate provider upload
		DocumentStatusError:           true,
	},
	DocumentStatusAwaitingInput: {
		DocumentStatusPendingProvider: true, // All interactive fields filled, proceed to signing
		DocumentStatusVoided:          true, // Cancelled by user while awaiting input
	},
	DocumentStatusPendingProvider: {
		DocumentStatusPending: true, // Worker completed upload to provider
		DocumentStatusError:   true, // Error during upload
	},
	DocumentStatusPending: {
		DocumentStatusInProgress: true,
		DocumentStatusCompleted:  true,
		DocumentStatusDeclined:   true,
		DocumentStatusVoided:     true,
		DocumentStatusExpired:    true,
		DocumentStatusError:      true,
	},
	DocumentStatusInProgress: {
		DocumentStatusCompleted: true,
		DocumentStatusDeclined:  true,
		DocumentStatusVoided:    true,
		DocumentStatusExpired:   true,
		DocumentStatusError:     true,
	},
	// Terminal states: Completed, Declined, Voided, Expired - no transitions allowed
	DocumentStatusError: {
		DocumentStatusDraft:   true, // Can retry from error
		DocumentStatusPending: true, // Direct retry: re-upload to provider
	},
}

// CanTransitionTo checks if transition to target status is allowed.
func (d DocumentStatus) CanTransitionTo(target DocumentStatus) bool {
	transitions, ok := validStatusTransitions[d]
	return ok && transitions[target]
}
