package event

// Event reasons for the Elastalert
const (
	// EventReasonCreated describes events where resources were created.
	EventReasonCreated = "Created"
	// EventReasonDeleted describes events where resources were deleted.
	EventReasonDeleted = "Deleted"
	// EventReasonError describes events where resources were an error occurs.
	EventReasonError = "Error"
	// EventReasonSuccess describes events where resources were successfully reconciled.
	EventReasonSuccess = "Success"
)
