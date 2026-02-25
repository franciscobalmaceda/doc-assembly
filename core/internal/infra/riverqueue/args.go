package riverqueue

import (
	"time"

	"github.com/riverqueue/river"
)

// DocumentCompletedArgs are the arguments for the document completed job.
type DocumentCompletedArgs struct {
	DocumentID string `json:"document_id"`
}

// Kind returns the unique job kind identifier.
func (DocumentCompletedArgs) Kind() string { return "document_completed" }

// InsertOpts returns insert-time options including deduplication.
func (DocumentCompletedArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,
			ByPeriod: 1 * time.Hour,
		},
	}
}
