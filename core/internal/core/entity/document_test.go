package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDocument_MarkAsPendingProvider(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from DRAFT succeeds", DocumentStatusDraft, false},
		{"from PENDING fails", DocumentStatusPending, true},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
		{"from ERROR fails", DocumentStatusError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsPendingProvider()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusPendingProvider, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsPending(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from DRAFT succeeds", DocumentStatusDraft, false},
		{"from PENDING_PROVIDER succeeds", DocumentStatusPendingProvider, false},
		{"from ERROR succeeds", DocumentStatusError, false},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
		{"from IN_PROGRESS fails", DocumentStatusInProgress, true},
		{"from PENDING fails", DocumentStatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsPending()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusPending, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsInProgress(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from PENDING succeeds", DocumentStatusPending, false},
		{"from IN_PROGRESS succeeds (idempotent)", DocumentStatusInProgress, false},
		{"from DRAFT fails", DocumentStatusDraft, true},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
		{"from ERROR fails", DocumentStatusError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsInProgress()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusInProgress, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsCompleted(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from PENDING succeeds", DocumentStatusPending, false},
		{"from IN_PROGRESS succeeds", DocumentStatusInProgress, false},
		{"from DRAFT fails", DocumentStatusDraft, true},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
		{"from ERROR fails", DocumentStatusError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsCompleted()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusCompleted, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsDeclined(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from PENDING succeeds", DocumentStatusPending, false},
		{"from IN_PROGRESS succeeds", DocumentStatusInProgress, false},
		{"from DRAFT fails", DocumentStatusDraft, true},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsDeclined()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusDeclined, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsVoided(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from DRAFT succeeds", DocumentStatusDraft, false},
		{"from PENDING succeeds", DocumentStatusPending, false},
		{"from IN_PROGRESS succeeds", DocumentStatusInProgress, false},
		{"from ERROR succeeds", DocumentStatusError, false},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
		{"from VOIDED fails", DocumentStatusVoided, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsVoided()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusVoided, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsExpired(t *testing.T) {
	tests := []struct {
		name    string
		status  DocumentStatus
		wantErr bool
	}{
		{"from PENDING succeeds", DocumentStatusPending, false},
		{"from IN_PROGRESS succeeds", DocumentStatusInProgress, false},
		{"from DRAFT fails", DocumentStatusDraft, true},
		{"from COMPLETED fails", DocumentStatusCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			err := d.MarkAsExpired()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, DocumentStatusExpired, d.Status)
			}
		})
	}
}

func TestDocument_MarkAsError(t *testing.T) {
	statuses := []DocumentStatus{
		DocumentStatusDraft, DocumentStatusPending, DocumentStatusInProgress,
		DocumentStatusCompleted, DocumentStatusError,
	}
	for _, s := range statuses {
		t.Run("from "+string(s)+" succeeds", func(t *testing.T) {
			d := &Document{Status: s}
			err := d.MarkAsError()
			assert.NoError(t, err)
			assert.Equal(t, DocumentStatusError, d.Status)
		})
	}
}

func TestDocument_RecoverToAwaitingInput(t *testing.T) {
	t.Run("from ERROR succeeds", func(t *testing.T) {
		d := &Document{Status: DocumentStatusError}
		err := d.RecoverToAwaitingInput()
		assert.NoError(t, err)
		assert.Equal(t, DocumentStatusAwaitingInput, d.Status)
	})

	t.Run("from non-ERROR fails", func(t *testing.T) {
		d := &Document{Status: DocumentStatusPending}
		err := d.RecoverToAwaitingInput()
		assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
	})
}

func TestDocument_RecoverToPendingProvider(t *testing.T) {
	t.Run("from ERROR succeeds", func(t *testing.T) {
		d := &Document{Status: DocumentStatusError}
		err := d.RecoverToPendingProvider()
		assert.NoError(t, err)
		assert.Equal(t, DocumentStatusPendingProvider, d.Status)
	})

	t.Run("from non-ERROR fails", func(t *testing.T) {
		d := &Document{Status: DocumentStatusPending}
		err := d.RecoverToPendingProvider()
		assert.ErrorIs(t, err, ErrInvalidDocumentStatusTransition)
	})
}

func TestDocument_UpdateStatus(t *testing.T) {
	tests := []struct {
		name      string
		newStatus DocumentStatus
		wantErr   bool
	}{
		{"valid PENDING", DocumentStatusPending, false},
		{"valid COMPLETED", DocumentStatusCompleted, false},
		{"valid ERROR", DocumentStatusError, false},
		{"invalid status", DocumentStatus("INVALID"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: DocumentStatusDraft}
			err := d.UpdateStatus(tt.newStatus)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidDocumentStatus)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.newStatus, d.Status)
			}
		})
	}
}

func TestDocument_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   DocumentStatus
		terminal bool
	}{
		{"DRAFT not terminal", DocumentStatusDraft, false},
		{"PENDING_PROVIDER not terminal", DocumentStatusPendingProvider, false},
		{"PENDING not terminal", DocumentStatusPending, false},
		{"IN_PROGRESS not terminal", DocumentStatusInProgress, false},
		{"COMPLETED is terminal", DocumentStatusCompleted, true},
		{"DECLINED is terminal", DocumentStatusDeclined, true},
		{"VOIDED is terminal", DocumentStatusVoided, true},
		{"EXPIRED is terminal", DocumentStatusExpired, true},
		{"ERROR not terminal", DocumentStatusError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Document{Status: tt.status}
			assert.Equal(t, tt.terminal, d.IsTerminal())
		})
	}
}

func TestDocument_ScheduleRetry(t *testing.T) {
	t.Run("increments count and sets exponential backoff", func(t *testing.T) {
		d := &Document{RetryCount: 0}

		ok := d.ScheduleRetry(5)
		assert.True(t, ok)
		assert.Equal(t, 1, d.RetryCount)
		assert.NotNil(t, d.LastRetryAt)
		assert.NotNil(t, d.NextRetryAt)
		// After first retry (count=1): 60s * 2^1 = 120s
		assert.InDelta(t, 120, d.NextRetryAt.Sub(*d.LastRetryAt).Seconds(), 1)
	})

	t.Run("second retry doubles backoff", func(t *testing.T) {
		d := &Document{RetryCount: 1}

		ok := d.ScheduleRetry(5)
		assert.True(t, ok)
		assert.Equal(t, 2, d.RetryCount)
		// After second retry (count=2): 60s * 2^2 = 240s
		assert.InDelta(t, 240, d.NextRetryAt.Sub(*d.LastRetryAt).Seconds(), 1)
	})

	t.Run("caps at 1 hour", func(t *testing.T) {
		d := &Document{RetryCount: 5}

		ok := d.ScheduleRetry(10)
		assert.True(t, ok)
		assert.Equal(t, 6, d.RetryCount)
		// 60s * 2^6 = 3840s > 3600s, so capped at 1h
		assert.InDelta(t, 3600, d.NextRetryAt.Sub(*d.LastRetryAt).Seconds(), 1)
	})

	t.Run("returns false at max retries", func(t *testing.T) {
		d := &Document{RetryCount: 3}

		ok := d.ScheduleRetry(3)
		assert.False(t, ok)
		assert.Equal(t, 3, d.RetryCount) // unchanged
		assert.Nil(t, d.NextRetryAt)
	})
}

func TestDocument_ResetRetry(t *testing.T) {
	now := time.Now().UTC()
	d := &Document{
		RetryCount:  3,
		LastRetryAt: &now,
		NextRetryAt: &now,
	}

	d.ResetRetry()

	assert.Equal(t, 0, d.RetryCount)
	assert.Nil(t, d.LastRetryAt)
	assert.Nil(t, d.NextRetryAt)
	assert.NotNil(t, d.UpdatedAt)
}

func TestDocument_IsExpired(t *testing.T) {
	t.Run("not expired when ExpiresAt is nil", func(t *testing.T) {
		d := &Document{}
		assert.False(t, d.IsExpired())
	})

	t.Run("not expired when ExpiresAt is in the future", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		d := &Document{ExpiresAt: &future}
		assert.False(t, d.IsExpired())
	})

	t.Run("expired when ExpiresAt is in the past", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		d := &Document{ExpiresAt: &past}
		assert.True(t, d.IsExpired())
	})
}

func TestDocument_Validate(t *testing.T) {
	tests := []struct {
		name    string
		doc     Document
		wantErr error
	}{
		{
			name: "valid document",
			doc: Document{
				WorkspaceID:       "ws-1",
				TemplateVersionID: "tv-1",
				DocumentTypeID:    "dt-1",
				Status:            DocumentStatusDraft,
			},
			wantErr: nil,
		},
		{
			name: "missing workspace ID",
			doc: Document{
				TemplateVersionID: "tv-1",
				DocumentTypeID:    "dt-1",
				Status:            DocumentStatusDraft,
			},
			wantErr: ErrRequiredField,
		},
		{
			name: "missing template version ID",
			doc: Document{
				WorkspaceID:    "ws-1",
				DocumentTypeID: "dt-1",
				Status:         DocumentStatusDraft,
			},
			wantErr: ErrRequiredField,
		},
		{
			name: "missing document type ID",
			doc: Document{
				WorkspaceID:       "ws-1",
				TemplateVersionID: "tv-1",
				Status:            DocumentStatusDraft,
			},
			wantErr: ErrRequiredField,
		},
		{
			name: "invalid status",
			doc: Document{
				WorkspaceID:       "ws-1",
				TemplateVersionID: "tv-1",
				DocumentTypeID:    "dt-1",
				Status:            DocumentStatus("INVALID"),
			},
			wantErr: ErrInvalidDocumentStatus,
		},
		{
			name: "title too long",
			doc: func() Document {
				title := strings.Repeat("a", 256)
				return Document{
					WorkspaceID:       "ws-1",
					TemplateVersionID: "tv-1",
					DocumentTypeID:    "dt-1",
					Status:            DocumentStatusDraft,
					Title:             &title,
				}
			}(),
			wantErr: ErrFieldTooLong,
		},
		{
			name: "title at max length is valid",
			doc: func() Document {
				title := strings.Repeat("a", 255)
				return Document{
					WorkspaceID:       "ws-1",
					TemplateVersionID: "tv-1",
					DocumentTypeID:    "dt-1",
					Status:            DocumentStatusDraft,
					Title:             &title,
				}
			}(),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewDocument(t *testing.T) {
	d := NewDocument("ws-1", "tv-1")
	assert.Equal(t, "ws-1", d.WorkspaceID)
	assert.Equal(t, "tv-1", d.TemplateVersionID)
	assert.Equal(t, OperationCreate, d.OperationType)
	assert.Equal(t, DocumentStatusDraft, d.Status)
	assert.False(t, d.CreatedAt.IsZero())
}
